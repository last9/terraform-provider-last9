package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func resourceDropRule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDropRuleCreate,
		ReadContext:   resourceDropRuleRead,
		UpdateContext: resourceDropRuleUpdate,
		DeleteContext: resourceDropRuleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Region for the drop rule",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the drop rule",
			},
			"telemetry": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Telemetry type (logs, traces, metrics)",
			},
			"filters": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "Filters to match logs for dropping",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Filter key (must use attributes[\"key\"] or resource.attributes[\"key\"])",
						},
						"value": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Filter value",
						},
						"operator": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Filter operator (equals, not_equals)",
							ValidateFunc: validation.StringInSlice([]string{"equals", "not_equals"}, false),
						},
						"conjunction": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Conjunction for combining filters (and)",
						},
					},
				},
			},
			"action": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: "Action to take when rule matches",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Action name",
						},
						"destination": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Action destination",
						},
						"properties": {
							Type:        schema.TypeMap,
							Optional:    true,
							Description: "Action properties",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func resourceDropRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	region := d.Get("region").(string)

	rule := &client.DropRule{
		Name:      d.Get("name").(string),
		Telemetry: d.Get("telemetry").(string),
		Filters:   expandRoutingFilters(d.Get("filters").([]interface{})),
		Action:    expandRoutingAction(d.Get("action").([]interface{})[0].(map[string]interface{})),
	}

	result, err := apiClient.UpsertDropRule(region, rule)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create drop rule: %w", err))
	}

	// Find the created rule and set ID
	for _, prop := range result.Properties {
		if prop.Name == rule.Name {
			d.SetId(fmt.Sprintf("%s:%s:%s", region, result.ID, prop.Name))
			break
		}
	}

	return resourceDropRuleRead(ctx, d, m)
}

func resourceDropRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	region := d.Get("region").(string)

	result, err := apiClient.GetDropRules(region)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read drop rules: %w", err))
	}

	// Parse ID to get rule name
	ruleName := d.Get("name").(string)
	var foundRule *client.DropRule
	for i := range result.Properties {
		if result.Properties[i].Name == ruleName {
			foundRule = &result.Properties[i]
			break
		}
	}

	if foundRule == nil {
		d.SetId("")
		return nil
	}

	d.Set("name", foundRule.Name)
	d.Set("telemetry", foundRule.Telemetry)
	d.Set("filters", flattenRoutingFilters(foundRule.Filters))
	d.Set("action", flattenRoutingAction(foundRule.Action))

	return nil
}

func resourceDropRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Update is same as create for drop rules (upsert)
	return resourceDropRuleCreate(ctx, d, m)
}

func resourceDropRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Drop rules are deleted by removing them from the list
	// This would require fetching all rules, removing this one, and updating
	// For now, we'll mark as deleted
	d.SetId("")
	return nil
}

func expandRoutingFilters(filters []interface{}) []client.RoutingFilter {
	result := make([]client.RoutingFilter, 0, len(filters))
	for _, f := range filters {
		filterMap := f.(map[string]interface{})
		filter := client.RoutingFilter{
			Key:      filterMap["key"].(string),
			Value:    filterMap["value"].(string),
			Operator: filterMap["operator"].(string),
		}
		if conj, ok := filterMap["conjunction"].(string); ok && conj != "" {
			filter.Conjunction = &conj
		}
		result = append(result, filter)
	}
	return result
}

func flattenRoutingFilters(filters []client.RoutingFilter) []interface{} {
	result := make([]interface{}, 0, len(filters))
	for _, f := range filters {
		filterMap := map[string]interface{}{
			"key":      f.Key,
			"value":    f.Value,
			"operator": f.Operator,
		}
		if f.Conjunction != nil {
			filterMap["conjunction"] = *f.Conjunction
		}
		result = append(result, filterMap)
	}
	return result
}

func expandRoutingAction(actionMap map[string]interface{}) client.RoutingAction {
	action := client.RoutingAction{
		Name:        actionMap["name"].(string),
		Destination: actionMap["destination"].(string),
		Properties:  make(map[string]string),
	}
	if props, ok := actionMap["properties"].(map[string]interface{}); ok {
		for k, v := range props {
			action.Properties[k] = v.(string)
		}
	}
	return action
}

func flattenRoutingAction(action client.RoutingAction) []interface{} {
	props := make(map[string]interface{})
	for k, v := range action.Properties {
		props[k] = v
	}
	return []interface{}{
		map[string]interface{}{
			"name":        action.Name,
			"destination": action.Destination,
			"properties":  props,
		},
	}
}
