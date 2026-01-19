package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func resourcePolicy() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePolicyCreate,
		ReadContext:   resourcePolicyRead,
		UpdateContext: resourcePolicyUpdate,
		DeleteContext: resourcePolicyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Policy name",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Policy description",
			},
			"filters": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Entity filters for policy application",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"rules": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of policy rules",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Rule type",
						},
						"config": {
							Type:        schema.TypeMap,
							Required:    true,
							Description: "Rule configuration",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"entity_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of entities attached",
			},
			"entity_compliant_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of compliant entities",
			},
		},
	}
}

func resourcePolicyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	req := &client.PolicyCreateRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Rules:       expandPolicyRules(d.Get("rules").([]interface{})),
	}

	if v, ok := d.GetOk("filters"); ok {
		filters := make(map[string]interface{})
		for k, v := range v.(map[string]interface{}) {
			filters[k] = v
		}
		req.Filters = filters
	}

	policy, err := apiClient.CreatePolicy(req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create policy: %w", err))
	}

	d.SetId(policy.ID)
	return resourcePolicyRead(ctx, d, m)
}

func resourcePolicyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	policy, err := apiClient.GetPolicy(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read policy: %w", err))
	}

	d.Set("name", policy.Name)
	d.Set("description", policy.Description)
	d.Set("rules", flattenPolicyRules(policy.Rules))
	d.Set("filters", policy.Filters)
	d.Set("entity_count", policy.EntityCount)
	d.Set("entity_compliant_count", policy.EntityCompliantCount)

	return nil
}

func resourcePolicyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	req := &client.PolicyUpdateRequest{}

	if d.HasChange("name") {
		name := d.Get("name").(string)
		req.Name = &name
	}
	if d.HasChange("description") {
		description := d.Get("description").(string)
		req.Description = &description
	}
	if d.HasChange("rules") {
		req.Rules = expandPolicyRules(d.Get("rules").([]interface{}))
	}
	if d.HasChange("filters") {
		filters := make(map[string]interface{})
		if v, ok := d.GetOk("filters"); ok {
			for k, v := range v.(map[string]interface{}) {
				filters[k] = v
			}
		}
		req.Filters = filters
	}

	_, err := apiClient.UpdatePolicy(d.Id(), req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update policy: %w", err))
	}

	return resourcePolicyRead(ctx, d, m)
}

func resourcePolicyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	err := apiClient.DeletePolicy(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete policy: %w", err))
	}

	d.SetId("")
	return nil
}

func expandPolicyRules(rules []interface{}) []client.PolicyRule {
	result := make([]client.PolicyRule, 0, len(rules))
	for _, r := range rules {
		ruleMap := r.(map[string]interface{})
		rule := client.PolicyRule{
			Type: ruleMap["type"].(string),
		}
		if config, ok := ruleMap["config"].(map[string]interface{}); ok {
			rule.Config = make(map[string]interface{})
			for k, v := range config {
				rule.Config[k] = v
			}
		}
		result = append(result, rule)
	}
	return result
}

func flattenPolicyRules(rules []client.PolicyRule) []interface{} {
	result := make([]interface{}, 0, len(rules))
	for _, r := range rules {
		ruleMap := map[string]interface{}{
			"type": r.Type,
		}
		if r.Config != nil {
			config := make(map[string]interface{})
			for k, v := range r.Config {
				config[k] = v
			}
			ruleMap["config"] = config
		}
		result = append(result, ruleMap)
	}
	return result
}
