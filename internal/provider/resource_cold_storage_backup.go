package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func resourceColdStorageBackup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceColdStorageBackupCreate,
		ReadContext:   resourceColdStorageBackupRead,
		UpdateContext: resourceColdStorageBackupUpdate,
		DeleteContext: resourceColdStorageBackupDelete,
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
			},
			"bucket_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Cold storage bucket name (Last9 bucket config name)",
			},
			"granularity": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"index", "service"}, false),
			},
			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"targets": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Required when granularity=service; must be empty for index",
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceColdStorageBackupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region := d.Get("region").(string)

	resp, err := c.CreateColdStorageBackup(region, buildColdStorageBackupRequest(d))
	if err != nil {
		return diag.FromErr(fmt.Errorf("create cold storage backup: %w", err))
	}
	d.SetId(fmt.Sprintf("%s:%s", region, resp.ID))
	return resourceColdStorageBackupRead(ctx, d, m)
}

func resourceColdStorageBackupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, id, err := parseRegionID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := c.GetColdStorageBackup(id, region)
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("read cold storage backup: %w", err))
	}

	_ = d.Set("region", region)
	_ = d.Set("name", resp.Name)
	_ = d.Set("bucket_name", resp.Properties.BucketName)
	_ = d.Set("granularity", resp.Properties.Granularity)
	_ = d.Set("targets", resp.Properties.Targets)
	_ = d.Set("status", resp.Status)
	if resp.Properties.Enabled != nil {
		_ = d.Set("enabled", *resp.Properties.Enabled)
	}
	return nil
}

func resourceColdStorageBackupUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, id, err := parseRegionID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if _, err := c.UpdateColdStorageBackup(id, region, buildColdStorageBackupRequest(d)); err != nil {
		return diag.FromErr(fmt.Errorf("update cold storage backup: %w", err))
	}
	return resourceColdStorageBackupRead(ctx, d, m)
}

func resourceColdStorageBackupDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, id, err := parseRegionID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteColdStorageBackup(id, region); err != nil {
		return diag.FromErr(fmt.Errorf("delete cold storage backup: %w", err))
	}
	d.SetId("")
	return nil
}

func buildColdStorageBackupRequest(d *schema.ResourceData) *client.ColdStorageBackupRequest {
	enabled := d.Get("enabled").(bool)
	return &client.ColdStorageBackupRequest{
		Name: d.Get("name").(string),
		Properties: client.ColdStorageBackupProperties{
			Enabled:     &enabled,
			BucketName:  d.Get("bucket_name").(string),
			Granularity: d.Get("granularity").(string),
			Targets:     expandStringList(d.Get("targets").([]interface{})),
		},
	}
}
