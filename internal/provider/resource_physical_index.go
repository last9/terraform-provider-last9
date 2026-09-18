package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

var physicalIndexNameRE = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func resourcePhysicalIndex() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePhysicalIndexCreate,
		ReadContext:   resourcePhysicalIndexRead,
		UpdateContext: resourcePhysicalIndexUpdate,
		DeleteContext: resourcePhysicalIndexDelete,
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
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(physicalIndexNameRE, "name must be alphanumeric with underscores"),
				Description:  "Physical index name (alphanumeric and underscores)",
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"telemetry": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"logs"}, false),
			},
			"filters": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem:     otelFilterSchema(),
			},
			"retain": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"retention_period": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"bucket_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourcePhysicalIndexCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region := d.Get("region").(string)
	clusterID, err := resolveClusterID(c, d)
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := c.CreatePhysicalIndex(region, clusterID, buildPhysicalIndexRequest(d))
	if err != nil {
		return diag.FromErr(fmt.Errorf("create physical index: %w", err))
	}
	_ = d.Set("cluster_id", clusterID)
	d.SetId(fmt.Sprintf("%s:%s:%s", region, clusterID, resp.ID))
	return resourcePhysicalIndexRead(ctx, d, m)
}

func resourcePhysicalIndexRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, clusterID, otelID, err := parseOTelResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := c.GetPhysicalIndex(otelID, region)
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("read physical index: %w", err))
	}

	_ = d.Set("region", region)
	_ = d.Set("cluster_id", clusterID)
	_ = d.Set("name", resp.Name)
	_ = d.Set("description", resp.Properties.Description)
	_ = d.Set("telemetry", resp.Properties.Telemetry)
	_ = d.Set("filters", flattenOTelFilters(resp.Properties.Filters))
	_ = d.Set("retain", resp.Properties.Retain)
	if resp.Properties.RetentionPeriod != nil {
		_ = d.Set("retention_period", *resp.Properties.RetentionPeriod)
	}
	if resp.Properties.BucketName != nil {
		_ = d.Set("bucket_name", *resp.Properties.BucketName)
	}
	_ = d.Set("status", resp.Status)
	return nil
}

func resourcePhysicalIndexUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, clusterID, otelID, err := parseOTelResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if _, err := c.UpdatePhysicalIndex(otelID, region, clusterID, buildPhysicalIndexRequest(d)); err != nil {
		return diag.FromErr(fmt.Errorf("update physical index: %w", err))
	}
	return resourcePhysicalIndexRead(ctx, d, m)
}

func resourcePhysicalIndexDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, clusterID, otelID, err := parseOTelResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var lastErr error
	for attempt := 0; attempt < 10; attempt++ {
		lastErr = c.DeletePhysicalIndex(otelID, region, clusterID)
		if lastErr == nil {
			d.SetId("")
			return nil
		}
		msg := strings.ToLower(lastErr.Error())
		// Index still provisioning — keep ID and retry so destroy can succeed later.
		if strings.Contains(msg, "not created yet") || strings.Contains(msg, "cannot soft-delete") {
			select {
			case <-ctx.Done():
				return diag.FromErr(fmt.Errorf("delete physical index: %w", ctx.Err()))
			case <-time.After(time.Duration(attempt+1) * 500 * time.Millisecond):
			}
			continue
		}
		return diag.FromErr(fmt.Errorf("delete physical index: %w", lastErr))
	}
	// Still pending after retries — retain ID so a later destroy can retry.
	return diag.FromErr(fmt.Errorf("delete physical index: %w", lastErr))
}

func buildPhysicalIndexRequest(d *schema.ResourceData) *client.PhysicalIndexRequest {
	props := client.PhysicalIndexProperties{
		Description: d.Get("description").(string),
		Telemetry:   d.Get("telemetry").(string),
		Filters:     expandOTelFilters(d.Get("filters").([]interface{})),
		Retain:      d.Get("retain").(bool),
	}
	if v, ok := d.GetOk("retention_period"); ok {
		rp := v.(int)
		props.RetentionPeriod = &rp
	}
	if v, ok := d.GetOk("bucket_name"); ok && v.(string) != "" {
		bn := v.(string)
		props.BucketName = &bn
	}
	return &client.PhysicalIndexRequest{
		Name:       d.Get("name").(string),
		Properties: props,
	}
}
