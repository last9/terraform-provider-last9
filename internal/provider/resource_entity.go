package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func resourceEntity() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceEntityCreate,
		ReadContext:   resourceEntityRead,
		UpdateContext: resourceEntityUpdate,
		DeleteContext: resourceEntityDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
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
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Notification channel IDs or names assigned to this entity",
				Elem:        &schema.Schema{Type: schema.TypeString},
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

	// Notification channels
	if v, ok := d.GetOk("notification_channels"); ok {
		channelsList := v.([]interface{})
		channels := make([]string, len(channelsList))
		for i, ch := range channelsList {
			channels[i] = ch.(string)
		}
		req.NotificationChannels = channels
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
	d.Set("notification_channels", entity.NotificationChannels)

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

	// Check if metadata fields changed (tags, labels, team, links, adhoc_filter)
	// These are updated via PUT /entities/{id}/metadata
	metadataFieldsChanged := d.HasChange("tags") || d.HasChange("labels") || d.HasChange("team") ||
		d.HasChange("links") || d.HasChange("adhoc_filter")

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

		err := apiClient.UpdateEntityMetadata(d.Id(), metadataReq)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to update entity metadata: %w", err))
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
