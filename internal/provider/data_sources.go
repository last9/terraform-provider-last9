package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func dataSourceDashboard() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDashboardRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Dashboard ID",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Dashboard name",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Dashboard description",
			},
			"readonly": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the dashboard is read-only",
			},
			"panels": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of dashboard panels",
			},
			"tags": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of tags",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceDashboardRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	var dashboard *client.Dashboard
	var err error

	if id, ok := d.GetOk("id"); ok {
		dashboard, err = apiClient.GetDashboard(id.(string))
	} else if _, ok := d.GetOk("name"); ok {
		// List dashboards and find by name
		// This would require a list endpoint implementation
		return diag.FromErr(fmt.Errorf("searching by name not yet implemented, please use id"))
	} else {
		return diag.FromErr(fmt.Errorf("either id or name must be provided"))
	}

	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read dashboard: %w", err))
	}

	d.SetId(dashboard.ID)
	d.Set("name", dashboard.Name)
	d.Set("description", dashboard.Description)
	d.Set("readonly", dashboard.Readonly)
	d.Set("tags", dashboard.Tags)

	return nil
}

func dataSourceEntity() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceEntityRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Entity ID",
			},
			"external_ref": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Entity external reference",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Entity name",
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Entity type",
			},
		},
	}
}

func dataSourceEntityRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Implementation would query entity by ID or external_ref
	// This is a placeholder
	return diag.FromErr(fmt.Errorf("data source entity not yet implemented"))
}

func dataSourceNotificationDestination() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceNotificationDestinationRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Notification destination ID",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Notification destination name",
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Destination type (email, slack, pagerduty, webhook, etc.)",
			},
			"destination": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Destination address (email, webhook URL, etc.)",
			},
			"global": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether destination is globally available",
			},
			"send_resolved": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether to send resolved notifications",
			},
			"in_use": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether destination is currently in use",
			},
		},
	}
}

func dataSourceNotificationDestinationRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	// List all notification destinations
	result, err := apiClient.ListNotificationDestinations()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to list notification destinations: %w", err))
	}

	var destination *client.NotificationDestination

	// Search by ID or name
	if id, ok := d.GetOk("id"); ok {
		idInt := id.(int)
		for _, dest := range result.NotificationDestinations {
			if dest.ID == idInt {
				destination = &dest
				break
			}
		}
		if destination == nil {
			return diag.FromErr(fmt.Errorf("notification destination with ID %d not found", idInt))
		}
	} else if name, ok := d.GetOk("name"); ok {
		nameStr := name.(string)
		for _, dest := range result.NotificationDestinations {
			if dest.Name == nameStr {
				destination = &dest
				break
			}
		}
		if destination == nil {
			return diag.FromErr(fmt.Errorf("notification destination with name '%s' not found", nameStr))
		}
	} else {
		return diag.FromErr(fmt.Errorf("either id or name must be provided"))
	}

	// Set resource data
	d.SetId(fmt.Sprintf("%d", destination.ID))
	d.Set("name", destination.Name)
	d.Set("type", destination.Type)
	d.Set("destination", destination.Destination)
	d.Set("global", destination.Global)
	d.Set("send_resolved", destination.SendResolved)
	d.Set("in_use", destination.InUse)

	return nil
}
