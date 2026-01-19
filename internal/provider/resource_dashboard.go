package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func resourceDashboard() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDashboardCreate,
		ReadContext:   resourceDashboardRead,
		UpdateContext: resourceDashboardUpdate,
		DeleteContext: resourceDashboardDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Dashboard name",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Dashboard description",
			},
			"readonly": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether the dashboard is read-only",
			},
			"panels": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of dashboard panels",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"title": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Panel title",
						},
						"query": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Panel query",
						},
						"visualization": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Panel visualization type",
						},
						"config": {
							Type:        schema.TypeMap,
							Optional:    true,
							Description: "Panel configuration",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"tags": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of tags",
				Elem:        &schema.Schema{Type: schema.TypeString},
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

func resourceDashboardCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	req := &client.DashboardCreateRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Readonly:    d.Get("readonly").(bool),
	}

	if v, ok := d.GetOk("panels"); ok {
		panels := expandDashboardPanels(v.([]interface{}))
		req.Panels = panels
	}

	if v, ok := d.GetOk("tags"); ok {
		tags := expandStringList(v.([]interface{}))
		req.Tags = tags
	}

	dashboard, err := apiClient.CreateDashboard(req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create dashboard: %w", err))
	}

	d.SetId(dashboard.ID)
	return resourceDashboardRead(ctx, d, m)
}

func resourceDashboardRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	dashboard, err := apiClient.GetDashboard(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read dashboard: %w", err))
	}

	d.Set("name", dashboard.Name)
	d.Set("description", dashboard.Description)
	d.Set("readonly", dashboard.Readonly)
	d.Set("panels", flattenDashboardPanels(dashboard.Panels))
	d.Set("tags", dashboard.Tags)
	d.Set("created_at", dashboard.CreatedAt)
	d.Set("updated_at", dashboard.UpdatedAt)

	return nil
}

func resourceDashboardUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	req := &client.DashboardUpdateRequest{}

	if d.HasChange("name") {
		req.Name = d.Get("name").(string)
	}
	if d.HasChange("description") {
		req.Description = d.Get("description").(string)
	}
	if d.HasChange("readonly") {
		readonly := d.Get("readonly").(bool)
		req.Readonly = &readonly
	}
	if d.HasChange("panels") {
		panels := expandDashboardPanels(d.Get("panels").([]interface{}))
		req.Panels = panels
	}
	if d.HasChange("tags") {
		tags := expandStringList(d.Get("tags").([]interface{}))
		req.Tags = tags
	}

	_, err := apiClient.UpdateDashboard(d.Id(), req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update dashboard: %w", err))
	}

	return resourceDashboardRead(ctx, d, m)
}

func resourceDashboardDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	err := apiClient.DeleteDashboard(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete dashboard: %w", err))
	}

	d.SetId("")
	return nil
}

func expandDashboardPanels(panels []interface{}) []client.DashboardPanel {
	result := make([]client.DashboardPanel, 0, len(panels))
	for _, p := range panels {
		panelMap := p.(map[string]interface{})
		panel := client.DashboardPanel{
			Title:         panelMap["title"].(string),
			Query:         panelMap["query"].(string),
			Visualization: panelMap["visualization"].(string),
		}
		if config, ok := panelMap["config"].(map[string]interface{}); ok {
			panel.Config = make(map[string]interface{})
			for k, v := range config {
				panel.Config[k] = v
			}
		}
		result = append(result, panel)
	}
	return result
}

func flattenDashboardPanels(panels []client.DashboardPanel) []interface{} {
	result := make([]interface{}, 0, len(panels))
	for _, p := range panels {
		panelMap := map[string]interface{}{
			"title":         p.Title,
			"query":         p.Query,
			"visualization": p.Visualization,
		}
		if p.Config != nil {
			config := make(map[string]interface{})
			for k, v := range p.Config {
				config[k] = v
			}
			panelMap["config"] = config
		}
		result = append(result, panelMap)
	}
	return result
}

func expandStringList(list []interface{}) []string {
	result := make([]string, 0, len(list))
	for _, v := range list {
		result = append(result, v.(string))
	}
	return result
}
