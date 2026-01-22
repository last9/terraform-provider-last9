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

func resourceNotificationChannelAttachment() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNotificationChannelAttachmentCreate,
		ReadContext:   resourceNotificationChannelAttachmentRead,
		UpdateContext: resourceNotificationChannelAttachmentUpdate,
		DeleteContext: resourceNotificationChannelAttachmentDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceNotificationChannelAttachmentImport,
		},
		Schema: map[string]*schema.Schema{
			"channel_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The notification channel ID (master channel)",
			},
			"entity_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The entity/alert-group UUID to attach to",
			},
			"severity": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Severity level: breach or threat",
				ValidateFunc: validation.StringInSlice([]string{
					"breach",
					"threat",
				}, false),
			},
			// Computed field - the ID of the child notification destination created
			"child_channel_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The ID of the child notification channel created by the attachment",
			},
		},
	}
}

// buildAttachmentID creates a composite ID from channel_id, entity_id, and severity
func buildAttachmentID(channelID int, entityID, severity string) string {
	return fmt.Sprintf("%d:%s:%s", channelID, entityID, severity)
}

// parseAttachmentID parses a composite ID into its components
func parseAttachmentID(id string) (channelID int, entityID, severity string, err error) {
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return 0, "", "", fmt.Errorf("invalid attachment ID format, expected channel_id:entity_id:severity")
	}

	channelID, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", "", fmt.Errorf("invalid channel_id in attachment ID: %w", err)
	}

	return channelID, parts[1], parts[2], nil
}

func resourceNotificationChannelAttachmentCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	channelID := d.Get("channel_id").(int)
	entityID := d.Get("entity_id").(string)
	severity := d.Get("severity").(string)

	req := &client.NotificationChannelAttachRequest{
		EntityID: entityID,
		Severity: severity,
	}

	result, err := apiClient.AttachNotificationChannel(channelID, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to attach notification channel: %w", err))
	}

	// Store the child channel ID if available
	if result != nil {
		d.Set("child_channel_id", result.ID)
	}

	d.SetId(buildAttachmentID(channelID, entityID, severity))

	return resourceNotificationChannelAttachmentRead(ctx, d, m)
}

func resourceNotificationChannelAttachmentRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	channelID, entityID, severity, err := parseAttachmentID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// List all notification destinations to find the child channel
	destinations, err := apiClient.ListNotificationDestinations()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to list notification destinations: %w", err))
	}

	// Find the child channel that matches our attachment criteria
	// Child channels have the entity_id set (via ServiceFqid or similar) and matching severity
	var found *client.NotificationDestination
	for _, dest := range destinations {
		// Child channels are linked to specific entities
		// The master channel has Global=true, children have the entity association
		if !dest.Global && dest.ServiceFqid == entityID && dest.Severity == severity {
			// Check if this is a child of our master channel by name pattern or other identifier
			found = &dest
			break
		}
	}

	if found == nil {
		// Attachment no longer exists
		d.SetId("")
		return nil
	}

	d.Set("channel_id", channelID)
	d.Set("entity_id", entityID)
	d.Set("severity", severity)
	d.Set("child_channel_id", found.ID)

	return nil
}

func resourceNotificationChannelAttachmentUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Only severity can be updated (channel_id and entity_id are ForceNew)
	// To update severity, we need to detach and re-attach
	if d.HasChange("severity") {
		apiClient := m.(*client.Client)

		channelID := d.Get("channel_id").(int)
		entityID := d.Get("entity_id").(string)
		oldSeverity, newSeverity := d.GetChange("severity")

		// Detach the old attachment
		err := apiClient.DetachNotificationChannel(channelID, entityID)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to detach notification channel during update: %w", err))
		}

		// Attach with new severity
		req := &client.NotificationChannelAttachRequest{
			EntityID: entityID,
			Severity: newSeverity.(string),
		}

		result, err := apiClient.AttachNotificationChannel(channelID, req)
		if err != nil {
			// Try to restore the old attachment
			restoreReq := &client.NotificationChannelAttachRequest{
				EntityID: entityID,
				Severity: oldSeverity.(string),
			}
			apiClient.AttachNotificationChannel(channelID, restoreReq)
			return diag.FromErr(fmt.Errorf("failed to re-attach notification channel with new severity: %w", err))
		}

		if result != nil {
			d.Set("child_channel_id", result.ID)
		}

		// Update the ID to reflect new severity
		d.SetId(buildAttachmentID(channelID, entityID, newSeverity.(string)))
	}

	return resourceNotificationChannelAttachmentRead(ctx, d, m)
}

func resourceNotificationChannelAttachmentDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	channelID, entityID, _, err := parseAttachmentID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = apiClient.DetachNotificationChannel(channelID, entityID)
	if err != nil {
		// Check if already deleted
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to detach notification channel: %w", err))
	}

	d.SetId("")
	return nil
}

func resourceNotificationChannelAttachmentImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	// Import format: channel_id:entity_id:severity
	channelID, entityID, severity, err := parseAttachmentID(d.Id())
	if err != nil {
		return nil, fmt.Errorf("import format should be channel_id:entity_id:severity - %w", err)
	}

	d.Set("channel_id", channelID)
	d.Set("entity_id", entityID)
	d.Set("severity", severity)

	return []*schema.ResourceData{d}, nil
}
