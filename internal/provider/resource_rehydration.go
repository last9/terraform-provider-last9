package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func resourceRehydration() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRehydrationCreate,
		ReadContext:   resourceRehydrationRead,
		DeleteContext: resourceRehydrationDelete,
		// Rehydration jobs are typically create-once; updates are status patches only.
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"physical_index": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"telemetry": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"logs"}, false),
			},
			"from": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Start unix timestamp",
			},
			"to": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "End unix timestamp",
			},
			"message": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"granularity": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"bucket_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"notification_channel_id": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"targets": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"filters": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     otelFilterSchema(),
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceRehydrationCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region := d.Get("region").(string)

	resp, err := c.CreateOTelRehydration(region, buildRehydrationRequest(d))
	if err != nil {
		return diag.FromErr(fmt.Errorf("create rehydration: %w", err))
	}
	d.SetId(fmt.Sprintf("%s:%s", region, resp.ID))
	return resourceRehydrationRead(ctx, d, m)
}

func resourceRehydrationRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, id, err := parseRegionID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := c.GetOTelRehydration(id, region)
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("read rehydration: %w", err))
	}

	_ = d.Set("region", region)
	_ = d.Set("name", resp.Name)
	_ = d.Set("physical_index", resp.Properties.PhysicalIndex)
	_ = d.Set("telemetry", resp.Properties.Telemetry)
	_ = d.Set("from", int(resp.Properties.From))
	_ = d.Set("to", int(resp.Properties.To))
	_ = d.Set("message", resp.Properties.Message)
	_ = d.Set("granularity", resp.Properties.Granularity)
	_ = d.Set("targets", resp.Properties.Targets)
	_ = d.Set("filters", flattenOTelFilters(resp.Properties.Filters))
	_ = d.Set("status", resp.Status)
	if resp.Properties.BucketName != nil {
		_ = d.Set("bucket_name", *resp.Properties.BucketName)
	}
	if resp.Properties.NotificationChannelID != nil {
		_ = d.Set("notification_channel_id", int(*resp.Properties.NotificationChannelID))
	}
	return nil
}

func resourceRehydrationDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, id, err := parseRegionID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteOTelRehydration(id, region); err != nil {
		return diag.FromErr(fmt.Errorf("delete rehydration: %w", err))
	}
	d.SetId("")
	return nil
}

func buildRehydrationRequest(d *schema.ResourceData) *client.RehydrationOTelRequest {
	props := client.RehydrationOTelProperties{
		PhysicalIndex: d.Get("physical_index").(string),
		Telemetry:     d.Get("telemetry").(string),
		From:          int64(d.Get("from").(int)),
		To:            int64(d.Get("to").(int)),
		Message:       d.Get("message").(string),
		Granularity:   d.Get("granularity").(string),
		Filters:       expandOTelFilters(d.Get("filters").([]interface{})),
		Targets:       expandStringList(d.Get("targets").([]interface{})),
	}
	if v, ok := d.GetOk("bucket_name"); ok && v.(string) != "" {
		bn := v.(string)
		props.BucketName = &bn
	}
	if v, ok := d.GetOk("notification_channel_id"); ok {
		id := int64(v.(int))
		props.NotificationChannelID = &id
	}
	return &client.RehydrationOTelRequest{
		Name:       d.Get("name").(string),
		Properties: props,
	}
}
