package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func resourceColdStorageBucket() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceColdStorageBucketCreate,
		ReadContext:   resourceColdStorageBucketRead,
		UpdateContext: resourceColdStorageBucketUpdate,
		DeleteContext: resourceColdStorageBucketDelete,
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
			"aws_region": {
				Type:     schema.TypeString,
				Required: true,
			},
			"aws_bucket": {
				Type:     schema.TypeString,
				Required: true,
			},
			"auth_type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"credentials", "role"}, false),
			},
			"aws_access_key": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},
			"aws_secret_key": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},
			"aws_role": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"default": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"retention_period": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceColdStorageBucketCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region := d.Get("region").(string)

	resp, err := c.CreateColdStorageBucket(region, buildColdStorageBucketRequest(d))
	if err != nil {
		return diag.FromErr(fmt.Errorf("create cold storage bucket: %w", err))
	}
	d.SetId(fmt.Sprintf("%s:%s", region, resp.ID))

	// Create may ignore default; explicitly mark when requested.
	if d.Get("default").(bool) {
		if err := c.MarkDefaultColdStorageBucket(resp.ID, region); err != nil {
			return diag.FromErr(fmt.Errorf("mark default cold storage bucket: %w", err))
		}
	}
	return resourceColdStorageBucketRead(ctx, d, m)
}

func resourceColdStorageBucketRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, id, err := parseRegionID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := c.GetColdStorageBucket(id, region)
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("read cold storage bucket: %w", err))
	}

	_ = d.Set("region", region)
	_ = d.Set("name", resp.Name)
	_ = d.Set("aws_region", resp.Properties.AWSRegion)
	_ = d.Set("aws_bucket", resp.Properties.AWSBucket)
	_ = d.Set("auth_type", resp.Properties.AuthType)
	_ = d.Set("aws_role", resp.Properties.AWSRole)
	_ = d.Set("default", resp.Properties.Default)
	_ = d.Set("status", resp.Status)
	if resp.Properties.RetentionPeriod != nil {
		_ = d.Set("retention_period", *resp.Properties.RetentionPeriod)
	}
	// Preserve secrets in state when API omits/masks them.
	if resp.Properties.AWSAccessKey != "" {
		_ = d.Set("aws_access_key", resp.Properties.AWSAccessKey)
	}
	if resp.Properties.AWSSecretKey != "" {
		_ = d.Set("aws_secret_key", resp.Properties.AWSSecretKey)
	}
	return nil
}

func resourceColdStorageBucketUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, id, err := parseRegionID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if _, err := c.UpdateColdStorageBucket(id, region, buildColdStorageBucketRequest(d)); err != nil {
		return diag.FromErr(fmt.Errorf("update cold storage bucket: %w", err))
	}
	if d.HasChange("default") && d.Get("default").(bool) {
		if err := c.MarkDefaultColdStorageBucket(id, region); err != nil {
			return diag.FromErr(fmt.Errorf("mark default cold storage bucket: %w", err))
		}
	}
	return resourceColdStorageBucketRead(ctx, d, m)
}

func resourceColdStorageBucketDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, id, err := parseRegionID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteColdStorageBucket(id, region); err != nil {
		return diag.FromErr(fmt.Errorf("delete cold storage bucket: %w", err))
	}
	d.SetId("")
	return nil
}

func buildColdStorageBucketRequest(d *schema.ResourceData) *client.ColdStorageBucketRequest {
	props := client.ColdStorageBucketProperties{
		Default:   d.Get("default").(bool),
		AWSRegion: d.Get("aws_region").(string),
		AWSBucket: d.Get("aws_bucket").(string),
		AuthType:  d.Get("auth_type").(string),
		AWSRole:   d.Get("aws_role").(string),
	}
	if v, ok := d.GetOk("aws_access_key"); ok {
		props.AWSAccessKey = v.(string)
	}
	if v, ok := d.GetOk("aws_secret_key"); ok {
		props.AWSSecretKey = v.(string)
	}
	if v, ok := d.GetOk("retention_period"); ok {
		rp := v.(int)
		props.RetentionPeriod = &rp
	}
	return &client.ColdStorageBucketRequest{
		Name:       d.Get("name").(string),
		Properties: props,
	}
}
