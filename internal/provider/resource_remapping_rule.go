package provider

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

// remappingRuleMutex serializes remapping rule operations to prevent race conditions.
// The remapping rules API stores all rules of a type as a single list, so concurrent
// create/update/delete operations can overwrite each other.
var remappingRuleMutex sync.Mutex

func resourceRemappingRule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRemappingRuleCreate,
		ReadContext:   resourceRemappingRuleRead,
		UpdateContext: resourceRemappingRuleUpdate,
		DeleteContext: resourceRemappingRuleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Region for the remapping rule",
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Remapping rule type: logs_extract, logs_map, or traces_map",
				ValidateFunc: validation.StringInSlice([]string{"logs_extract", "logs_map", "traces_map"}, false),
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the remapping rule (unique within type and region)",
			},
			"remap_keys": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "Source field(s) to remap from",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"target_attributes": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Target attribute to map to. For logs_extract: log_attributes or resource_attributes. For logs_map: service, severity, or resource_deployment.environment. For traces_map: service.",
			},
			"action": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "upsert",
				Description:  "Action to take: insert (only if not exists) or upsert (insert or update)",
				ValidateFunc: validation.StringInSlice([]string{"insert", "upsert"}, false),
			},
			// logs_extract specific fields
			"extract_type": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Extraction type for logs_extract: pattern (regex) or json",
				ValidateFunc: validation.StringInSlice([]string{"pattern", "json"}, false),
			},
			"prefix": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Optional prefix for extracted values (logs_extract only)",
			},
			"preconditions": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Conditional rules for when to apply extraction (logs_extract only)",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Attribute key to match",
						},
						"value": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Value to match against",
						},
						"operator": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Match operator: equals, not_equals, or like (regex)",
							ValidateFunc: validation.StringInSlice([]string{"equals", "not_equals", "like"}, false),
						},
					},
				},
			},
			// Computed fields
			"created_at": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Creation timestamp",
			},
			"updated_at": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Last update timestamp",
			},
		},
		CustomizeDiff: validateRemappingRule,
	}
}

// validateRemappingRule validates the remapping rule configuration based on type
func validateRemappingRule(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
	ruleType := d.Get("type").(string)
	extractType := d.Get("extract_type").(string)
	preconditions := d.Get("preconditions").([]interface{})
	prefix := d.Get("prefix").(string)
	targetAttributes := d.Get("target_attributes").(string)

	// Validate logs_extract specific fields
	if ruleType == "logs_extract" {
		if extractType == "" {
			return fmt.Errorf("extract_type is required for logs_extract type")
		}
		// Validate target_attributes for logs_extract
		if targetAttributes != "log_attributes" && targetAttributes != "resource_attributes" {
			return fmt.Errorf("target_attributes must be 'log_attributes' or 'resource_attributes' for logs_extract type")
		}
	} else {
		// For non-extract types, these fields should not be set
		if extractType != "" {
			return fmt.Errorf("extract_type is only valid for logs_extract type")
		}
		if len(preconditions) > 0 {
			return fmt.Errorf("preconditions are only valid for logs_extract type")
		}
		if prefix != "" {
			return fmt.Errorf("prefix is only valid for logs_extract type")
		}
	}

	// Validate target_attributes for logs_map
	if ruleType == "logs_map" {
		validTargets := map[string]bool{
			"service":                        true,
			"severity":                       true,
			"resource_deployment.environment": true,
		}
		if !validTargets[targetAttributes] {
			return fmt.Errorf("target_attributes must be 'service', 'severity', or 'resource_deployment.environment' for logs_map type")
		}
	}

	// Validate target_attributes for traces_map
	if ruleType == "traces_map" {
		if targetAttributes != "service" {
			return fmt.Errorf("target_attributes must be 'service' for traces_map type")
		}
	}

	return nil
}

func resourceRemappingRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	region := d.Get("region").(string)
	ruleType := d.Get("type").(string)
	ruleName := d.Get("name").(string)

	newProperty := buildRemapProperty(d)

	// Lock to prevent race conditions
	remappingRuleMutex.Lock()
	defer remappingRuleMutex.Unlock()

	// Get existing rules for this type
	existing, err := apiClient.GetRemappingRuleByType(region, ruleType)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return diag.FromErr(fmt.Errorf("failed to get existing remapping rules: %w", err))
	}

	var properties []client.RemapProperty
	if existing != nil {
		// Check if rule already exists
		for _, prop := range existing.Properties {
			if prop.Name != nil && *prop.Name == ruleName {
				return diag.FromErr(fmt.Errorf("remapping rule %s already exists for type %s in region %s", ruleName, ruleType, region))
			}
		}
		properties = existing.Properties
	}

	// Add new property
	properties = append(properties, newProperty)

	// Upsert the remapping rules
	req := &client.RemappingRuleRequest{
		Region:     region,
		Properties: properties,
	}

	result, err := apiClient.UpsertRemappingRule(region, ruleType, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create remapping rule: %w", err))
	}

	// Set ID (format: region:type:name)
	d.SetId(fmt.Sprintf("%s:%s:%s", region, ruleType, ruleName))

	// Set computed fields
	d.Set("created_at", result.CreatedAt)
	d.Set("updated_at", result.UpdatedAt)

	return resourceRemappingRuleRead(ctx, d, m)
}

func resourceRemappingRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	// Parse ID (format: region:type:name)
	id := d.Id()
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return diag.FromErr(fmt.Errorf("invalid remapping rule ID format: %s (expected region:type:name)", id))
	}

	region := parts[0]
	ruleType := parts[1]
	ruleName := parts[2]

	result, err := apiClient.GetRemappingRuleByType(region, ruleType)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to read remapping rules: %w", err))
	}

	if result == nil {
		d.SetId("")
		return nil
	}

	// Find our specific rule by name
	var foundProperty *client.RemapProperty
	for i := range result.Properties {
		prop := &result.Properties[i]
		if prop.Name != nil && *prop.Name == ruleName {
			foundProperty = prop
			break
		}
		// For map types, match by remap_keys + target_attributes if name not set
		if prop.Name == nil && ruleType != "logs_extract" {
			// Use the first matching rule for map types
			if foundProperty == nil {
				foundProperty = prop
			}
		}
	}

	if foundProperty == nil {
		d.SetId("")
		return nil
	}

	// Set attributes
	d.Set("region", region)
	d.Set("type", ruleType)
	if foundProperty.Name != nil {
		d.Set("name", *foundProperty.Name)
	}
	d.Set("remap_keys", foundProperty.RemapKeys)
	d.Set("target_attributes", foundProperty.TargetAttributes)
	if foundProperty.Action != nil {
		d.Set("action", *foundProperty.Action)
	}
	if foundProperty.Type != nil {
		d.Set("extract_type", *foundProperty.Type)
	}
	if foundProperty.Prefix != nil {
		d.Set("prefix", *foundProperty.Prefix)
	}
	if len(foundProperty.Preconditions) > 0 {
		d.Set("preconditions", flattenPreconditions(foundProperty.Preconditions))
	}
	d.Set("created_at", foundProperty.CreatedAt)
	d.Set("updated_at", result.UpdatedAt)

	return nil
}

func resourceRemappingRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	region := d.Get("region").(string)
	ruleType := d.Get("type").(string)
	ruleName := d.Get("name").(string)

	updatedProperty := buildRemapProperty(d)

	// Lock to prevent race conditions
	remappingRuleMutex.Lock()
	defer remappingRuleMutex.Unlock()

	// Get existing rules
	existing, err := apiClient.GetRemappingRuleByType(region, ruleType)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get existing remapping rules: %w", err))
	}

	// Update the rule in the list
	properties := make([]client.RemapProperty, 0, len(existing.Properties))
	found := false
	for _, prop := range existing.Properties {
		if prop.Name != nil && *prop.Name == ruleName {
			properties = append(properties, updatedProperty)
			found = true
		} else {
			properties = append(properties, prop)
		}
	}

	if !found {
		return diag.FromErr(fmt.Errorf("remapping rule %s not found for update", ruleName))
	}

	// Upsert the updated list
	req := &client.RemappingRuleRequest{
		Region:     region,
		Properties: properties,
	}

	_, err = apiClient.UpsertRemappingRule(region, ruleType, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update remapping rule: %w", err))
	}

	return resourceRemappingRuleRead(ctx, d, m)
}

func resourceRemappingRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	// Parse ID (format: region:type:name)
	id := d.Id()
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return diag.FromErr(fmt.Errorf("invalid remapping rule ID format: %s", id))
	}

	region := parts[0]
	ruleType := parts[1]
	ruleName := parts[2]

	// Lock to prevent race conditions
	remappingRuleMutex.Lock()
	defer remappingRuleMutex.Unlock()

	// Get existing rules
	existing, err := apiClient.GetRemappingRuleByType(region, ruleType)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to get existing remapping rules: %w", err))
	}

	// Remove the rule from the list
	properties := make([]client.RemapProperty, 0)
	for _, prop := range existing.Properties {
		if prop.Name == nil || *prop.Name != ruleName {
			properties = append(properties, prop)
		}
	}

	// Upsert the filtered list
	req := &client.RemappingRuleRequest{
		Region:     region,
		Properties: properties,
	}

	_, err = apiClient.UpsertRemappingRule(region, ruleType, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete remapping rule: %w", err))
	}

	d.SetId("")
	return nil
}

// Helper functions

func buildRemapProperty(d *schema.ResourceData) client.RemapProperty {
	ruleName := d.Get("name").(string)
	action := d.Get("action").(string)

	property := client.RemapProperty{
		Name:             &ruleName,
		RemapKeys:        expandStringList(d.Get("remap_keys").([]interface{})),
		TargetAttributes: d.Get("target_attributes").(string),
		Action:           &action,
	}

	// logs_extract specific fields
	if extractType := d.Get("extract_type").(string); extractType != "" {
		property.Type = &extractType
	}
	if prefix := d.Get("prefix").(string); prefix != "" {
		property.Prefix = &prefix
	}
	if preconditions := d.Get("preconditions").([]interface{}); len(preconditions) > 0 {
		property.Preconditions = expandPreconditions(preconditions)
	}

	return property
}

func expandStringList(input []interface{}) []string {
	result := make([]string, 0, len(input))
	for _, v := range input {
		result = append(result, v.(string))
	}
	return result
}

func expandPreconditions(input []interface{}) []*client.RemapPrecondition {
	result := make([]*client.RemapPrecondition, 0, len(input))
	for _, v := range input {
		m := v.(map[string]interface{})
		result = append(result, &client.RemapPrecondition{
			Key:      m["key"].(string),
			Value:    m["value"].(string),
			Operator: m["operator"].(string),
		})
	}
	return result
}

func flattenPreconditions(preconditions []*client.RemapPrecondition) []interface{} {
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
