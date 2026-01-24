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

// dropRuleMutex serializes drop rule operations to prevent race conditions.
// The drop rules API stores all rules as a single list, so concurrent
// create/update/delete operations can overwrite each other.
var dropRuleMutex sync.Mutex

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
			"cluster_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Cluster ID for the drop rule. If not specified, the default cluster for the region will be used.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the drop rule",
			},
			"telemetry": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Telemetry type: 'logs', 'traces', or 'metrics'.",
				ValidateFunc: validation.StringInSlice([]string{"logs", "traces", "metrics"}, false),
			},
			"filters": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "Filters to match telemetry for dropping. For logs/traces, use attribute filters. For metrics, use PromQL expressions in the value field.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Filter key. For logs/traces: must use attributes[\"key\"] or resource.attributes[\"key\"]. For metrics: can be empty (PromQL goes in value).",
						},
						"value": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Filter value. For logs/traces: the attribute value to match. For metrics: a PromQL expression like {__name__=~\"metric_name\"}.",
						},
						"operator": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Filter operator: equals, not_equals, or like (regex match)",
							ValidateFunc: validation.StringInSlice([]string{"equals", "not_equals", "like"}, false),
						},
						"conjunction": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Conjunction for combining filters (AND)",
						},
					},
				},
			},
			"action": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: "Action to take when rule matches. Use name = 'drop-matching' for drop rules.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Action name. Must be 'drop-matching' for drop rules.",
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
	clusterID := d.Get("cluster_id").(string)
	ruleName := d.Get("name").(string)

	// If cluster_id is not provided, fetch the default cluster for the region
	if clusterID == "" {
		defaultCluster, err := apiClient.GetDefaultCluster(region)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to get default cluster for region %s: %w", region, err))
		}
		clusterID = defaultCluster.ID
		d.Set("cluster_id", clusterID)
	}

	newRule := client.DropRule{
		Name:      ruleName,
		Telemetry: d.Get("telemetry").(string),
		Filters:   expandRoutingFilters(d.Get("filters").([]interface{})),
		Action:    expandDropAction(d.Get("action").([]interface{})[0].(map[string]interface{})),
	}

	// Lock to prevent race conditions - drop rules are stored as a single list
	dropRuleMutex.Lock()
	defer dropRuleMutex.Unlock()

	// Get existing rules
	existing, err := apiClient.GetDropRules(region)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get existing drop rules: %w", err))
	}

	// Check if rule already exists (to avoid duplicates)
	for _, rule := range existing.Properties {
		if rule.Name == ruleName {
			return diag.FromErr(fmt.Errorf("drop rule %s already exists in region %s", ruleName, region))
		}
	}

	// Add new rule to the list
	rules := append(existing.Properties, newRule)

	// POST the updated list
	req := &client.DropRulesRequest{
		Properties: rules,
	}

	_, err = apiClient.UpdateDropRules(region, clusterID, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create drop rule: %w", err))
	}

	// Set ID (format: region:cluster_id:rule_name)
	d.SetId(fmt.Sprintf("%s:%s:%s", region, clusterID, ruleName))

	return resourceDropRuleRead(ctx, d, m)
}

func resourceDropRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	// Parse ID to extract region, cluster_id, and rule name (format: region:cluster_id:rule_name)
	// During import, d.Get("region") will be empty, so we need to parse from ID
	id := d.Id()
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return diag.FromErr(fmt.Errorf("invalid drop rule ID format: %s (expected region:cluster_id:rule_name)", id))
	}

	region := parts[0]
	clusterID := parts[1]
	ruleName := parts[2]

	result, err := apiClient.GetDropRules(region)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read drop rules: %w", err))
	}

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

	// Set all attributes from the found rule
	d.Set("region", region)
	d.Set("cluster_id", clusterID)
	d.Set("name", foundRule.Name)
	d.Set("telemetry", foundRule.Telemetry)
	d.Set("filters", flattenRoutingFilters(foundRule.Filters))
	d.Set("action", flattenDropAction(foundRule.Action))

	return nil
}

func resourceDropRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	region := d.Get("region").(string)
	clusterID := d.Get("cluster_id").(string)
	ruleName := d.Get("name").(string)

	// Lock to prevent race conditions - drop rules are stored as a single list
	dropRuleMutex.Lock()
	defer dropRuleMutex.Unlock()

	// Get existing rules
	existing, err := apiClient.GetDropRules(region)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get existing drop rules: %w", err))
	}

	// Create updated rule
	updatedRule := client.DropRule{
		Name:      ruleName,
		Telemetry: d.Get("telemetry").(string),
		Filters:   expandRoutingFilters(d.Get("filters").([]interface{})),
		Action:    expandDropAction(d.Get("action").([]interface{})[0].(map[string]interface{})),
	}

	// Update the rule in the list
	rules := make([]client.DropRule, 0, len(existing.Properties))
	found := false
	for _, rule := range existing.Properties {
		if rule.Name == ruleName {
			rules = append(rules, updatedRule)
			found = true
		} else {
			rules = append(rules, rule)
		}
	}

	if !found {
		return diag.FromErr(fmt.Errorf("drop rule %s not found for update", ruleName))
	}

	// POST the updated list
	req := &client.DropRulesRequest{
		Properties: rules,
	}

	result, err := apiClient.UpdateDropRules(region, clusterID, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update drop rule: %w", err))
	}

	// Update ID with new response ID
	d.SetId(fmt.Sprintf("%s:%s:%s", region, clusterID, ruleName))
	_ = result // Response ID not needed as we use cluster_id

	return resourceDropRuleRead(ctx, d, m)
}

func resourceDropRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	// Parse ID to extract region, cluster_id, and rule name (format: region:cluster_id:rule_name)
	id := d.Id()
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return diag.FromErr(fmt.Errorf("invalid drop rule ID format: %s", id))
	}

	region := parts[0]
	clusterID := parts[1]
	ruleName := parts[2]

	// Lock to prevent race conditions - drop rules are stored as a single list
	dropRuleMutex.Lock()
	defer dropRuleMutex.Unlock()

	// Get existing rules
	existing, err := apiClient.GetDropRules(region)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get existing drop rules: %w", err))
	}

	// Remove the rule from the list
	rules := make([]client.DropRule, 0)
	for _, rule := range existing.Properties {
		if rule.Name != ruleName {
			rules = append(rules, rule)
		}
	}

	// Update with the filtered list using cluster_id
	req := &client.DropRulesRequest{
		Properties: rules,
	}

	_, err = apiClient.UpdateDropRules(region, clusterID, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete drop rule: %w", err))
	}

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

func expandDropAction(actionMap map[string]interface{}) client.RoutingAction {
	return client.RoutingAction{
		Name: actionMap["name"].(string),
	}
}

func flattenDropAction(action client.RoutingAction) []interface{} {
	return []interface{}{
		map[string]interface{}{
			"name": action.Name,
		},
	}
}
