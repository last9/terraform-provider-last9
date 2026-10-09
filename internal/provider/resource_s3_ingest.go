package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func resourceS3Ingest() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceS3IngestCreate,
		ReadContext:   resourceS3IngestRead,
		UpdateContext: resourceS3IngestUpdate,
		DeleteContext: resourceS3IngestDelete,
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
			"aws_role": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "IAM role ARN (auth_type is always role)",
			},
			"auth_type": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "role",
				ValidateFunc: validation.StringInSlice([]string{"role"}, false),
			},
			"default": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceS3IngestCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region := d.Get("region").(string)

	resp, err := c.CreateS3Ingest(region, buildS3IngestRequest(d))
	if err != nil {
		return diag.FromErr(fmt.Errorf("create s3 ingest: %w", err))
	}
	d.SetId(fmt.Sprintf("%s:%s", region, resp.ID))
	return resourceS3IngestRead(ctx, d, m)
}

func resourceS3IngestRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, id, err := parseRegionID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := c.GetS3Ingest(id, region)
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("read s3 ingest: %w", err))
	}

	_ = d.Set("region", region)
	_ = d.Set("name", resp.Name)
	_ = d.Set("aws_region", resp.Properties.AWSRegion)
	_ = d.Set("aws_bucket", resp.Properties.AWSBucket)
	_ = d.Set("aws_role", resp.Properties.AWSRole)
	_ = d.Set("auth_type", resp.Properties.AuthType)
	_ = d.Set("default", resp.Properties.Default)
	_ = d.Set("status", resp.Status)
	return nil
}

func resourceS3IngestUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, id, err := parseRegionID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if _, err := c.UpdateS3Ingest(id, region, buildS3IngestRequest(d)); err != nil {
		return diag.FromErr(fmt.Errorf("update s3 ingest: %w", err))
	}
	return resourceS3IngestRead(ctx, d, m)
}

func resourceS3IngestDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, id, err := parseRegionID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteS3Ingest(id, region); err != nil {
		return diag.FromErr(fmt.Errorf("delete s3 ingest: %w", err))
	}
	d.SetId("")
	return nil
}

func buildS3IngestRequest(d *schema.ResourceData) *client.S3IngestRequest {
	return &client.S3IngestRequest{
		Name: d.Get("name").(string),
		Properties: client.S3IngestProperties{
			Default:   d.Get("default").(bool),
			AWSBucket: d.Get("aws_bucket").(string),
			AuthType:  d.Get("auth_type").(string),
			AWSRole:   d.Get("aws_role").(string),
			AWSRegion: d.Get("aws_region").(string),
		},
	}
}
