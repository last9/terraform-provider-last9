package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func resourceMacro() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMacroCreate,
		ReadContext:   resourceMacroRead,
		UpdateContext: resourceMacroUpdate,
		DeleteContext: resourceMacroDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"cluster_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Cluster ID",
			},
			"body": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Macro configuration as JSON string",
				ValidateFunc: validateJSON,
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation timestamp",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Update timestamp",
			},
		},
	}
}

func resourceMacroCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	clusterID := d.Get("cluster_id").(string)

	req := &client.MacroUpsertRequest{
		Body: d.Get("body").(string),
	}

	macro, err := apiClient.UpsertMacro(clusterID, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create macro: %w", err))
	}

	d.SetId(macro.ID)
	return resourceMacroRead(ctx, d, m)
}

func resourceMacroRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	clusterID := d.Get("cluster_id").(string)

	macro, err := apiClient.GetMacro(clusterID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read macro: %w", err))
	}

	d.Set("cluster_id", macro.ClusterID)
	d.Set("body", macro.Body)
	d.Set("created_at", macro.CreatedAt)
	d.Set("updated_at", macro.UpdatedAt)

	return nil
}

func resourceMacroUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	clusterID := d.Get("cluster_id").(string)

	req := &client.MacroUpsertRequest{
		Body: d.Get("body").(string),
	}

	macro, err := apiClient.UpsertMacro(clusterID, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update macro: %w", err))
	}

	d.SetId(macro.ID)
	return resourceMacroRead(ctx, d, m)
}

func resourceMacroDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	clusterID := d.Get("cluster_id").(string)

	err := apiClient.DeleteMacro(clusterID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete macro: %w", err))
	}

	d.SetId("")
	return nil
}

func validateJSON(val interface{}, key string) (warns []string, errs []error) {
	body := val.(string)
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		errs = append(errs, fmt.Errorf("invalid JSON: %w", err))
	}
	return warns, errs
}
