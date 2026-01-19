package provider

import (
	"context"
	"fmt"

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

	// Get existing rules
	existing, err := apiClient.GetForwardRules(region)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get existing forward rules: %w", err))
	}

	// Create new rule
	newRule := client.ForwardRule{
		Name:        d.Get("name").(string),
		Telemetry:   d.Get("telemetry").(string),
		Destination: d.Get("destination").(string),
		Filters:     expandRoutingFilters(d.Get("filters").([]interface{})),
	}

	// Add to existing rules
	rules := existing.Properties
	rules = append(rules, newRule)

	req := &client.ForwardRulesRequest{
		Properties: rules,
	}

	result, err := apiClient.UpsertForwardRules(region, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create forward rule: %w", err))
	}

	d.SetId(fmt.Sprintf("%s:%s:%s", region, result.ID, newRule.Name))
	return resourceForwardRuleRead(ctx, d, m)
}

func resourceForwardRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	region := d.Get("region").(string)

	result, err := apiClient.GetForwardRules(region)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read forward rules: %w", err))
	}

	ruleName := d.Get("name").(string)
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

	d.Set("name", foundRule.Name)
	d.Set("telemetry", foundRule.Telemetry)
	d.Set("destination", foundRule.Destination)
	d.Set("filters", flattenRoutingFilters(foundRule.Filters))

	return nil
}

func resourceForwardRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	region := d.Get("region").(string)

	// Get existing rules
	existing, err := apiClient.GetForwardRules(region)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get existing forward rules: %w", err))
	}

	ruleName := d.Get("name").(string)
	updated := false

	// Update the rule in the list
	rules := make([]client.ForwardRule, 0, len(existing.Properties))
	for _, rule := range existing.Properties {
		if rule.Name == ruleName {
			rule.Telemetry = d.Get("telemetry").(string)
			rule.Destination = d.Get("destination").(string)
			rule.Filters = expandRoutingFilters(d.Get("filters").([]interface{}))
			updated = true
		}
		rules = append(rules, rule)
	}

	if !updated {
		return diag.FromErr(fmt.Errorf("forward rule not found"))
	}

	req := &client.ForwardRulesRequest{
		Properties: rules,
	}

	_, err = apiClient.UpsertForwardRules(region, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update forward rule: %w", err))
	}

	return resourceForwardRuleRead(ctx, d, m)
}

func resourceForwardRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	region := d.Get("region").(string)

	// Get existing rules
	existing, err := apiClient.GetForwardRules(region)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get existing forward rules: %w", err))
	}

	ruleName := d.Get("name").(string)

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

	_, err = apiClient.UpsertForwardRules(region, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete forward rule: %w", err))
	}

	d.SetId("")
	return nil
}
