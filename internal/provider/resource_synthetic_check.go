package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func resourceSyntheticCheck() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSyntheticCheckCreate,
		ReadContext:   resourceSyntheticCheckRead,
		UpdateContext: resourceSyntheticCheckUpdate,
		DeleteContext: resourceSyntheticCheckDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the synthetic check",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the synthetic check",
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"http", "https", "api", "script", "tcp", "dns", "icmp",
				}, false),
				Description: "Check type: http, https, api, script, tcp, dns, or icmp",
			},
			"schedule": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Cron expression (e.g. '*/5 * * * *') or interval (e.g. 'every 5m')",
			},
			"config": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "JSON config for the check (type-specific: url, host, script, etc.)",
				ValidateFunc:     validation.StringIsJSON,
				DiffSuppressFunc: suppressEquivalentJSON,
			},
			"timeout": {
				Type:         schema.TypeInt,
				Required:     true,
				Description:  "Timeout in seconds (1-300)",
				ValidateFunc: validation.IntBetween(1, 300),
			},
			"frequency": {
				Type:         schema.TypeInt,
				Required:     true,
				Description:  "Frequency in seconds (minimum 60)",
				ValidateFunc: validation.IntAtLeast(60),
			},
			"locations": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: "Locations to run the check from",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"tags": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Tags for the check",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"status": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ValidateFunc: validation.StringInSlice([]string{
					"active", "paused",
				}, false),
				Description: "Check status: active or paused",
			},
			"created_by": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceSyntheticCheckCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)

	req, err := buildSyntheticCheckCreateRequest(d)
	if err != nil {
		return diag.FromErr(err)
	}

	check, err := c.CreateSyntheticCheck(req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("create synthetic check: %w", err))
	}

	d.SetId(check.ID)
	return resourceSyntheticCheckRead(ctx, d, m)
}

func resourceSyntheticCheckRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)

	check, err := c.GetSyntheticCheck(d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("read synthetic check: %w", err))
	}

	_ = d.Set("name", check.Name)
	_ = d.Set("description", check.Description)
	_ = d.Set("type", check.Type)
	_ = d.Set("schedule", check.Schedule)
	_ = d.Set("timeout", check.Timeout)
	_ = d.Set("frequency", check.Frequency)
	_ = d.Set("locations", check.Locations)
	_ = d.Set("status", check.Status)
	_ = d.Set("created_by", check.CreatedBy)
	if check.Tags != nil {
		_ = d.Set("tags", check.Tags)
	}
	if len(check.Config) > 0 {
		apiConfig := string(check.Config)
		if planned, ok := d.GetOk("config"); ok {
			if suppressEquivalentJSON("", planned.(string), apiConfig, d) {
				_ = d.Set("config", planned.(string))
			} else {
				_ = d.Set("config", apiConfig)
			}
		} else {
			_ = d.Set("config", apiConfig)
		}
	}
	return nil
}

func resourceSyntheticCheckUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)

	req := &client.UpdateSyntheticCheckRequest{}
	if d.HasChange("name") {
		v := d.Get("name").(string)
		req.Name = &v
	}
	if d.HasChange("description") {
		v := d.Get("description").(string)
		req.Description = &v
	}
	if d.HasChange("status") {
		v := d.Get("status").(string)
		req.Status = &v
	}
	if d.HasChange("schedule") {
		v := d.Get("schedule").(string)
		req.Schedule = &v
	}
	if d.HasChange("config") {
		raw := json.RawMessage(d.Get("config").(string))
		req.Config = &raw
	}
	if d.HasChange("timeout") {
		v := d.Get("timeout").(int)
		req.Timeout = &v
	}
	if d.HasChange("frequency") {
		v := d.Get("frequency").(int)
		req.Frequency = &v
	}
	if d.HasChange("locations") {
		locs := expandStringList(d.Get("locations").([]interface{}))
		req.Locations = &locs
	}
	if d.HasChange("tags") {
		tags := expandStringMap(d.Get("tags").(map[string]interface{}))
		req.Tags = &tags
	}

	if _, err := c.UpdateSyntheticCheck(d.Id(), req); err != nil {
		return diag.FromErr(fmt.Errorf("update synthetic check: %w", err))
	}
	return resourceSyntheticCheckRead(ctx, d, m)
}

func resourceSyntheticCheckDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	if err := c.DeleteSyntheticCheck(d.Id()); err != nil {
		return diag.FromErr(fmt.Errorf("delete synthetic check: %w", err))
	}
	d.SetId("")
	return nil
}

func buildSyntheticCheckCreateRequest(d *schema.ResourceData) (*client.CreateSyntheticCheckRequest, error) {
	configStr := d.Get("config").(string)
	if !json.Valid([]byte(configStr)) {
		return nil, fmt.Errorf("config must be valid JSON")
	}

	req := &client.CreateSyntheticCheckRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Type:        d.Get("type").(string),
		Schedule:    d.Get("schedule").(string),
		Config:      json.RawMessage(configStr),
		Timeout:     d.Get("timeout").(int),
		Frequency:   d.Get("frequency").(int),
		Locations:   expandStringList(d.Get("locations").([]interface{})),
	}
	if tags, ok := d.GetOk("tags"); ok {
		req.Tags = expandStringMap(tags.(map[string]interface{}))
	}
	return req, nil
}

func expandStringMap(in map[string]interface{}) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "404") || strings.Contains(msg, "not found")
}

func suppressEquivalentJSON(k, old, new string, d *schema.ResourceData) bool {
	if old == "" || new == "" {
		return old == new
	}
	var o, n interface{}
	if err := json.Unmarshal([]byte(old), &o); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(new), &n); err != nil {
		return false
	}
	ob, _ := json.Marshal(o)
	nb, _ := json.Marshal(n)
	return string(ob) == string(nb)
}
