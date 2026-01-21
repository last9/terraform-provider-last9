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

func resourceForwardRule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceForwardRuleCreate,
		ReadContext:   resourceForwardRuleRead,
		UpdateContext: resourceForwardRuleUpdate,
		DeleteContext: resourceForwardRuleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Region for the forward rule",
			},
			"cluster_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Cluster ID for the forward rule (from the clusters API)",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the forward rule",
			},
			"telemetry": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Telemetry type (logs, traces)",
				ValidateFunc: validation.StringInSlice([]string{"logs", "traces"}, false),
			},
			"destination": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Destination URL for forwarding logs",
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
			},
			"filters": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "Filters to match logs for forwarding",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Filter key",
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
		},
	}
}

func resourceForwardRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	region := d.Get("region").(string)
	clusterID := d.Get("cluster_id").(string)
	ruleName := d.Get("name").(string)

	// Get existing rules
	existing, err := apiClient.GetForwardRules(region)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get existing forward rules: %w", err))
	}

	// Check if rule already exists (to avoid duplicates)
	for _, rule := range existing.Properties {
		if rule.Name == ruleName {
			return diag.FromErr(fmt.Errorf("forward rule %s already exists in region %s", ruleName, region))
		}
	}

	// Create new rule
	newRule := client.ForwardRule{
		Name:        ruleName,
		Telemetry:   d.Get("telemetry").(string),
		Destination: d.Get("destination").(string),
		Filters:     expandRoutingFilters(d.Get("filters").([]interface{})),
	}

	// Add to existing rules
	rules := append(existing.Properties, newRule)

	req := &client.ForwardRulesRequest{
		Properties: rules,
	}

	_, err = apiClient.UpdateForwardRules(region, clusterID, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create forward rule: %w", err))
	}

	d.SetId(fmt.Sprintf("%s:%s:%s", region, clusterID, ruleName))
	return resourceForwardRuleRead(ctx, d, m)
}

func resourceForwardRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	// Parse ID to extract region, cluster_id, and rule name (format: region:cluster_id:rule_name)
	// During import, d.Get("region") will be empty, so we need to parse from ID
	id := d.Id()
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return diag.FromErr(fmt.Errorf("invalid forward rule ID format: %s (expected region:cluster_id:rule_name)", id))
	}

	region := parts[0]
	clusterID := parts[1]
	ruleName := parts[2]

	result, err := apiClient.GetForwardRules(region)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read forward rules: %w", err))
	}

	var foundRule *client.ForwardRule
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
	d.Set("destination", foundRule.Destination)
	d.Set("filters", flattenRoutingFilters(foundRule.Filters))

	return nil
}

func resourceForwardRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	region := d.Get("region").(string)
	clusterID := d.Get("cluster_id").(string)
	ruleName := d.Get("name").(string)

	// Get existing rules
	existing, err := apiClient.GetForwardRules(region)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get existing forward rules: %w", err))
	}

	// Create updated rule
	updatedRule := client.ForwardRule{
		Name:        ruleName,
		Telemetry:   d.Get("telemetry").(string),
		Destination: d.Get("destination").(string),
		Filters:     expandRoutingFilters(d.Get("filters").([]interface{})),
	}

	// Update the rule in the list
	rules := make([]client.ForwardRule, 0, len(existing.Properties))
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
		return diag.FromErr(fmt.Errorf("forward rule %s not found for update", ruleName))
	}

	req := &client.ForwardRulesRequest{
		Properties: rules,
	}

	_, err = apiClient.UpdateForwardRules(region, clusterID, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update forward rule: %w", err))
	}

	return resourceForwardRuleRead(ctx, d, m)
}

func resourceForwardRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	// Parse ID to extract region, cluster_id, and rule name (format: region:cluster_id:rule_name)
	id := d.Id()
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return diag.FromErr(fmt.Errorf("invalid forward rule ID format: %s", id))
	}

	region := parts[0]
	clusterID := parts[1]
	ruleName := parts[2]

	// Get existing rules
	existing, err := apiClient.GetForwardRules(region)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get existing forward rules: %w", err))
	}

	// Remove the rule from the list
	rules := make([]client.ForwardRule, 0)
	for _, rule := range existing.Properties {
		if rule.Name != ruleName {
			rules = append(rules, rule)
		}
	}

	req := &client.ForwardRulesRequest{
		Properties: rules,
	}

	_, err = apiClient.UpdateForwardRules(region, clusterID, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete forward rule: %w", err))
	}

	d.SetId("")
	return nil
}
