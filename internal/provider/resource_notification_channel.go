package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func resourceNotificationChannel() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNotificationChannelCreate,
		ReadContext:   resourceNotificationChannelRead,
		UpdateContext: resourceNotificationChannelUpdate,
		DeleteContext: resourceNotificationChannelDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Channel name (colons are not allowed)",
				ValidateFunc: validation.StringDoesNotContainAny(":"),
			},
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Notification type: slack, pagerduty, opsgenie, email, generic_webhook",
				ValidateFunc: validation.StringInSlice([]string{
					"slack",
					"pagerduty",
					"opsgenie",
					"email",
					"generic_webhook",
				}, false),
			},
			"destination": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "Destination address: webhook URL, email address, or API key depending on type",
			},
			"send_resolved": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether to send resolved notifications",
			},
			// Computed fields
			"global": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether this is a global (master) channel",
			},
			"in_use": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the channel has any attachments",
			},
			"organization_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Organization ID",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation timestamp",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Last update timestamp",
			},
		},
	}
}

func resourceNotificationChannelCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	req := &client.NotificationChannelRequest{
		Name:         d.Get("name").(string),
		Type:         d.Get("type").(string),
		Destination:  d.Get("destination").(string),
		SendResolved: d.Get("send_resolved").(bool),
	}

	channel, err := apiClient.CreateNotificationDestination(req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create notification channel: %w", err))
	}

	d.SetId(strconv.Itoa(channel.ID))

	return resourceNotificationChannelRead(ctx, d, m)
}

func resourceNotificationChannelRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid notification channel ID: %w", err))
	}

	channel, err := apiClient.GetNotificationDestination(id)
	if err != nil {
		// Check if the error indicates the resource was not found
		if strings.Contains(err.Error(), "not found") {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to read notification channel: %w", err))
	}

	d.Set("name", channel.Name)
	d.Set("type", channel.Type)
	d.Set("destination", channel.Destination)
	d.Set("send_resolved", channel.SendResolved)
	d.Set("global", channel.Global)
	d.Set("in_use", channel.InUse)
	d.Set("organization_id", channel.OrganizationID)
	d.Set("created_at", channel.CreatedAt)
	d.Set("updated_at", channel.UpdatedAt)

	return nil
}

func resourceNotificationChannelUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid notification channel ID: %w", err))
	}

	req := &client.NotificationChannelRequest{
		Name:         d.Get("name").(string),
		Type:         d.Get("type").(string),
		Destination:  d.Get("destination").(string),
		SendResolved: d.Get("send_resolved").(bool),
	}

	_, err = apiClient.UpdateNotificationDestination(id, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update notification channel: %w", err))
	}

	return resourceNotificationChannelRead(ctx, d, m)
}

func resourceNotificationChannelDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid notification channel ID: %w", err))
	}

	err = apiClient.DeleteNotificationDestination(id)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete notification channel: %w", err))
	}

	d.SetId("")
	return nil
}
