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
			"headers": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Custom HTTP headers to send with webhook requests. Only applicable for generic_webhook type.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"slack_app_mode": {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Description: "When true, deliver via the Last9 Slack App (bot token + chat.postMessage) and treat destination as a Slack channel ID (e.g. C0123456789). When false or unset, destination must be a https://hooks.slack.com/ webhook URL. Mode cannot be changed after the channel is created. Only applicable for slack type. New Slack webhook channels are no longer accepted by the API; new Slack channels must set slack_app_mode = true.",
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
		CustomizeDiff: validateNotificationChannel,
	}
}

// validateNotificationChannel enforces type-specific destination/option constraints.
func validateNotificationChannel(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
	channelType := d.Get("type").(string)
	headers := d.Get("headers").(map[string]interface{})
	slackAppMode := d.Get("slack_app_mode").(bool)
	destination := d.Get("destination").(string)

	if len(headers) > 0 && channelType != "generic_webhook" {
		return fmt.Errorf("headers can only be specified for generic_webhook type, got type: %s", channelType)
	}

	if slackAppMode && channelType != "slack" {
		return fmt.Errorf("slack_app_mode can only be set for slack type, got type: %s", channelType)
	}

	if channelType == "slack" && destination != "" {
		if slackAppMode {
			if strings.HasPrefix(destination, "http") {
				return fmt.Errorf("slack_app_mode requires a Slack channel ID as destination (e.g. C0123456789), not a webhook URL")
			}
		} else {
			if !strings.HasPrefix(destination, "https://hooks.slack.com/") {
				return fmt.Errorf("slack webhook destination must be a valid https://hooks.slack.com/ URL; set slack_app_mode = true to use a Slack App channel ID instead")
			}
		}
	}

	return nil
}

// buildProperty creates the Property struct for channel-specific options.
// Returns nil when there are no options to send so the API payload stays minimal.
func buildProperty(d *schema.ResourceData) *client.NotificationSettingProperty {
	channelType := d.Get("type").(string)

	prop := &client.NotificationSettingProperty{}
	hasValue := false

	if channelType == "generic_webhook" {
		headersRaw := d.Get("headers").(map[string]interface{})
		if len(headersRaw) > 0 {
			headers := make(map[string]string, len(headersRaw))
			for k, v := range headersRaw {
				headers[k] = v.(string)
			}
			prop.WebhookHeaders = headers
			hasValue = true
		}
	}

	if channelType == "slack" && d.Get("slack_app_mode").(bool) {
		prop.SlackAppMode = true
		hasValue = true
	}

	if !hasValue {
		return nil
	}
	return prop
}

func resourceNotificationChannelCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	req := &client.NotificationChannelRequest{
		Name:         d.Get("name").(string),
		Type:         d.Get("type").(string),
		Destination:  d.Get("destination").(string),
		SendResolved: d.Get("send_resolved").(bool),
		Property:     buildProperty(d),
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

	// Extract webhook headers from property if this is a webhook channel
	if channel.Type == "generic_webhook" && channel.Property != nil {
		if webhookHeaders, ok := channel.Property["webhook_headers"]; ok {
			if headersMap, ok := webhookHeaders.(map[string]interface{}); ok {
				headers := make(map[string]string)
				for k, v := range headersMap {
					if strVal, ok := v.(string); ok {
						headers[k] = strVal
					}
				}
				if len(headers) > 0 {
					d.Set("headers", headers)
				}
			}
		}
	}

	// Extract slack_app_mode flag for slack channels
	if channel.Type == "slack" && channel.Property != nil {
		if appMode, ok := channel.Property["slack_app_mode"].(bool); ok {
			d.Set("slack_app_mode", appMode)
		}
	}

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
		Property:     buildProperty(d),
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
