package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func resourceAlertSnooze() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAlertSnoozeCreate,
		ReadContext:   resourceAlertSnoozeRead,
		UpdateContext: resourceAlertSnoozeUpdate,
		DeleteContext: resourceAlertSnoozeDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"entity_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Entity (alert group) ID to snooze",
			},
			"until": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Unix timestamp until which alerts are snoozed. Set to 0 to clear.",
			},
			"alert_snoozed_until": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Effective snooze end timestamp from the API",
			},
		},
	}
}

func resourceAlertSnoozeCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	entityID := d.Get("entity_id").(string)
	until := d.Get("until").(int)

	if _, err := c.UpsertEntitySnooze(entityID, until); err != nil {
		return diag.FromErr(fmt.Errorf("create alert snooze: %w", err))
	}
	d.SetId(entityID)
	return resourceAlertSnoozeRead(ctx, d, m)
}

func resourceAlertSnoozeRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	entityID := d.Id()

	resp, err := c.GetEntitySnooze(entityID)
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("read alert snooze: %w", err))
	}

	_ = d.Set("entity_id", entityID)
	_ = d.Set("alert_snoozed_until", resp.AlertSnoozedUntil)
	// Keep until in state; if API reports 0 and we expected a future value, drift will show.
	if _, ok := d.GetOk("until"); !ok {
		_ = d.Set("until", resp.AlertSnoozedUntil)
	}
	return nil
}

func resourceAlertSnoozeUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	entityID := d.Id()
	until := d.Get("until").(int)

	if _, err := c.UpsertEntitySnooze(entityID, until); err != nil {
		return diag.FromErr(fmt.Errorf("update alert snooze: %w", err))
	}
	return resourceAlertSnoozeRead(ctx, d, m)
}

func resourceAlertSnoozeDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	// Clear snooze by setting until=0
	if _, err := c.UpsertEntitySnooze(d.Id(), 0); err != nil {
		return diag.FromErr(fmt.Errorf("clear alert snooze: %w", err))
	}
	d.SetId("")
	return nil
}
