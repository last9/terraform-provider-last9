package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func resourceStreamingAggregation() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceStreamingAggregationCreate,
		ReadContext:   resourceStreamingAggregationRead,
		UpdateContext: resourceStreamingAggregationUpdate,
		DeleteContext: resourceStreamingAggregationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"cluster_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"telemetry": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"metrics", "events"}, false),
				Description:  "metrics or events (logs/traces use scheduled search)",
			},
			"metric": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Source metric name",
			},
			"resolution": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Resolution like 1m, 5m, 1h",
			},
			"aggregation": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"max", "sum", "sum2", "sum_counter"}, false),
			},
			"clause": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"with", "without"}, false),
				Description:  "Label clause: with (by) or without",
			},
			"labels": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"output_metric": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Prometheus-style output metric name",
			},
			"with_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"with_value": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceStreamingAggregationCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region := d.Get("region").(string)
	clusterID, err := resolveClusterID(c, d)
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := c.CreateStreamingAggregation(clusterID, region, buildStreamingAggRequest(d))
	if err != nil {
		return diag.FromErr(fmt.Errorf("create streaming aggregation: %w", err))
	}
	_ = d.Set("cluster_id", clusterID)
	d.SetId(fmt.Sprintf("%s:%s:%s", region, clusterID, resp.ID))
	return resourceStreamingAggregationRead(ctx, d, m)
}

func resourceStreamingAggregationRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, clusterID, id, err := parseOTelResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := c.GetStreamingAggregation(clusterID, region, id)
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("read streaming aggregation: %w", err))
	}

	_ = d.Set("region", region)
	_ = d.Set("cluster_id", clusterID)
	_ = d.Set("name", resp.Name)
	_ = d.Set("telemetry", resp.Telemetry)
	_ = d.Set("metric", resp.Properties.Metric)
	_ = d.Set("resolution", resp.Properties.Resolution)
	_ = d.Set("aggregation", resp.Properties.Aggregation)
	_ = d.Set("clause", resp.Properties.Clause)
	_ = d.Set("labels", resp.Properties.Labels)
	_ = d.Set("output_metric", resp.Properties.OutputMetric)
	_ = d.Set("with_name", resp.Properties.WithName)
	_ = d.Set("with_value", resp.Properties.WithValue)
	return nil
}

func resourceStreamingAggregationUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, clusterID, id, err := parseOTelResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if _, err := c.UpdateStreamingAggregation(clusterID, region, id, buildStreamingAggRequest(d)); err != nil {
		return diag.FromErr(fmt.Errorf("update streaming aggregation: %w", err))
	}
	return resourceStreamingAggregationRead(ctx, d, m)
}

func resourceStreamingAggregationDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, clusterID, id, err := parseOTelResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteStreamingAggregation(clusterID, region, id); err != nil {
		return diag.FromErr(fmt.Errorf("delete streaming aggregation: %w", err))
	}
	d.SetId("")
	return nil
}

func buildStreamingAggRequest(d *schema.ResourceData) *client.StreamingAggRequest {
	return &client.StreamingAggRequest{
		Name:      d.Get("name").(string),
		Telemetry: d.Get("telemetry").(string),
		Properties: client.StreamingAggProperties{
			Metric:       d.Get("metric").(string),
			Resolution:   d.Get("resolution").(string),
			Aggregation:  d.Get("aggregation").(string),
			Clause:       d.Get("clause").(string),
			Labels:       expandStringList(d.Get("labels").([]interface{})),
			OutputMetric: d.Get("output_metric").(string),
			WithName:     d.Get("with_name").(string),
			WithValue:    d.Get("with_value").(string),
		},
	}
}
