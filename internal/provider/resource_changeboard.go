package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func resourceChangeboard() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceChangeboardCreate,
		ReadContext:   resourceChangeboardRead,
		UpdateContext: resourceChangeboardUpdate,
		DeleteContext: resourceChangeboardDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Changeboard name",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Changeboard description",
			},
			"owner_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Owner ID (user, organization, or team UUID)",
			},
			"owner_type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					"user", "organization", "team",
				}, false),
				Description: "Owner type: user, organization, or team",
			},
			"filter": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Entity filters defining which entities belong to this changeboard",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"filter_type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"operator": {
							Type:     schema.TypeString,
							Required: true,
						},
						"conjunction": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"group": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Entity grouping dimensions",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"order": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"relationship": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Hierarchy of entity types (id = entity type, one nesting level)",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Entity type at this hierarchy node",
						},
						"child_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Nested child entity type (one level)",
						},
					},
				},
			},
			"granularity": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Changeboard time granularity",
			},
			"created_at": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"updated_at": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func resourceChangeboardCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	req := buildChangeBoardRequest(d)

	cb, err := c.CreateChangeBoard(req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("create changeboard: %w", err))
	}
	d.SetId(cb.ID)
	return resourceChangeboardRead(ctx, d, m)
}

func resourceChangeboardRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)

	cb, err := c.GetChangeBoard(d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("read changeboard: %w", err))
	}

	_ = d.Set("name", cb.Name)
	_ = d.Set("description", cb.Description)
	_ = d.Set("owner_id", cb.OwnerID)
	_ = d.Set("owner_type", cb.OwnerType)
	_ = d.Set("granularity", cb.Properties.Granularity)
	_ = d.Set("created_at", cb.CreatedAt)
	_ = d.Set("updated_at", cb.UpdatedAt)
	_ = d.Set("filter", flattenChangeBoardFilters(cb.Filters))
	_ = d.Set("group", flattenChangeBoardGroups(cb.Groups))
	_ = d.Set("relationship", flattenChangeBoardRelationships(cb.Relationships))
	return nil
}

func resourceChangeboardUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	req := buildChangeBoardRequest(d)

	if _, err := c.UpdateChangeBoard(d.Id(), req); err != nil {
		return diag.FromErr(fmt.Errorf("update changeboard: %w", err))
	}
	return resourceChangeboardRead(ctx, d, m)
}

func resourceChangeboardDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	if err := c.DeleteChangeBoard(d.Id()); err != nil {
		return diag.FromErr(fmt.Errorf("delete changeboard: %w", err))
	}
	d.SetId("")
	return nil
}

func buildChangeBoardRequest(d *schema.ResourceData) *client.ChangeBoardRequest {
	req := &client.ChangeBoardRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		OwnerID:     d.Get("owner_id").(string),
		OwnerType:   d.Get("owner_type").(string),
		Filters:     expandChangeBoardFilters(d.Get("filter").([]interface{})),
		Groups:      expandChangeBoardGroups(d.Get("group").([]interface{})),
	}
	if req.Filters == nil {
		req.Filters = []client.ChangeBoardFilter{}
	}
	if req.Groups == nil {
		req.Groups = []client.ChangeBoardGroup{}
	}
	if rels := expandChangeBoardRelationships(d.Get("relationship").([]interface{})); len(rels) > 0 {
		req.Relationships = rels
	}
	if g, ok := d.GetOk("granularity"); ok && g.(string) != "" {
		req.Properties = &client.ChangeBoardProperties{Granularity: g.(string)}
	}
	return req
}

func expandChangeBoardFilters(in []interface{}) []client.ChangeBoardFilter {
	out := make([]client.ChangeBoardFilter, 0, len(in))
	for _, raw := range in {
		m := raw.(map[string]interface{})
		f := client.ChangeBoardFilter{
			FilterType: m["filter_type"].(string),
			Key:        m["key"].(string),
			Value:      m["value"].(string),
			Operator:   m["operator"].(string),
		}
		if c, ok := m["conjunction"].(string); ok && c != "" {
			f.Conjunction = &c
		}
		out = append(out, f)
	}
	return out
}

func flattenChangeBoardFilters(in []client.ChangeBoardFilter) []interface{} {
	out := make([]interface{}, 0, len(in))
	for _, f := range in {
		m := map[string]interface{}{
			"filter_type": f.FilterType,
			"key":         f.Key,
			"value":       f.Value,
			"operator":    f.Operator,
		}
		if f.Conjunction != nil {
			m["conjunction"] = *f.Conjunction
		}
		out = append(out, m)
	}
	return out
}

func expandChangeBoardGroups(in []interface{}) []client.ChangeBoardGroup {
	out := make([]client.ChangeBoardGroup, 0, len(in))
	for _, raw := range in {
		m := raw.(map[string]interface{})
		out = append(out, client.ChangeBoardGroup{
			Name:  m["name"].(string),
			Order: m["order"].(string),
		})
	}
	return out
}

func flattenChangeBoardGroups(in []client.ChangeBoardGroup) []interface{} {
	out := make([]interface{}, 0, len(in))
	for _, g := range in {
		out = append(out, map[string]interface{}{
			"name":  g.Name,
			"order": g.Order,
		})
	}
	return out
}

func expandChangeBoardRelationships(in []interface{}) []client.ChangeBoardRelationshipNode {
	out := make([]client.ChangeBoardRelationshipNode, 0, len(in))
	for _, raw := range in {
		m := raw.(map[string]interface{})
		node := client.ChangeBoardRelationshipNode{ID: m["id"].(string)}
		if child, ok := m["child_id"].(string); ok && child != "" {
			node.Children = []client.ChangeBoardRelationshipNode{{ID: child}}
		}
		out = append(out, node)
	}
	return out
}

func flattenChangeBoardRelationships(in []client.ChangeBoardRelationshipNode) []interface{} {
	out := make([]interface{}, 0, len(in))
	for _, n := range in {
		m := map[string]interface{}{"id": n.ID}
		if len(n.Children) > 0 {
			m["child_id"] = n.Children[0].ID
		}
		out = append(out, m)
	}
	return out
}
