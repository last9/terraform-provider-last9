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

func resourceSensitiveDataRule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSensitiveDataRuleCreate,
		ReadContext:   resourceSensitiveDataRuleRead,
		UpdateContext: resourceSensitiveDataRuleUpdate,
		DeleteContext: resourceSensitiveDataRuleDelete,
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
			"telemetry": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"logs"}, false),
				Description:  "Telemetry type (currently only logs)",
			},
			"order": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntAtLeast(1),
				Description:  "Rule evaluation order (minimum 1)",
			},
			"scan_email": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"scan_phone_number": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"scan_credit_card": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"action_name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"none", "redact"}, false),
				Description:  "Action when match found: none or redact",
			},
			"labels": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceSensitiveDataRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region := d.Get("region").(string)
	req := buildSensitiveDataRequest(d)

	resp, err := c.CreateSensitiveData(region, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("create sensitive data rule: %w", err))
	}
	d.SetId(fmt.Sprintf("%s:%s", region, resp.ID))
	return resourceSensitiveDataRuleRead(ctx, d, m)
}

func resourceSensitiveDataRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, id, err := parseRegionID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := c.GetSensitiveData(id, region)
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("read sensitive data rule: %w", err))
	}

	_ = d.Set("region", region)
	_ = d.Set("name", resp.Name)
	_ = d.Set("telemetry", resp.Properties.Telemetry)
	_ = d.Set("order", int(resp.Properties.Order))
	_ = d.Set("scan_email", resp.Properties.ScanRules.Email)
	_ = d.Set("scan_phone_number", resp.Properties.ScanRules.PhoneNumber)
	_ = d.Set("scan_credit_card", resp.Properties.ScanRules.CreditCardNumber)
	_ = d.Set("action_name", resp.Properties.Action.Name)
	_ = d.Set("labels", resp.Properties.Labels)
	_ = d.Set("status", resp.Status)
	return nil
}

func resourceSensitiveDataRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, id, err := parseRegionID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if _, err := c.UpdateSensitiveData(id, region, buildSensitiveDataRequest(d)); err != nil {
		return diag.FromErr(fmt.Errorf("update sensitive data rule: %w", err))
	}
	return resourceSensitiveDataRuleRead(ctx, d, m)
}

func resourceSensitiveDataRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region, id, err := parseRegionID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteSensitiveData(id, region); err != nil {
		return diag.FromErr(fmt.Errorf("delete sensitive data rule: %w", err))
	}
	d.SetId("")
	return nil
}

func buildSensitiveDataRequest(d *schema.ResourceData) *client.SensitiveDataRequest {
	req := &client.SensitiveDataRequest{
		Name: d.Get("name").(string),
		Properties: client.SensitiveDataProperties{
			Telemetry: d.Get("telemetry").(string),
			Order:     int64(d.Get("order").(int)),
			ScanRules: client.SensitiveDataScanRules{
				Email:            d.Get("scan_email").(bool),
				PhoneNumber:      d.Get("scan_phone_number").(bool),
				CreditCardNumber: d.Get("scan_credit_card").(bool),
			},
			Action: client.SensitiveDataAction{
				Name: d.Get("action_name").(string),
			},
		},
	}
	if labels, ok := d.GetOk("labels"); ok {
		req.Properties.Labels = expandStringMap(labels.(map[string]interface{}))
	}
	return req
}

func parseRegionID(id string) (region, resourceID string, err error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid ID %q; expected region:id", id)
	}
	return parts[0], parts[1], nil
}
