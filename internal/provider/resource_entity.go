package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

// entitySeverityChannels reads the notification_channels block into a map
// from severity to its configured channel list, erroring on a duplicate
// severity block (the schema allows repeating the block; only one per
// severity makes sense since the API has one binding set per
// (entity, severity)).
func entitySeverityChannels(d *schema.ResourceData) (map[string][]string, error) {
	raw, ok := d.GetOk("notification_channels")
	if !ok {
		return nil, nil
	}
	return parseEntitySeverityChannels(raw.([]interface{}))
}

// parseEntitySeverityChannels is the raw-value form of entitySeverityChannels,
// for use with d.GetChange (which returns []interface{} directly rather
// than through GetOk).
func parseEntitySeverityChannels(blocks []interface{}) (map[string][]string, error) {
	bySeverity := make(map[string][]string, len(blocks))
	for _, b := range blocks {
		block := b.(map[string]interface{})
		severity := block["severity"].(string)
		if _, dup := bySeverity[severity]; dup {
			return nil, fmt.Errorf("notification_channels has more than one block for severity %q", severity)
		}
		channelsList := block["channels"].([]interface{})
		bySeverity[severity] = toStringSlice(channelsList)
	}
	return bySeverity, nil
}

// reconcileEntityNotificationChannels makes entityID's live notification
// bindings match the configured notification_channels blocks exactly, for
// every severity that appears in EITHER the configured or previously
// applied state. Unlike last9_alert.notification_channels (attach-only —
// see reconcileNotificationChannels), this performs a full reconcile
// including detach: the entity is the sole real owner of a
// (entity, severity, channel) binding — the Last9 backend has no per-alert-
// rule binding at all, every alert rule on this entity at a given severity
// shares these exact rows — so there is no sibling resource whose channels
// could be mistakenly removed by fully reconciling here.
//
// prevBySeverity/wantBySeverity may each be missing a severity the other
// has (e.g. the block for "threat" was removed from config entirely), so
// this iterates the union of both keys rather than just wantBySeverity's.
func reconcileEntityNotificationChannels(apiClient *client.Client, entityID string, prevBySeverity, wantBySeverity map[string][]string) error {
	severities := make(map[string]bool, len(prevBySeverity)+len(wantBySeverity))
	for s := range prevBySeverity {
		severities[s] = true
	}
	for s := range wantBySeverity {
		severities[s] = true
	}

	for severity := range severities {
		if err := reconcileEntityNotificationChannelsAtSeverity(apiClient, entityID, severity, wantBySeverity[severity]); err != nil {
			return err
		}
	}
	return nil
}

// reconcileEntityNotificationChannelsAtSeverity attaches every channel in
// wantChannels not yet bound to entityID at severity, and detaches every
// currently-bound channel at that severity not in wantChannels. See
// reconcileEntityNotificationChannels for why detach is safe here (unlike
// the identically-shaped attach-only reconcileNotificationChannels for
// last9_alert).
func reconcileEntityNotificationChannelsAtSeverity(apiClient *client.Client, entityID, severity string, wantChannels []string) error {
	live, err := apiClient.GetEntityNotificationBindings(entityID)
	if err != nil {
		return fmt.Errorf("failed to read existing notification bindings for entity %s: %w", entityID, err)
	}

	liveByName := make(map[string]client.NotificationDestination, len(live))
	for _, b := range live {
		if b.Severity == severity {
			liveByName[b.Name] = b
		}
	}

	wantResolved, err := resolveChannels(apiClient, wantChannels)
	if err != nil {
		return err
	}
	want := make(map[string]bool, len(wantResolved))
	for _, r := range wantResolved {
		want[r.name] = true
	}

	for _, r := range wantResolved {
		if _, ok := liveByName[r.name]; ok {
			continue // already bound at this severity
		}
		if _, err := apiClient.AttachNotificationSettings(r.id, entityID, severity); err != nil {
			return fmt.Errorf("failed to attach notification channel %q (id %d) to entity %s: %w", r.value, r.id, entityID, err)
		}
	}

	for name, binding := range liveByName {
		if want[name] {
			continue // still wanted
		}
		if err := apiClient.DetachNotificationSettings(binding.ID); err != nil {
			return fmt.Errorf("failed to detach notification channel %q (binding id %d) from entity %s: %w", name, binding.ID, entityID, err)
		}
	}

	return nil
}

// setEntityNotificationChannels writes notification_channels from the real
// live bindings for entityID, one block for every severity the current
// config already declares a block for — never for a severity this
// resource's config doesn't mention at all.
//
// It's tempting to think "the entity IS the sole owner of every binding on
// it, so reporting everything live is just accurate drift detection" — but
// that reasoning breaks the moment a customer manages a severity's
// channels via last9_alert.notification_channels (still supported,
// attach-only) and simply never adds a notification_channels block for
// that severity on last9_entity. Iterating every severity with a live
// binding would then adopt that alert-managed channel into this resource's
// state on the very next Read, and the following plan/apply would delete
// it as "no longer configured" — destroying paging that was never this
// resource's to touch. So: only severities present in configured are ever
// examined or emitted, including one with an explicit empty channels list
// (an empty block is a real "I want zero channels here" declaration and
// must survive Read unchanged, not disappear for having nothing live to
// report). A severity with real bindings that isn't in configured is left
// entirely alone — visible only via last9_alert or the Last9 UI, exactly
// as it was before this resource existed. Adopting an existing severity's
// bindings into last9_entity happens explicitly, via `terraform import`
// (see resourceEntityImportState) or by adding the block to config, never
// implicitly via Read.
//
// A channel already present in the current config for its severity keeps
// its configured spelling (ID vs. name) rather than always being rewritten
// to the API's display name, avoiding perpetual plan diffs from that
// alone.
func setEntityNotificationChannels(d *schema.ResourceData, apiClient *client.Client, entityID string) diag.Diagnostics {
	configured, err := entitySeverityChannels(d)
	if err != nil {
		return diag.FromErr(err)
	}
	if len(configured) == 0 {
		return nil // this resource doesn't manage any severity's channels
	}

	bindings, err := apiClient.GetEntityNotificationBindings(entityID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read notification bindings for entity %s: %w", entityID, err))
	}
	liveNamesBySeverity := make(map[string]map[string]bool)
	for _, b := range bindings {
		if liveNamesBySeverity[b.Severity] == nil {
			liveNamesBySeverity[b.Severity] = make(map[string]bool)
		}
		liveNamesBySeverity[b.Severity][b.Name] = true
	}

	blocks := make([]interface{}, 0, len(configured))
	for severity, configuredChannels := range configured {
		liveNames := liveNamesBySeverity[severity]

		channels := make([]string, 0, len(configuredChannels))
		for _, value := range configuredChannels {
			dest, err := resolveNotificationChannel(apiClient, value)
			if err != nil {
				if errors.Is(err, client.ErrNotificationChannelNotFound) {
					continue // genuinely deleted from the org -- drop it, surfaces as drift
				}
				// A transport error, non-2xx response, or malformed JSON from
				// the lookup is NOT the same as "channel doesn't exist" —
				// propagate it rather than silently dropping the value,
				// which would otherwise report a transient API failure as a
				// real configuration change.
				return diag.FromErr(fmt.Errorf("failed to resolve notification channel %q: %w", value, err))
			}
			if liveNames[dest.Name] {
				channels = append(channels, value)
			}
		}

		blocks = append(blocks, map[string]interface{}{
			"severity": severity,
			"channels": channels,
		})
	}

	return diag.FromErr(d.Set("notification_channels", blocks))
}

func resourceEntity() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceEntityCreate,
		ReadContext:   resourceEntityRead,
		UpdateContext: resourceEntityUpdate,
		DeleteContext: resourceEntityDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceEntityImportState,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Entity name",
			},
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Entity type (e.g., service, service_alert_manager)",
			},
			"external_ref": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Unique slug identifier for the entity",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Entity description",
			},
			"data_source": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Data source name",
			},
			"data_source_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Data source ID (resolved from data_source name)",
			},
			"namespace": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Entity namespace",
			},
			"team": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Owning team",
			},
			"tier": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Entity tier (e.g., critical, high, medium, low)",
			},
			"workspace": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Workspace",
			},
			"tags": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Array of tags",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Key-value pairs for group labels inherited across indicators",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"entity_class": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Entity classification (e.g., alert-manager)",
			},
			"ui_readonly": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Disable UI edits to avoid configuration conflicts with IaC",
			},
			"adhoc_filter": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Common rule filters applied across all indicators",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"data_source": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Data source for adhoc filter",
						},
						"labels": {
							Type:        schema.TypeMap,
							Required:    true,
							Description: "PromQL label filter conditions",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"indicators": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Array of indicators (metrics)",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Indicator name",
						},
						"query": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "PromQL query",
						},
						"unit": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Measurement unit",
						},
					},
				},
			},
			"links": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Array of related links",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Link name/title",
						},
						"url": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Link URL",
						},
					},
				},
			},
			"notification_channels": {
				Type:     schema.TypeList,
				Optional: true,
				Description: "Notification channel bindings for this entity (alert group), one " +
					"block per severity. This is the authoritative place to manage channel " +
					"bindings: the underlying binding is keyed by (entity, severity, channel) " +
					"with no per-alert-rule ownership at all — every alert rule on this " +
					"entity at a given severity shares the exact same bindings, matching " +
					"how the Last9 UI itself only lets you edit notification channels at " +
					"the alert-group level (\"Inherited from the alert group\"). Because " +
					"this resource is the entity, it's safe to fully reconcile (attach AND " +
					"detach) here, unlike last9_alert.notification_channels.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"severity": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Severity this channel list applies to (breach or threat).",
							ValidateFunc: validation.StringInSlice([]string{"breach", "threat"}, false),
						},
						"channels": {
							Type:        schema.TypeList,
							Required:    true,
							Description: "Notification channel IDs or names to bind at this severity.",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"renotify_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "When false, only the first firing notification and the resolved notification are sent (notify-once). When true, notifications repeat per renotify_interval_seconds. Omit to inherit the tenant default.",
			},
			"renotify_interval_seconds": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Seconds between repeat notifications while an alert stays firing. Must be a positive integer (> 0). Ignored when renotify_enabled is false. Omit to inherit the tenant default.",
				ValidateFunc: validation.IntAtLeast(1),
			},
			"renotify_occurrences": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Maximum number of repeat notifications per firing episode. Use -1 for unlimited. Must be -1 or a positive integer (>= 1). Omit to inherit the tenant default.",
				ValidateFunc: func(v interface{}, k string) (warns []string, errs []error) {
					val := v.(int)
					if val != -1 && val < 1 {
						errs = append(errs, fmt.Errorf("%q must be -1 (unlimited) or >= 1, got: %d", k, val))
					}
					return
				},
			},
		},
	}
}

func resourceEntityCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	req := &client.EntityCreateRequest{
		Name:        d.Get("name").(string),
		Type:        d.Get("type").(string),
		ExternalRef: d.Get("external_ref").(string),
		Description: d.Get("description").(string),
		UIReadonly:  d.Get("ui_readonly").(bool),
	}

	// Optional string fields
	if v, ok := d.GetOk("data_source"); ok {
		req.DataSource = v.(string)
	}
	if v, ok := d.GetOk("data_source_id"); ok {
		req.DataSourceID = v.(string)
	}
	if v, ok := d.GetOk("namespace"); ok {
		req.Namespace = v.(string)
	}
	if v, ok := d.GetOk("team"); ok {
		req.Team = v.(string)
	}
	if v, ok := d.GetOk("tier"); ok {
		req.Tier = v.(string)
	}
	if v, ok := d.GetOk("workspace"); ok {
		req.Workspace = v.(string)
	}
	if v, ok := d.GetOk("entity_class"); ok {
		req.EntityClass = v.(string)
	}

	// Tags
	if v, ok := d.GetOk("tags"); ok {
		tagsList := v.([]interface{})
		tags := make([]string, len(tagsList))
		for i, tag := range tagsList {
			tags[i] = tag.(string)
		}
		req.Tags = tags
	}

	// Labels
	if v, ok := d.GetOk("labels"); ok {
		labelsMap := v.(map[string]interface{})
		labels := make(map[string]string)
		for k, val := range labelsMap {
			labels[k] = val.(string)
		}
		req.Labels = labels
	}

	// Adhoc filter
	if v, ok := d.GetOk("adhoc_filter"); ok {
		filterList := v.([]interface{})
		if len(filterList) > 0 {
			filterMap := filterList[0].(map[string]interface{})
			adhocFilter := &client.AdhocFilter{
				DataSource: filterMap["data_source"].(string),
				Labels:     make(map[string]string),
			}
			if labelsInterface, ok := filterMap["labels"].(map[string]interface{}); ok {
				for k, val := range labelsInterface {
					adhocFilter.Labels[k] = val.(string)
				}
			}
			req.AdhocFilter = adhocFilter
		}
	}

	// Indicators
	if v, ok := d.GetOk("indicators"); ok {
		indicatorsList := v.([]interface{})
		indicators := make([]client.Indicator, len(indicatorsList))
		for i, ind := range indicatorsList {
			indMap := ind.(map[string]interface{})
			indicators[i] = client.Indicator{
				Name:  indMap["name"].(string),
				Query: indMap["query"].(string),
			}
			if unit, ok := indMap["unit"].(string); ok {
				indicators[i].Unit = unit
			}
		}
		req.Indicators = indicators
	}

	// Links
	if v, ok := d.GetOk("links"); ok {
		linksList := v.([]interface{})
		links := make([]client.EntityLink, len(linksList))
		for i, lnk := range linksList {
			lnkMap := lnk.(map[string]interface{})
			links[i] = client.EntityLink{
				Name: lnkMap["name"].(string),
				URL:  lnkMap["url"].(string),
			}
		}
		req.Links = links
	}

	entity, err := apiClient.CreateEntity(req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create entity: %w", err))
	}

	d.SetId(entity.ID)

	// Metadata (tags, labels, team, links, adhoc_filter) must be set via separate endpoint
	// The POST /entities API doesn't accept these fields
	hasMetadata := false
	metadataReq := &client.EntityMetadataUpdateRequest{}

	// Team (required for metadata update API, use "default" if not specified)
	// Only mark hasMetadata=true if user explicitly specified team
	if v, ok := d.GetOk("team"); ok {
		metadataReq.Team = v.(string)
		hasMetadata = true
	} else {
		metadataReq.Team = "default" // Required by API but doesn't count as user-specified metadata
	}

	// Renotify fields — use GetRawConfig to detect explicit false for the bool field
	rawCfg := d.GetRawConfig()
	renotifyEnabledRaw := rawCfg.GetAttr("renotify_enabled")
	renotifyIntervalRaw := rawCfg.GetAttr("renotify_interval_seconds")
	renotifyOccurrencesRaw := rawCfg.GetAttr("renotify_occurrences")
	if !renotifyEnabledRaw.IsNull() {
		v := d.Get("renotify_enabled").(bool)
		metadataReq.RenotifyEnabled = &v
		hasMetadata = true
	}
	if !renotifyIntervalRaw.IsNull() {
		v := d.Get("renotify_interval_seconds").(int)
		metadataReq.RenotifyIntervalSeconds = &v
		hasMetadata = true
	}
	if !renotifyOccurrencesRaw.IsNull() {
		v := d.Get("renotify_occurrences").(int)
		metadataReq.RenotifyOccurrences = &v
		hasMetadata = true
	}

	// Tags
	if v, ok := d.GetOk("tags"); ok {
		tagsList := v.([]interface{})
		tags := make([]string, len(tagsList))
		for i, tag := range tagsList {
			tags[i] = tag.(string)
		}
		metadataReq.Tags = tags
		hasMetadata = true
	}

	// Labels
	if v, ok := d.GetOk("labels"); ok {
		labelsMap := v.(map[string]interface{})
		labels := make(map[string]string)
		for k, val := range labelsMap {
			labels[k] = val.(string)
		}
		metadataReq.Labels = labels
		hasMetadata = true
	}

	// Links
	if v, ok := d.GetOk("links"); ok {
		linksList := v.([]interface{})
		links := make([]client.EntityLink, len(linksList))
		for i, lnk := range linksList {
			lnkMap := lnk.(map[string]interface{})
			links[i] = client.EntityLink{
				Name: lnkMap["name"].(string),
				URL:  lnkMap["url"].(string),
			}
		}
		metadataReq.Links = links
		hasMetadata = true
	}

	// Adhoc filter
	if v, ok := d.GetOk("adhoc_filter"); ok {
		filterList := v.([]interface{})
		if len(filterList) > 0 {
			filterMap := filterList[0].(map[string]interface{})
			adhocFilter := &client.AdhocFilter{
				DataSource: filterMap["data_source"].(string),
				Labels:     make(map[string]string),
			}
			if labelsInterface, ok := filterMap["labels"].(map[string]interface{}); ok {
				for k, val := range labelsInterface {
					adhocFilter.Labels[k] = val.(string)
				}
			}
			metadataReq.AdhocFilter = adhocFilter
			hasMetadata = true
		}
	}

	// Update metadata if any metadata fields were specified
	if hasMetadata {
		err := apiClient.UpdateEntityMetadata(entity.ID, metadataReq)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to set entity metadata: %w", err))
		}
	}

	// Notification channel bindings live in a separate API
	// (/notification_settings) from entity create/update — see
	// reconcileEntityNotificationChannels. Nothing was live before a
	// brand-new entity, so prevBySeverity is nil.
	wantBySeverity, err := entitySeverityChannels(d)
	if err != nil {
		return diag.FromErr(err)
	}
	if len(wantBySeverity) > 0 {
		if err := reconcileEntityNotificationChannels(apiClient, entity.ID, nil, wantBySeverity); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceEntityRead(ctx, d, m)
}

func resourceEntityRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	entity, err := apiClient.GetEntity(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read entity: %w", err))
	}

	d.Set("name", entity.Name)
	d.Set("type", entity.Type)
	d.Set("external_ref", entity.ExternalRef)
	d.Set("description", entity.Description)
	d.Set("data_source", entity.DataSource)
	d.Set("data_source_id", entity.DataSourceID)
	d.Set("namespace", entity.Namespace)
	d.Set("tier", entity.Tier)
	d.Set("workspace", entity.Workspace)
	d.Set("entity_class", entity.EntityClass)
	d.Set("ui_readonly", entity.UIReadonly)

	if diags := setEntityNotificationChannels(d, apiClient, entity.ID); diags.HasError() {
		return diags
	}

	// Metadata fields (tags, labels, team, links, adhoc_filter) are returned
	// nested in the 'metadata' object from the API response
	if entity.Metadata != nil {
		// Only set team if it's not "default" (we use "default" internally when user doesn't specify)
		if entity.Metadata.Team != "" && entity.Metadata.Team != "default" {
			d.Set("team", entity.Metadata.Team)
		}
		d.Set("tags", entity.Metadata.Tags)
		d.Set("labels", entity.Metadata.Labels)

		// Set adhoc filter from metadata
		if entity.Metadata.AdhocFilter != nil {
			adhocFilter := []interface{}{
				map[string]interface{}{
					"data_source": entity.Metadata.AdhocFilter.DataSource,
					"labels":      entity.Metadata.AdhocFilter.Labels,
				},
			}
			d.Set("adhoc_filter", adhocFilter)
		}

		// Set links from metadata
		if len(entity.Metadata.Links) > 0 {
			links := make([]interface{}, len(entity.Metadata.Links))
			for i, link := range entity.Metadata.Links {
				links[i] = map[string]interface{}{
					"name": link.Name,
					"url":  link.URL,
				}
			}
			d.Set("links", links)
		}

		// Renotify fields — only write to state when the API returns a value.
		// For the bool field: leave absent (not set) when nil so Terraform can cleanly
		// compare absent-state against absent-config without a perpetual diff.
		// For int fields: write 0 when nil so stale non-zero values don't linger in state
		// after a clear_override; a config with the field absent also evaluates to 0.
		if entity.Metadata.RenotifyEnabled != nil {
			d.Set("renotify_enabled", *entity.Metadata.RenotifyEnabled)
		}
		if entity.Metadata.RenotifyIntervalSeconds != nil {
			d.Set("renotify_interval_seconds", *entity.Metadata.RenotifyIntervalSeconds)
		} else {
			d.Set("renotify_interval_seconds", 0)
		}
		if entity.Metadata.RenotifyOccurrences != nil {
			d.Set("renotify_occurrences", *entity.Metadata.RenotifyOccurrences)
		} else {
			d.Set("renotify_occurrences", 0)
		}
	} else {
		// Fallback to top-level fields if metadata is not present
		d.Set("team", entity.Team)
		d.Set("tags", entity.Tags)
		d.Set("labels", entity.Labels)

		// Set adhoc filter
		if entity.AdhocFilter != nil {
			adhocFilter := []interface{}{
				map[string]interface{}{
					"data_source": entity.AdhocFilter.DataSource,
					"labels":      entity.AdhocFilter.Labels,
				},
			}
			d.Set("adhoc_filter", adhocFilter)
		}

		// Set links
		if len(entity.Links) > 0 {
			links := make([]interface{}, len(entity.Links))
			for i, link := range entity.Links {
				links[i] = map[string]interface{}{
					"name": link.Name,
					"url":  link.URL,
				}
			}
			d.Set("links", links)
		}
	}

	// Set indicators
	if len(entity.Indicators) > 0 {
		indicators := make([]interface{}, len(entity.Indicators))
		for i, ind := range entity.Indicators {
			indicators[i] = map[string]interface{}{
				"name":  ind.Name,
				"query": ind.Query,
				"unit":  ind.Unit,
			}
		}
		d.Set("indicators", indicators)
	}

	return nil
}

func resourceEntityUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	// Check if core entity fields changed (name, type, description, data_source_id, etc.)
	// These are updated via PUT /entities/{id}
	coreFieldsChanged := d.HasChange("name") || d.HasChange("type") || d.HasChange("description") ||
		d.HasChange("data_source_id") || d.HasChange("namespace") || d.HasChange("tier") ||
		d.HasChange("workspace") || d.HasChange("ui_readonly")

	if coreFieldsChanged {
		// PUT requires all mandatory fields: name, type, data_source_id
		name := d.Get("name").(string)
		entityType := d.Get("type").(string)
		dataSourceID := d.Get("data_source_id").(string)

		req := &client.EntityUpdateRequest{
			Name: &name,
			Type: &entityType,
		}

		// data_source_id is mandatory for update
		if dataSourceID != "" {
			req.DataSourceID = &dataSourceID
		}

		// Include optional fields
		if v, ok := d.GetOk("description"); ok {
			desc := v.(string)
			req.Description = &desc
		}
		if v, ok := d.GetOk("namespace"); ok {
			ns := v.(string)
			req.Namespace = &ns
		}
		if v, ok := d.GetOk("tier"); ok {
			tier := v.(string)
			req.Tier = &tier
		}
		if v, ok := d.GetOk("workspace"); ok {
			ws := v.(string)
			req.Workspace = &ws
		}
		uiReadonly := d.Get("ui_readonly").(bool)
		req.UIReadonly = &uiReadonly

		_, err := apiClient.UpdateEntity(d.Id(), req)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to update entity: %w", err))
		}
	}

	// Check if metadata fields changed (tags, labels, team, links, adhoc_filter, renotify)
	// These are updated via PUT /entities/{id}/metadata
	metadataFieldsChanged := d.HasChange("tags") || d.HasChange("labels") || d.HasChange("team") ||
		d.HasChange("links") || d.HasChange("adhoc_filter") ||
		d.HasChange("renotify_enabled") || d.HasChange("renotify_interval_seconds") || d.HasChange("renotify_occurrences")

	if metadataFieldsChanged {
		// Team is required for metadata update (per OpenAPI spec)
		team := d.Get("team").(string)
		if team == "" {
			team = "default" // Use default if not specified
		}

		metadataReq := &client.EntityMetadataUpdateRequest{
			Team: team,
		}

		// Tags
		if v, ok := d.GetOk("tags"); ok {
			tagsList := v.([]interface{})
			tags := make([]string, len(tagsList))
			for i, tag := range tagsList {
				tags[i] = tag.(string)
			}
			metadataReq.Tags = tags
		}

		// Labels
		if v, ok := d.GetOk("labels"); ok {
			labelsMap := v.(map[string]interface{})
			labels := make(map[string]string)
			for k, val := range labelsMap {
				labels[k] = val.(string)
			}
			metadataReq.Labels = labels
		}

		// Links
		if v, ok := d.GetOk("links"); ok {
			linksList := v.([]interface{})
			links := make([]client.EntityLink, len(linksList))
			for i, lnk := range linksList {
				lnkMap := lnk.(map[string]interface{})
				links[i] = client.EntityLink{
					Name: lnkMap["name"].(string),
					URL:  lnkMap["url"].(string),
				}
			}
			metadataReq.Links = links
		}

		// Adhoc filter
		if v, ok := d.GetOk("adhoc_filter"); ok {
			filterList := v.([]interface{})
			if len(filterList) > 0 {
				filterMap := filterList[0].(map[string]interface{})
				adhocFilter := &client.AdhocFilter{
					DataSource: filterMap["data_source"].(string),
					Labels:     make(map[string]string),
				}
				if labelsInterface, ok := filterMap["labels"].(map[string]interface{}); ok {
					for k, val := range labelsInterface {
						adhocFilter.Labels[k] = val.(string)
					}
				}
				metadataReq.AdhocFilter = adhocFilter
			}
		}

		// Renotify fields — detect which are explicitly set in the new config
		rawCfgU := d.GetRawConfig()
		renotifyEnabledRawU := rawCfgU.GetAttr("renotify_enabled")
		renotifyIntervalRawU := rawCfgU.GetAttr("renotify_interval_seconds")
		renotifyOccurrencesRawU := rawCfgU.GetAttr("renotify_occurrences")
		hasEnabledU := !renotifyEnabledRawU.IsNull()
		hasIntervalU := !renotifyIntervalRawU.IsNull()
		hasOccurrencesU := !renotifyOccurrencesRawU.IsNull()

		renotifyChanged := d.HasChange("renotify_enabled") || d.HasChange("renotify_interval_seconds") || d.HasChange("renotify_occurrences")
		if !hasEnabledU && !hasIntervalU && !hasOccurrencesU && renotifyChanged {
			// All three fields removed from config and at least one was previously set —
			// send clear_override to restore tenant defaults. Scoping to renotifyChanged
			// prevents clearing overrides on unrelated metadata updates (e.g., tag changes).
			metadataReq.RenotifyClearOverride = true
		} else {
			if hasEnabledU {
				v := d.Get("renotify_enabled").(bool)
				metadataReq.RenotifyEnabled = &v
			}
			if hasIntervalU {
				v := d.Get("renotify_interval_seconds").(int)
				metadataReq.RenotifyIntervalSeconds = &v
			}
			if hasOccurrencesU {
				v := d.Get("renotify_occurrences").(int)
				metadataReq.RenotifyOccurrences = &v
			}
		}

		err := apiClient.UpdateEntityMetadata(d.Id(), metadataReq)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to update entity metadata: %w", err))
		}
	}

	if d.HasChange("notification_channels") {
		oldRaw, newRaw := d.GetChange("notification_channels")
		prevBySeverity, err := parseEntitySeverityChannels(oldRaw.([]interface{}))
		if err != nil {
			return diag.FromErr(err)
		}
		wantBySeverity, err := parseEntitySeverityChannels(newRaw.([]interface{}))
		if err != nil {
			return diag.FromErr(err)
		}
		if err := reconcileEntityNotificationChannels(apiClient, d.Id(), prevBySeverity, wantBySeverity); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceEntityRead(ctx, d, m)
}

func resourceEntityDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	err := apiClient.DeleteEntity(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete entity: %w", err))
	}

	d.SetId("")
	return nil
}

// resourceEntityImportState seeds notification_channels from every severity
// that currently has at least one live binding, before delegating the rest
// of the import to a normal Read. setEntityNotificationChannels only ever
// examines a severity already present in config (see its docstring) —
// necessary so an ordinary Read never adopts a severity managed by
// last9_alert instead, but that same rule means a fresh import (no prior
// config at all) needs an explicit seed here, or every existing binding
// would be invisible to this resource forever. Import is the one point
// where adopting "everything currently live" as this resource's declared
// ownership is exactly what the user is asking for.
func resourceEntityImportState(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	apiClient := m.(*client.Client)

	bindings, err := apiClient.GetEntityNotificationBindings(d.Id())
	if err != nil {
		return nil, fmt.Errorf("failed to read notification bindings for import: %w", err)
	}
	namesBySeverity := make(map[string][]string)
	for _, b := range bindings {
		namesBySeverity[b.Severity] = append(namesBySeverity[b.Severity], b.Name)
	}
	if len(namesBySeverity) > 0 {
		blocks := make([]interface{}, 0, len(namesBySeverity))
		for severity, names := range namesBySeverity {
			channels := make([]interface{}, len(names))
			for i, n := range names {
				channels[i] = n
			}
			blocks = append(blocks, map[string]interface{}{"severity": severity, "channels": channels})
		}
		d.Set("notification_channels", blocks)
	}

	return []*schema.ResourceData{d}, nil
}
