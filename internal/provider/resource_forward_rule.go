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
				Description: "Name of the forward rule",
			},
			"telemetry": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Telemetry type: logs or traces",
				ValidateFunc: validation.StringInSlice([]string{"logs", "traces"}, false),
			},
			"destination": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Destination URL for forwarding",
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
			},
			"filters": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: "Filters to match telemetry for forwarding",
				Elem:        otelFilterSchema(),
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceForwardRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region := d.Get("region").(string)
	clusterID, err := resolveClusterID(c, d)
	if err != nil {
		return diag.FromErr(err)
	}

	req := buildOTelForwardRequest(d)
	resp, err := c.CreateOTelForward(region, clusterID, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("create forward rule: %w", err))
	}

	_ = d.Set("cluster_id", clusterID)
	d.SetId(fmt.Sprintf("%s:%s:%s", region, clusterID, resp.ID))
	return resourceForwardRuleRead(ctx, d, m)
}

func resourceForwardRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, clusterID, otelID, err := parseOTelResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := c.GetOTelForward(otelID, region)
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("read forward rule: %w", err))
	}

	_ = d.Set("region", region)
	_ = d.Set("cluster_id", clusterID)
	_ = d.Set("name", resp.Name)
	_ = d.Set("telemetry", resp.Properties.Telemetry)
	_ = d.Set("destination", resp.Properties.Destination)
	_ = d.Set("filters", flattenOTelFilters(resp.Properties.Filters))
	_ = d.Set("status", resp.Status)
	return nil
}

func resourceForwardRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, clusterID, otelID, err := parseOTelResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := buildOTelForwardRequest(d)
	if _, err := c.UpdateOTelForward(otelID, region, clusterID, req); err != nil {
		return diag.FromErr(fmt.Errorf("update forward rule: %w", err))
	}
	return resourceForwardRuleRead(ctx, d, m)
}

func resourceForwardRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, clusterID, otelID, err := parseOTelResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteOTelForward(otelID, region, clusterID); err != nil {
		return diag.FromErr(fmt.Errorf("delete forward rule: %w", err))
	}
	d.SetId("")
	return nil
}

func buildOTelForwardRequest(d *schema.ResourceData) *client.OTelForwardRequest {
	return &client.OTelForwardRequest{
		Name: d.Get("name").(string),
		Properties: client.OTelForwardProperties{
			Telemetry:   d.Get("telemetry").(string),
			Destination: d.Get("destination").(string),
			Filters:     expandOTelFilters(d.Get("filters").([]interface{})),
		},
	}
}
