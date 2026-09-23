package provider

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

// toStringSlice converts a schema.TypeList's raw []interface{} value (as
// returned by d.Get/d.GetChange for a list-of-string attribute) to []string.
func toStringSlice(raw []interface{}) []string {
	out := make([]string, len(raw))
	for i, v := range raw {
		out[i] = v.(string)
	}
	return out
}

// resolveNotificationChannel looks up a notification_channels entry. The API
// (and examples/entity-with-alerts/main.tf, which passes
// data.last9_notification_destination.*.id) accepts either the channel's
// numeric ID or its display name, so a value that parses as an integer is
// resolved by ID and everything else by name.
func resolveNotificationChannel(apiClient *client.Client, value string) (*client.NotificationDestination, error) {
	if id, err := strconv.Atoi(value); err == nil {
		return apiClient.GetNotificationDestination(id)
	}
	return apiClient.FindNotificationDestinationByName(value)
}

// resolvedChannel is a notification_channels entry resolved to the master
// channel's canonical name and attach ID.
type resolvedChannel struct {
	value string // the original config string ("1" or "Channel A")
	name  string // canonical display name, used to match live binding rows
	id    int    // master channel ID, used for AttachNotificationSettings
}

// resolveChannels resolves each notification_channels entry (an ID or a
// name) to the underlying channel's canonical name and ID. Matching by name
// rather than ID is required on the live-binding side: a per-entity binding
// row (as returned by GetEntityNotificationBindings) carries only its own
// row ID and the channel's Name, not the master channel's ID (attach and
// detach are genuinely different ID spaces — see DetachNotificationSettings),
// so Name is the only field that reliably joins a binding row back to a
// channel resolved from config. Resolving here still means "1" and "Channel
// A" compare equal when they name the same channel, since both resolve to
// the same canonical Name.
//
// An entry that fails to resolve is skipped rather than erroring when
// skipUnresolvable is true. This is for prevChannels: a previously-owned
// channel can have been deleted from the org entirely between applies, and
// failing the whole reconcile over a now-nonexistent channel would also
// block legitimate attach/detach work for the current wantChannels list —
// there is nothing to detach for a channel that no longer exists anyway.
// wantChannels must NOT skip: a channel the user's config currently asks
// for that fails to resolve is a real configuration error and should fail
// the apply.
func resolveChannels(apiClient *client.Client, values []string, skipUnresolvable bool) ([]resolvedChannel, error) {
	resolved := make([]resolvedChannel, 0, len(values))
	for _, v := range values {
		dest, err := resolveNotificationChannel(apiClient, v)
		if err != nil {
			if skipUnresolvable {
				continue
			}
			return nil, fmt.Errorf("failed to resolve notification channel %q: %w", v, err)
		}
		resolved = append(resolved, resolvedChannel{value: v, name: dest.Name, id: dest.ID})
	}
	return resolved, nil
}

// reconcileNotificationChannels makes entityID's live notification bindings
// at the given severity match wantChannels exactly, without disturbing
// bindings at other severities or bindings this alert never owned. The
// alert-rules Create/Update API has no notification_channels field of its
// own — a channel is bound to an entity via dedicated attach/detach calls,
// so simply including notification_channels in the alert-rule request body
// is silently ignored server-side and creates no live binding. This
// function makes the calls that actually create/remove one:
//   - any wanted channel not currently bound (at this severity) gets attached
//   - any channel this alert previously attached (per prevChannels) that is
//     no longer wanted gets detached — this is what makes a channel
//     *removed* from config actually stop notifying, instead of the stale
//     binding silently surviving
//   - a channel already correctly bound is left untouched, regardless of
//     whether config spells it as a name or a numeric ID
//
// The binding API has no concept of which alert owns a binding — a row is
// keyed only by (entity, severity, channel). Scoping detachment to
// prevChannels (this resource's own prior notification_channels — see the
// ownership-tracking note on resourceAlertRead for where that value must
// come from) rather than "every live binding not currently wanted" is what
// keeps one alert's reconcile from deleting a sibling alert's or an
// externally-managed binding that happens to share the same entity and
// severity.
//
// Matching against live bindings is done by resolved canonical name, not
// the raw config string or the master channel ID: a binding row (from
// GetEntityNotificationBindings) carries only its own row ID and the
// channel's Name — never the master channel ID that AttachNotificationSettings
// takes — so Name is the only field that reliably joins a live row back to
// a channel resolved from config. This still means "1" and "Channel A"
// compare equal when they name the same channel, since resolveChannels
// maps both to that channel's canonical Name.
//
// entityID must already exist (this is called after alert create/update).
func reconcileNotificationChannels(apiClient *client.Client, entityID, severity string, prevChannels, wantChannels []string) error {
	live, err := apiClient.GetEntityNotificationBindings(entityID)
	if err != nil {
		return fmt.Errorf("failed to read existing notification bindings for entity %s: %w", entityID, err)
	}

	// Scope to this severity only — GetEntityNotificationBindings returns
	// every severity's rows for the entity, and a breach reconcile must not
	// see or touch threat bindings (and vice versa).
	liveByName := make(map[string]client.NotificationDestination, len(live))
	for _, b := range live {
		if b.Severity == severity {
			liveByName[b.Name] = b
		}
	}

	wantResolved, err := resolveChannels(apiClient, wantChannels, false)
	if err != nil {
		return err
	}
	// prevChannels tolerates unresolvable entries -- see resolveChannels.
	prevResolved, _ := resolveChannels(apiClient, prevChannels, true)

	want := make(map[string]bool, len(wantResolved))
	for _, r := range wantResolved {
		want[r.name] = true
	}
	owned := make(map[string]bool, len(prevResolved))
	for _, r := range prevResolved {
		owned[r.name] = true
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
		if !owned[name] {
			continue // this alert never attached it — leave it to its owner
		}
		if err := apiClient.DetachNotificationSettings(binding.ID); err != nil {
			return fmt.Errorf("failed to detach notification channel %q (binding id %d) from entity %s: %w", name, binding.ID, entityID, err)
		}
	}

	return nil
}

// generateKPIName creates a unique KPI name from the rule name plus a random token
func generateKPIName(ruleName string) string {
	token := make([]byte, 4)
	rand.Read(token)
	return fmt.Sprintf("%s-%s", ruleName, hex.EncodeToString(token))
}

func resourceAlert() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAlertCreate,
		ReadContext:   resourceAlertRead,
		UpdateContext: resourceAlertUpdate,
		DeleteContext: resourceAlertDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceAlertImportState,
		},
		Schema: map[string]*schema.Schema{
			"entity_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Entity ID this alert belongs to",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Alert name",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Alert description",
			},
			"query": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "PromQL query for the KPI/indicator that will be created for this alert",
			},
			"kpi_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the automatically created KPI for this alert",
			},
			"kpi_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the automatically created KPI for this alert",
			},
			"indicator": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Indicator name (derived from KPI name)",
			},
			"expression": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Alert expression (computed from KPI name)",
			},
			"greater_than": {
				Type:        schema.TypeFloat,
				Optional:    true,
				Description: "Threshold value for greater than condition",
			},
			"less_than": {
				Type:        schema.TypeFloat,
				Optional:    true,
				Description: "Threshold value for less than condition",
			},
			"equal_to": {
				Type:        schema.TypeFloat,
				Optional:    true,
				Description: "Threshold value for equality condition",
			},
			"not_equal": {
				Type:        schema.TypeFloat,
				Optional:    true,
				Description: "Threshold value for inequality condition",
			},
			"bad_minutes": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Minutes threshold must be exceeded",
			},
			"total_minutes": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Evaluation window in minutes",
			},
			"severity": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "breach",
				Description:  "Alert severity (breach or threat)",
				ValidateFunc: validation.StringInSlice([]string{"breach", "threat"}, false),
			},
			"is_disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether alert is disabled",
			},
			"properties": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Alert properties",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"runbook_url": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Runbook URL",
						},
						"annotations": {
							Type:        schema.TypeMap,
							Optional:    true,
							Description: "Alert annotations",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"group_timeseries_notifications": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Group timeseries notifications",
			},
			"notification_channels": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Notification channel IDs or names to send alerts to",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceAlertCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	entityID := d.Get("entity_id").(string)
	ruleName := d.Get("name").(string)
	query := d.Get("query").(string)

	// Step 1: Create a KPI for this alert
	kpiName := generateKPIName(ruleName)
	kpiReq := &client.KPICreateRequest{
		Name: kpiName,
		Definition: client.KPIDefinition{
			Query:  query,
			Source: "levitate",
			Unit:   "count",
		},
		KPIType: "custom",
	}

	kpi, err := apiClient.CreateKPI(entityID, kpiReq)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create KPI for alert: %w", err))
	}

	// Store KPI info in state
	d.Set("kpi_id", kpi.ID)
	d.Set("kpi_name", kpi.Name)
	d.Set("indicator", kpi.Name)

	// Step 2: Create the alert using the KPI
	req := &client.AlertCreateRequest{
		RuleName:                     ruleName,
		PrimaryIndicator:             kpi.Name,
		Severity:                     d.Get("severity").(string),
		IsDisabled:                   d.Get("is_disabled").(bool),
		GroupTimeseriesNotifications: d.Get("group_timeseries_notifications").(bool),
		ExpressionArgs: map[string]interface{}{
			kpi.Name: map[string]interface{}{
				"id": kpi.ID,
			},
		},
	}

	// Handle notification channels
	if v, ok := d.GetOk("notification_channels"); ok {
		channelsList := v.([]interface{})
		channels := make([]string, len(channelsList))
		for i, ch := range channelsList {
			channels[i] = ch.(string)
		}
		req.NotificationChannels = channels
	}

	// Handle static threshold alerts.
	// Use GetRawConfig to distinguish explicit 0 from omitted — d.GetOk returns false
	// for zero-value floats, making "greater_than = 0" indistinguishable from absent.
	if condition := buildStaticThresholdCondition(d); condition != "" {
		badMinutes := d.Get("bad_minutes").(int)
		totalMinutes := d.Get("total_minutes").(int)
		req.Condition = condition
		req.AlertCondition = fmt.Sprintf("count_true(result) >= %d", badMinutes)
		req.EvalWindow = totalMinutes
	}

	// Handle properties
	if v, ok := d.GetOk("properties"); ok {
		propsList := v.([]interface{})
		if len(propsList) > 0 {
			propsMap := propsList[0].(map[string]interface{})
			req.Properties = client.AlertProperties{
				Description: d.Get("description").(string),
			}
			if runbookURL, ok := propsMap["runbook_url"].(string); ok && runbookURL != "" {
				req.Properties.Runbook = map[string]interface{}{
					"link": runbookURL,
				}
			}
			if annotations, ok := propsMap["annotations"].(map[string]interface{}); ok {
				req.Properties.Annotations = make(map[string]string)
				for k, v := range annotations {
					req.Properties.Annotations[k] = v.(string)
				}
			}
		}
	} else {
		req.Properties = client.AlertProperties{
			Description: d.Get("description").(string),
		}
	}

	alert, err := apiClient.CreateAlert(entityID, req)
	if err != nil {
		// Clean up the KPI if alert creation fails
		if cleanupErr := apiClient.DeleteKPI(entityID, kpi.ID); cleanupErr != nil {
			return diag.FromErr(fmt.Errorf("failed to create alert: %w (also failed to cleanup KPI: %v)", err, cleanupErr))
		}
		return diag.FromErr(fmt.Errorf("failed to create alert: %w", err))
	}

	// Validate that we got a valid alert ID back from the API
	if alert.ID == "" {
		// Clean up the KPI since alert creation didn't return a valid ID
		if cleanupErr := apiClient.DeleteKPI(entityID, kpi.ID); cleanupErr != nil {
			return diag.FromErr(fmt.Errorf("alert creation succeeded but API returned empty alert ID - alert may not have been created (also failed to cleanup KPI: %v)", cleanupErr))
		}
		return diag.FromErr(fmt.Errorf("alert creation succeeded but API returned empty alert ID - alert may not have been created"))
	}

	d.SetId(alert.ID)

	// The alert-rules API has no notification_channels field of its own (see
	// reconcileNotificationChannels docstring) — bind each requested channel
	// via a separate attach call now that the entity/alert exist. Nothing to
	// detach on a brand-new entity, so skip the reconcile call entirely when
	// the list is empty.
	if len(req.NotificationChannels) > 0 {
		if err := reconcileNotificationChannels(apiClient, entityID, req.Severity, nil, req.NotificationChannels); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceAlertRead(ctx, d, m)
}

func resourceAlertRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	entityID := d.Get("entity_id").(string)

	alert, err := apiClient.GetAlert(entityID, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read alert: %w", err))
	}

	d.Set("entity_id", entityID)
	d.Set("name", alert.Name)
	d.Set("description", alert.Properties.Description)
	d.Set("indicator", alert.Indicator)
	d.Set("expression", alert.Expression)
	d.Set("severity", alert.Severity)
	// Note: mute and is_disabled fields are intentionally not read from API
	// The API may return different values than what was sent, causing drift
	d.Set("group_timeseries_notifications", alert.GroupTimeseriesNotifications)

	// alert.NotificationChannels is never populated by the API — the
	// alert-rules endpoint has no such field (see reconcileNotificationChannels).
	// The binding API has no per-alert ownership, so Read must not simply
	// report "every live binding on this entity/severity" as this alert's
	// notification_channels: on a shared entity that would promote a
	// sibling alert's channel into this alert's state, and the next
	// Update's reconcile (which trusts state as "what this alert owns," per
	// reconcileNotificationChannels) would then detach that sibling's
	// channel as no-longer-wanted the moment this alert's config doesn't
	// happen to list it too.
	//
	// So Read only ever keeps or drops entries this alert already claimed
	// in state — it never adds one: for each configured value, resolve it
	// to its canonical channel name (see resolveChannels for why name, not
	// ID, is the join key against live binding rows) and keep the value (in
	// its original ID-or-name spelling, so config doesn't drift from "1" to
	// "Channel A" or back) only if that name still has a live binding at
	// this alert's severity; drop it if the binding was removed outside
	// Terraform. A channel that failed to attach, or was detached
	// externally, still shows up as drift — it just disappears from the
	// list rather than the list growing to include channels this alert
	// never declared.
	configured := toStringSlice(d.Get("notification_channels").([]interface{}))
	bindings, err := apiClient.GetEntityNotificationBindings(entityID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read notification bindings for entity %s: %w", entityID, err))
	}
	liveAtSeverity := make(map[string]bool, len(bindings))
	for _, b := range bindings {
		if b.Severity == alert.Severity {
			liveAtSeverity[b.Name] = true
		}
	}
	stillOwned := make([]string, 0, len(configured))
	for _, value := range configured {
		dest, err := resolveNotificationChannel(apiClient, value)
		if err != nil {
			continue // channel no longer resolvable at all -- drop it, surfaces as drift
		}
		if liveAtSeverity[dest.Name] {
			stillOwned = append(stillOwned, value)
		}
	}
	d.Set("notification_channels", stillOwned)

	// Parse condition for static alerts to extract threshold values
	if alert.Condition != "" && alert.EvalWindow > 0 {
		// Parse condition like "expr > 100" or "expr < 50"
		// Extract threshold and operator
		if err := parseAndSetCondition(d, alert.Condition, alert.EvalWindow, alert.AlertCondition); err != nil {
			// Log warning but don't fail - condition parsing is best effort
			// The alert will still work, just won't have threshold values in state
		}
	}

	// Set properties (only if there are runbook_url or annotations)
	runbookURL := ""
	if alert.Properties.Runbook != nil {
		if link, ok := alert.Properties.Runbook["link"].(string); ok {
			runbookURL = link
		}
	}

	// Only set properties block if there are values to set
	if runbookURL != "" || len(alert.Properties.Annotations) > 0 {
		props := []interface{}{
			map[string]interface{}{
				"runbook_url": runbookURL,
				"annotations": alert.Properties.Annotations,
			},
		}
		d.Set("properties", props)
	}

	return nil
}

func resourceAlertUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	entityID := d.Get("entity_id").(string)
	oldKPIID := d.Get("kpi_id").(string)
	oldKPIName := d.Get("kpi_name").(string)

	// Track if we need to create a new KPI (name or query changed)
	needsNewKPI := d.HasChange("name") || d.HasChange("query")
	var newKPI *client.KPI

	if needsNewKPI {
		// Create new KPI with new name
		newName := d.Get("name").(string)
		newQuery := d.Get("query").(string)
		newKPIName := generateKPIName(newName)

		kpiReq := &client.KPICreateRequest{
			Name: newKPIName,
			Definition: client.KPIDefinition{
				Query:  newQuery,
				Source: "levitate",
				Unit:   "count",
			},
			KPIType: "custom",
		}

		var err error
		newKPI, err = apiClient.CreateKPI(entityID, kpiReq)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to create new KPI for alert update: %w", err))
		}
	}

	req := &client.AlertUpdateRequest{}

	// PUT requires all fields - always include name, severity, indicator refs
	// (Partial updates are not supported for alert-rules per OpenAPI spec)
	name := d.Get("name").(string)
	req.RuleName = &name

	severity := d.Get("severity").(string)
	req.Severity = &severity

	// Determine KPI name and ID to use
	kpiName := d.Get("kpi_name").(string)
	kpiID := d.Get("kpi_id").(string)

	// If we created a new KPI, use its details
	if newKPI != nil {
		kpiName = newKPI.Name
		kpiID = newKPI.ID
	}

	req.PrimaryIndicator = &kpiName
	req.ExpressionArgs = map[string]interface{}{
		kpiName: map[string]interface{}{
			"id": kpiID,
		},
	}

	// Note: mute field is not sent to API - it's ignored by Terraform

	isDisabled := d.Get("is_disabled").(bool)
	req.IsDisabled = &isDisabled

	group := d.Get("group_timeseries_notifications").(bool)
	req.GroupTimeseriesNotifications = &group
	// Always include notification_channels (PUT requires all fields)
	channelsList := d.Get("notification_channels").([]interface{})
	channels := make([]string, len(channelsList))
	for i, ch := range channelsList {
		channels[i] = ch.(string)
	}
	req.NotificationChannels = channels

	// PUT requires all fields - always include condition, alert_condition, eval_window
	// (Partial updates are not supported for alert-rules per OpenAPI spec)
	badMinutes := d.Get("bad_minutes").(int)
	totalMinutes := d.Get("total_minutes").(int)
	alertCondition := fmt.Sprintf("count_true(result) >= %d", badMinutes)
	req.AlertCondition = &alertCondition
	req.EvalWindow = &totalMinutes

	if condition := buildStaticThresholdCondition(d); condition != "" {
		req.Condition = &condition
	}

	// Always include properties (PUT requires all fields)
	props := client.AlertProperties{
		Description: d.Get("description").(string),
	}
	if v, ok := d.GetOk("properties"); ok {
		propsList := v.([]interface{})
		if len(propsList) > 0 {
			propsMap := propsList[0].(map[string]interface{})
			if runbookURL, ok := propsMap["runbook_url"].(string); ok && runbookURL != "" {
				props.Runbook = map[string]interface{}{
					"link": runbookURL,
				}
			}
			if annotations, ok := propsMap["annotations"].(map[string]interface{}); ok {
				props.Annotations = make(map[string]string)
				for k, v := range annotations {
					props.Annotations[k] = v.(string)
				}
			}
		}
	}
	req.Properties = &props

	updatedAlert, err := apiClient.UpdateAlert(entityID, d.Id(), req)
	if err != nil {
		// If we created a new KPI but alert update failed, clean it up
		if newKPI != nil {
			if cleanupErr := apiClient.DeleteKPI(entityID, newKPI.ID); cleanupErr != nil {
				return diag.FromErr(fmt.Errorf("failed to update alert: %w (also failed to cleanup KPI: %v)", err, cleanupErr))
			}
		}
		return diag.FromErr(fmt.Errorf("failed to update alert: %w", err))
	}

	// Validate that we got a valid alert ID back from the API
	if updatedAlert.ID == "" {
		// If we created a new KPI but got no alert ID, clean it up
		if newKPI != nil {
			if cleanupErr := apiClient.DeleteKPI(entityID, newKPI.ID); cleanupErr != nil {
				return diag.FromErr(fmt.Errorf("alert update succeeded but API returned empty alert ID - alert may not have been updated (also failed to cleanup KPI: %v)", cleanupErr))
			}
		}
		return diag.FromErr(fmt.Errorf("alert update succeeded but API returned empty alert ID - alert may not have been updated"))
	}

	// API deletes and recreates alert on update - update the ID
	// (per OpenAPI spec: "Partial updating is not supported for alert-rules.
	// The existing rule is deleted and a new one is created as process of updating alert-rules.")
	if updatedAlert.ID != d.Id() {
		d.SetId(updatedAlert.ID)
	}

	// If we created a new KPI, delete the old one and update state
	if newKPI != nil {
		// Delete old KPI (ignore errors - it may already be gone)
		if oldKPIID != "" && oldKPIName != "" {
			apiClient.DeleteKPI(entityID, oldKPIID)
		}

		// Update state with new KPI info
		d.Set("kpi_id", newKPI.ID)
		d.Set("kpi_name", newKPI.Name)
		d.Set("indicator", newKPI.Name)
	}

	// Bindings are keyed on entity_id, which is stable across this update
	// (only the alert-rule itself is deleted/recreated), so a binding that's
	// still wanted survives untouched. Always reconcile — including when
	// req.NotificationChannels is empty, which means "remove every channel"
	// and must still run to detach whatever was bound before. Skipping this
	// call on an empty list was the exact case that let a channel silently
	// stay bound after being removed from config.
	//
	// prevChannels scopes detachment to what THIS alert previously declared
	// (see reconcileNotificationChannels), so a sibling alert's binding on
	// the same entity/severity is never touched even if this alert's new
	// list happens not to include it.
	oldChannelsRaw, _ := d.GetChange("notification_channels")
	prevChannels := toStringSlice(oldChannelsRaw.([]interface{}))
	oldSeverity, _ := d.GetChange("severity")

	if oldSeverity.(string) != *req.Severity {
		// Severity changed: the previously-owned bindings live under the OLD
		// severity, so detach them there first (want=nil clears everything
		// this alert owned), then attach the full new list fresh under the
		// new severity.
		if err := reconcileNotificationChannels(apiClient, entityID, oldSeverity.(string), prevChannels, nil); err != nil {
			return diag.FromErr(err)
		}
		prevChannels = nil
	}

	if err := reconcileNotificationChannels(apiClient, entityID, *req.Severity, prevChannels, req.NotificationChannels); err != nil {
		return diag.FromErr(err)
	}

	return resourceAlertRead(ctx, d, m)
}

func resourceAlertDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	entityID := d.Get("entity_id").(string)
	kpiID := d.Get("kpi_id").(string)

	// Delete the alert first
	err := apiClient.DeleteAlert(entityID, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete alert: %w", err))
	}

	// Delete the associated KPI (ignore errors - it may already be gone)
	if kpiID != "" {
		apiClient.DeleteKPI(entityID, kpiID)
	}

	d.SetId("")
	return nil
}

// resourceAlertImportState handles importing alerts using composite ID format: entity_id:alert_id
func resourceAlertImportState(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), ":")
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		return nil, fmt.Errorf("invalid import ID format, expected 'entity_id:alert_id', got: %s", d.Id())
	}

	entityID := idParts[0]
	alertID := idParts[1]

	d.Set("entity_id", entityID)
	d.SetId(alertID)

	// resourceAlertRead treats current state as this alert's owned
	// notification_channels and only ever keeps or drops entries already
	// there (see the ownership note on resourceAlertRead) — it never adds
	// one, specifically so a shared entity's sibling bindings can't leak in.
	// A freshly imported resource has no state yet, so without seeding it
	// here Read would leave notification_channels permanently empty even
	// though the alert has real bindings. Import is the one case where
	// "adopt everything currently live at this alert's severity" is the
	// correct initial ownership — there is no prior config to compare
	// against, so the live set at import time IS what the user is asking
	// Terraform to take ownership of.
	apiClient := m.(*client.Client)
	alert, err := apiClient.GetAlert(entityID, alertID)
	if err != nil {
		return nil, fmt.Errorf("failed to read alert for import: %w", err)
	}
	bindings, err := apiClient.GetEntityNotificationBindings(entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to read notification bindings for import: %w", err)
	}
	initialChannels := make([]string, 0, len(bindings))
	for _, b := range bindings {
		if b.Severity == alert.Severity {
			initialChannels = append(initialChannels, b.Name)
		}
	}
	d.Set("notification_channels", initialChannels)

	return []*schema.ResourceData{d}, nil
}

// parseAndSetCondition parses the alert condition string and sets the appropriate
// schema fields (greater_than, less_than, equal_to, not_equal, bad_minutes, total_minutes)
func parseAndSetCondition(d *schema.ResourceData, condition string, evalWindow int, alertCondition string) error {
	// Parse condition like "expr > 100", "expr < 50", "expr == 1", or "expr != 0"
	// Extract operator and threshold value
	var threshold float64

	if len(condition) > 5 && condition[:5] == "expr " {
		rest := condition[5:]
		thresholdFields := []struct {
			prefix string
			field  string
		}{
			{"> ", "greater_than"},
			{"< ", "less_than"},
			{"== ", "equal_to"},
			{"!= ", "not_equal"},
		}
		for _, thresholdField := range thresholdFields {
			if !strings.HasPrefix(rest, thresholdField.prefix) {
				continue
			}
			if _, err := fmt.Sscanf(strings.TrimPrefix(rest, thresholdField.prefix), "%f", &threshold); err != nil {
				return fmt.Errorf("failed to parse threshold from condition: %w", err)
			}
			d.Set(thresholdField.field, threshold)
			break
		}
	}

	// Parse eval_window (total_minutes)
	if evalWindow > 0 {
		d.Set("total_minutes", evalWindow)
	}

	// Parse alert_condition like "count_true(result) >= 5" to extract bad_minutes
	if alertCondition != "" {
		var badMinutes int
		if _, err := fmt.Sscanf(alertCondition, "count_true(result) >= %d", &badMinutes); err == nil {
			d.Set("bad_minutes", badMinutes)
		}
	}

	return nil
}

func buildStaticThresholdCondition(d *schema.ResourceData) string {
	rawCfg := d.GetRawConfig()
	thresholdFields := []struct {
		field    string
		operator string
	}{
		{"greater_than", ">"},
		{"less_than", "<"},
		{"equal_to", "=="},
		{"not_equal", "!="},
	}

	for _, thresholdField := range thresholdFields {
		if !rawCfg.GetAttr(thresholdField.field).IsNull() {
			return fmt.Sprintf("expr %s %f", thresholdField.operator, d.Get(thresholdField.field).(float64))
		}
	}
	return ""
}
