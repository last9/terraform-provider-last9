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
				Description: "Cluster ID. If omitted, the default cluster for the region is used.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the drop rule",
			},
			"telemetry": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Telemetry type: logs, traces, or metrics",
				ValidateFunc: validation.StringInSlice([]string{"logs", "traces", "metrics"}, false),
			},
			"filters": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: "Filters to match telemetry for dropping",
				Elem:        otelFilterSchema(),
			},
			"action": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: "Drop action",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"destination": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"properties": {
							Type:     schema.TypeMap,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func otelFilterSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"key": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"value": {
				Type:     schema.TypeString,
				Required: true,
			},
			"operator": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"equals", "not_equals", "like"}, false),
			},
			"conjunction": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceDropRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region := d.Get("region").(string)
	clusterID, err := resolveClusterID(c, d)
	if err != nil {
		return diag.FromErr(err)
	}

	req := buildOTelDropRequest(d)
	resp, err := c.CreateOTelDrop(region, clusterID, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("create drop rule: %w", err))
	}

	_ = d.Set("cluster_id", clusterID)
	d.SetId(fmt.Sprintf("%s:%s:%s", region, clusterID, resp.ID))
	return resourceDropRuleRead(ctx, d, m)
}

func resourceDropRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, clusterID, otelID, err := parseOTelResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := c.GetOTelDrop(otelID, region)
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("read drop rule: %w", err))
	}

	_ = d.Set("region", region)
	_ = d.Set("cluster_id", clusterID)
	_ = d.Set("name", resp.Name)
	_ = d.Set("telemetry", resp.Properties.Telemetry)
	_ = d.Set("filters", flattenOTelFilters(resp.Properties.Filters))
	_ = d.Set("action", []interface{}{map[string]interface{}{
		"name":        resp.Properties.Action.Name,
		"destination": resp.Properties.Action.Destination,
		"properties":  resp.Properties.Action.Properties,
	}})
	_ = d.Set("status", resp.Status)
	return nil
}

func resourceDropRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, clusterID, otelID, err := parseOTelResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := buildOTelDropRequest(d)
	if _, err := c.UpdateOTelDrop(otelID, region, clusterID, req); err != nil {
		return diag.FromErr(fmt.Errorf("update drop rule: %w", err))
	}
	return resourceDropRuleRead(ctx, d, m)
}

func resourceDropRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, clusterID, otelID, err := parseOTelResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteOTelDrop(otelID, region, clusterID); err != nil {
		return diag.FromErr(fmt.Errorf("delete drop rule: %w", err))
	}
	d.SetId("")
	return nil
}

func buildOTelDropRequest(d *schema.ResourceData) *client.OTelDropRequest {
	actionList := d.Get("action").([]interface{})
	actionMap := actionList[0].(map[string]interface{})
	action := client.OTelDropAction{
		Name:        actionMap["name"].(string),
		Destination: actionMap["destination"].(string),
	}
	if props, ok := actionMap["properties"].(map[string]interface{}); ok {
		action.Properties = expandStringMap(props)
	}
	return &client.OTelDropRequest{
		Name: d.Get("name").(string),
		Properties: client.OTelDropProperties{
			Telemetry: d.Get("telemetry").(string),
			Filters:   expandOTelFilters(d.Get("filters").([]interface{})),
			Action:    action,
		},
	}
}

func expandOTelFilters(in []interface{}) []client.OTelSettingFilter {
	out := make([]client.OTelSettingFilter, 0, len(in))
	for _, raw := range in {
		m := raw.(map[string]interface{})
		f := client.OTelSettingFilter{
			Key:      m["key"].(string),
			Value:    m["value"].(string),
			Operator: m["operator"].(string),
		}
		if c, ok := m["conjunction"].(string); ok && c != "" {
			f.Conjunction = &c
		}
		out = append(out, f)
	}
	return out
}

func flattenOTelFilters(in []client.OTelSettingFilter) []interface{} {
	out := make([]interface{}, 0, len(in))
	for _, f := range in {
		m := map[string]interface{}{
			"key":      f.Key,
			"value":    f.Value,
			"operator": f.Operator,
		}
		if f.Conjunction != nil {
			m["conjunction"] = *f.Conjunction
		}
		out = append(out, m)
	}
	return out
}

func resolveClusterID(c *client.Client, d *schema.ResourceData) (string, error) {
	if v, ok := d.GetOk("cluster_id"); ok && v.(string) != "" {
		return v.(string), nil
	}
	cluster, err := c.GetDefaultCluster(d.Get("region").(string))
	if err != nil {
		return "", fmt.Errorf("resolve default cluster: %w", err)
	}
	return cluster.ID, nil
}

// parseOTelResourceID parses region:cluster_id:otel_id
func parseOTelResourceID(id string) (region, clusterID, otelID string, err error) {
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("invalid ID %q; expected region:cluster_id:id", id)
	}
	return parts[0], parts[1], parts[2], nil
}

// Compatibility helpers kept for unit tests / shared filter expand patterns.
func expandRoutingFilters(input []interface{}) []client.RoutingFilter {
	otel := expandOTelFilters(input)
	out := make([]client.RoutingFilter, 0, len(otel))
	for _, f := range otel {
		out = append(out, client.RoutingFilter(f))
	}
	return out
}

func flattenRoutingFilters(input []client.RoutingFilter) []interface{} {
	otel := make([]client.OTelSettingFilter, 0, len(input))
	for _, f := range input {
		otel = append(otel, client.OTelSettingFilter(f))
	}
	return flattenOTelFilters(otel)
}

func expandDropAction(input map[string]interface{}) client.RoutingAction {
	action := client.RoutingAction{Name: input["name"].(string)}
	if dest, ok := input["destination"].(string); ok {
		action.Destination = dest
	}
	if props, ok := input["properties"].(map[string]interface{}); ok {
		action.Properties = expandStringMap(props)
	}
	return action
}

func flattenDropAction(action client.RoutingAction) []interface{} {
	return []interface{}{map[string]interface{}{
		"name":        action.Name,
		"destination": action.Destination,
		"properties":  action.Properties,
	}}
}
