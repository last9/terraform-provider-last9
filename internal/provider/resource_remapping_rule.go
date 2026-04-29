package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func resourceRemappingRule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRemappingRuleCreate,
		ReadContext:   resourceRemappingRuleRead,
		UpdateContext: resourceRemappingRuleUpdate,
		DeleteContext: resourceRemappingRuleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceRemappingRuleImportState,
		},
		Schema: map[string]*schema.Schema{
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Region for the remapping rule",
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"logs_extract", "logs_map", "traces_map",
				}, false),
				Description: "Remapping rule type: logs_extract, logs_map, or traces_map",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the remapping rule",
			},
			"remap_keys": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: "Source field(s) to remap from",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"target_attributes": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Target attribute: 'log_attributes' or 'resource_attributes' for logs_extract; 'service', 'severity', or 'resource_deployment.environment' for logs_map; 'service' for traces_map",
			},
			// logs_extract only
			"extract_type": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					"pattern", "json",
				}, false),
				Description: "Extraction type for logs_extract: pattern (regex) or json",
			},
			"action": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "upsert",
				ValidateFunc: validation.StringInSlice([]string{
					"insert", "upsert",
				}, false),
				Description: "Action for logs_extract: insert or upsert",
			},
			"prefix": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Optional prefix for extracted values (logs_extract only)",
			},
			"preconditions": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Conditional rules for when to apply extraction (logs_extract only)",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
						"operator": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								"equals", "not_equals", "like",
							}, false),
						},
					},
				},
			},
			// Computed
			"created_at": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"updated_at": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"created_by": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
		CustomizeDiff: validateRemappingRule,
	}
}

func validateRemappingRule(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
	ruleType := d.Get("type").(string)
	extractType := d.Get("extract_type").(string)
	preconditions := d.Get("preconditions").([]interface{})

	switch ruleType {
	case "logs_extract":
		if extractType == "" {
			return fmt.Errorf("extract_type is required for logs_extract type")
		}
		target := d.Get("target_attributes").(string)
		if target != "" && target != "log_attributes" && target != "resource_attributes" {
			return fmt.Errorf("target_attributes must be 'log_attributes' or 'resource_attributes' for logs_extract type")
		}
	case "logs_map", "traces_map":
		if extractType != "" {
			return fmt.Errorf("extract_type is only valid for logs_extract type")
		}
		if len(preconditions) > 0 {
			return fmt.Errorf("preconditions are only valid for logs_extract type")
		}
	}

	if ruleType == "logs_map" {
		target := d.Get("target_attributes").(string)
		validTargets := map[string]bool{
			"service":                         true,
			"severity":                        true,
			"resource_deployment.environment": true,
		}
		if target != "" && !validTargets[target] {
			return fmt.Errorf("target_attributes must be 'service', 'severity', or 'resource_deployment.environment' for logs_map type")
		}
	}

	if ruleType == "traces_map" {
		target := d.Get("target_attributes").(string)
		if target != "" && target != "service" {
			return fmt.Errorf("target_attributes must be 'service' for traces_map type")
		}
	}

	return nil
}

func resourceRemappingRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	region := d.Get("region").(string)
	ruleType := d.Get("type").(string)

	switch ruleType {
	case "logs_extract":
		req := buildLogsExtractRequest(d)
		result, err := apiClient.CreateRemappingLogsExtract(region, req)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to create remapping rule: %w", err))
		}
		d.SetId(result.ID)
	case "logs_map":
		req := buildLogsMapRequest(d)
		result, err := apiClient.CreateRemappingLogsMap(region, req)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to create remapping rule: %w", err))
		}
		d.SetId(result.ID)
	case "traces_map":
		req := buildLogsMapRequest(d)
		result, err := apiClient.CreateRemappingTracesMap(region, req)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to create remapping rule: %w", err))
		}
		d.SetId(result.ID)
	}

	return resourceRemappingRuleRead(ctx, d, m)
}

func resourceRemappingRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	id := d.Id()
	region := d.Get("region").(string)
	ruleType := d.Get("type").(string)

	switch ruleType {
	case "logs_extract":
		result, err := apiClient.GetRemappingLogsExtract(id, region)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				d.SetId("")
				return nil
			}
			return diag.FromErr(fmt.Errorf("failed to read remapping rule: %w", err))
		}
		d.Set("name", result.Name)
		d.Set("remap_keys", result.Properties.RemapKeys)
		d.Set("target_attributes", result.Properties.TargetAttributes)
		if result.Properties.Type != nil {
			d.Set("extract_type", *result.Properties.Type)
		}
		if result.Properties.Action != nil {
			d.Set("action", *result.Properties.Action)
		}
		if result.Properties.Prefix != nil {
			d.Set("prefix", *result.Properties.Prefix)
		}
		d.Set("preconditions", flattenPreconditions(result.Properties.Preconditions))
		d.Set("created_at", result.CreatedAt)
		d.Set("created_by", result.CreatedBy)
		d.Set("status", result.Status)
		if result.UpdatedAt != nil {
			d.Set("updated_at", *result.UpdatedAt)
		}
	case "logs_map":
		result, err := apiClient.GetRemappingLogsMap(id, region)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				d.SetId("")
				return nil
			}
			return diag.FromErr(fmt.Errorf("failed to read remapping rule: %w", err))
		}
		setRemappingMapFields(d, result.Name, result.Properties, result.CreatedAt, result.CreatedBy, result.UpdatedAt, result.Status)
	case "traces_map":
		result, err := apiClient.GetRemappingTracesMap(id, region)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				d.SetId("")
				return nil
			}
			return diag.FromErr(fmt.Errorf("failed to read remapping rule: %w", err))
		}
		setRemappingMapFields(d, result.Name, result.Properties, result.CreatedAt, result.CreatedBy, result.UpdatedAt, result.Status)
	}

	return nil
}

func resourceRemappingRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	id := d.Id()
	region := d.Get("region").(string)
	ruleType := d.Get("type").(string)

	switch ruleType {
	case "logs_extract":
		req := buildLogsExtractRequest(d)
		if _, err := apiClient.UpdateRemappingLogsExtract(id, region, req); err != nil {
			return diag.FromErr(fmt.Errorf("failed to update remapping rule: %w", err))
		}
	case "logs_map":
		req := buildLogsMapRequest(d)
		if _, err := apiClient.UpdateRemappingLogsMap(id, region, req); err != nil {
			return diag.FromErr(fmt.Errorf("failed to update remapping rule: %w", err))
		}
	case "traces_map":
		req := buildLogsMapRequest(d)
		if _, err := apiClient.UpdateRemappingTracesMap(id, region, req); err != nil {
			return diag.FromErr(fmt.Errorf("failed to update remapping rule: %w", err))
		}
	}

	return resourceRemappingRuleRead(ctx, d, m)
}

func resourceRemappingRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	id := d.Id()
	region := d.Get("region").(string)
	ruleType := d.Get("type").(string)

	var err error
	switch ruleType {
	case "logs_extract":
		err = apiClient.DeleteRemappingLogsExtract(id, region)
	case "logs_map":
		err = apiClient.DeleteRemappingLogsMap(id, region)
	case "traces_map":
		err = apiClient.DeleteRemappingTracesMap(id, region)
	}

	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete remapping rule: %w", err))
	}

	d.SetId("")
	return nil
}

func resourceRemappingRuleImportState(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	parts := strings.SplitN(d.Id(), ":", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return nil, fmt.Errorf("invalid import ID, expected 'region:type:id', got: %s", d.Id())
	}
	d.Set("region", parts[0])
	d.Set("type", parts[1])
	d.SetId(parts[2])
	return []*schema.ResourceData{d}, nil
}

func buildLogsExtractRequest(d *schema.ResourceData) *client.RemappingLogsExtractRequest {
	extractType := d.Get("extract_type").(string)
	action := d.Get("action").(string)

	props := client.RemappingLogsExtractProperties{
		RemapKeys:        expandStringList(d.Get("remap_keys").([]interface{})),
		TargetAttributes: d.Get("target_attributes").(string),
		Type:             &extractType,
		Action:           &action,
	}

	if prefix := d.Get("prefix").(string); prefix != "" {
		props.Prefix = &prefix
	}

	if raw := d.Get("preconditions").([]interface{}); len(raw) > 0 {
		props.Preconditions = expandPreconditions(raw)
	}

	return &client.RemappingLogsExtractRequest{
		Name:       d.Get("name").(string),
		Properties: props,
	}
}

func buildLogsMapRequest(d *schema.ResourceData) *client.RemappingLogsMapRequest {
	return &client.RemappingLogsMapRequest{
		Name: d.Get("name").(string),
		Properties: client.RemappingMapProperties{
			RemapKeys:        expandStringList(d.Get("remap_keys").([]interface{})),
			TargetAttributes: d.Get("target_attributes").(string),
		},
	}
}

func setRemappingMapFields(d *schema.ResourceData, name string, props client.RemappingMapProperties, createdAt int64, createdBy string, updatedAt *int64, status string) {
	d.Set("name", name)
	d.Set("remap_keys", props.RemapKeys)
	d.Set("target_attributes", props.TargetAttributes)
	d.Set("created_at", createdAt)
	d.Set("created_by", createdBy)
	d.Set("status", status)
	if updatedAt != nil {
		d.Set("updated_at", *updatedAt)
	}
}

func expandStringList(input []interface{}) []string {
	result := make([]string, 0, len(input))
	for _, v := range input {
		result = append(result, v.(string))
	}
	return result
}

func expandPreconditions(input []interface{}) []*client.RemappingLogsExtractPrecondition {
	result := make([]*client.RemappingLogsExtractPrecondition, 0, len(input))
	for _, v := range input {
		m := v.(map[string]interface{})
		result = append(result, &client.RemappingLogsExtractPrecondition{
			Key:      m["key"].(string),
			Value:    m["value"].(string),
			Operator: m["operator"].(string),
		})
	}
	return result
}

func flattenPreconditions(preconditions []*client.RemappingLogsExtractPrecondition) []interface{} {
	result := make([]interface{}, 0, len(preconditions))
	for _, p := range preconditions {
		if p != nil {
			result = append(result, map[string]interface{}{
				"key":      p.Key,
				"value":    p.Value,
				"operator": p.Operator,
			})
		}
	}
	return result
}
