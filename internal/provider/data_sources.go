package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/last9/terraform-provider-last9/internal/client"
)

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
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Entity description",
			},
			"data_source": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Data source name",
			},
			"data_source_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Data source ID",
			},
			"namespace": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Entity namespace",
			},
			"team": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Owning team",
			},
			"tier": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Entity tier",
			},
			"workspace": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Workspace",
			},
			"tags": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Array of tags",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "Key-value pairs for group labels",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"entity_class": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Entity classification",
			},
			"ui_readonly": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether UI edits are disabled",
			},
		},
	}
}

func dataSourceEntityRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	var entity *client.Entity
	var err error

	if id, ok := d.GetOk("id"); ok {
		entity, err = apiClient.GetEntity(id.(string))
	} else if externalRef, ok := d.GetOk("external_ref"); ok {
		entity, err = apiClient.GetEntityByExternalRef(externalRef.(string))
	} else {
		return diag.FromErr(fmt.Errorf("either id or external_ref must be provided"))
	}

	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read entity: %w", err))
	}

	d.SetId(entity.ID)
	d.Set("name", entity.Name)
	d.Set("type", entity.Type)
	d.Set("external_ref", entity.ExternalRef)
	d.Set("description", entity.Description)
	d.Set("data_source", entity.DataSource)
	d.Set("data_source_id", entity.DataSourceID)
	d.Set("namespace", entity.Namespace)
	d.Set("team", entity.Team)
	d.Set("tier", entity.Tier)
	d.Set("workspace", entity.Workspace)
	d.Set("entity_class", entity.EntityClass)
	d.Set("ui_readonly", entity.UIReadonly)
	d.Set("tags", entity.Tags)
	d.Set("labels", entity.Labels)

	return nil
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
		for _, dest := range result {
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
		for _, dest := range result {
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

func dataSourceCluster() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceClusterRead,
		Schema: map[string]*schema.Schema{
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Region to look up clusters in",
			},
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Cluster ID. If omitted with name, returns the default cluster.",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Cluster name",
			},
			"default": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether this is the default cluster for the region",
			},
		},
	}
}

func dataSourceClusterRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	region := d.Get("region").(string)

	clusters, err := c.GetClusters(region)
	if err != nil {
		return diag.FromErr(fmt.Errorf("list clusters: %w", err))
	}

	var found *client.Cluster
	if id, ok := d.GetOk("id"); ok {
		for i := range clusters {
			if clusters[i].ID == id.(string) {
				found = &clusters[i]
				break
			}
		}
		if found == nil {
			return diag.FromErr(fmt.Errorf("cluster id %q not found in region %s", id, region))
		}
	} else if name, ok := d.GetOk("name"); ok {
		for i := range clusters {
			if clusters[i].Name == name.(string) {
				found = &clusters[i]
				break
			}
		}
		if found == nil {
			return diag.FromErr(fmt.Errorf("cluster name %q not found in region %s", name, region))
		}
	} else {
		def, err := c.GetDefaultCluster(region)
		if err != nil {
			return diag.FromErr(err)
		}
		found = def
	}

	d.SetId(found.ID)
	_ = d.Set("id", found.ID)
	_ = d.Set("name", found.Name)
	_ = d.Set("region", found.Region)
	_ = d.Set("default", found.IsDefault)
	return nil
}

func dataSourceDatasource() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDatasourceRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"region": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"default": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func dataSourceDatasourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)

	list, err := c.ListDatasources()
	if err != nil {
		return diag.FromErr(fmt.Errorf("list datasources: %w", err))
	}

	var found *client.Datasource
	if id, ok := d.GetOk("id"); ok {
		for i := range list {
			if list[i].ID == id.(string) {
				found = &list[i]
				break
			}
		}
		if found == nil {
			return diag.FromErr(fmt.Errorf("datasource id %q not found", id))
		}
	} else if name, ok := d.GetOk("name"); ok {
		for i := range list {
			if list[i].Name == name.(string) {
				found = &list[i]
				break
			}
		}
		if found == nil {
			return diag.FromErr(fmt.Errorf("datasource name %q not found", name))
		}
	} else {
		for i := range list {
			if list[i].Default {
				found = &list[i]
				break
			}
		}
		if found == nil && len(list) > 0 {
			found = &list[0]
		}
		if found == nil {
			return diag.FromErr(fmt.Errorf("no datasources found"))
		}
	}

	d.SetId(found.ID)
	_ = d.Set("id", found.ID)
	_ = d.Set("name", found.Name)
	_ = d.Set("type", found.Type)
	_ = d.Set("region", found.Region)
	_ = d.Set("default", found.Default)
	return nil
}

func dataSourceUser() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceUserRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "User ID",
			},
			"email": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "User email",
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"role": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"active": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"organization_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)

	var user *client.User
	var err error
	if id, ok := d.GetOk("id"); ok {
		user, err = c.FindUserByID(id.(string))
	} else if email, ok := d.GetOk("email"); ok {
		user, err = c.FindUserByEmail(email.(string))
	} else {
		return diag.FromErr(fmt.Errorf("either id or email must be provided"))
	}
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(user.ID)
	_ = d.Set("id", user.ID)
	_ = d.Set("email", user.Email)
	_ = d.Set("name", user.Name)
	_ = d.Set("role", user.Role)
	_ = d.Set("status", user.Status)
	_ = d.Set("organization_id", user.OrganizationID)
	_ = d.Set("active", user.DeletedAt == nil)
	return nil
}
