package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func New() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("LAST9_API_TOKEN", nil),
				Description: "Last9 API access token (legacy - use refresh_token instead)",
				Sensitive:   true,
			},
			"refresh_token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("LAST9_REFRESH_TOKEN", nil),
				Description: "Last9 refresh token for automatic access token management",
				Sensitive:   true,
			},
			"delete_token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("LAST9_DELETE_TOKEN", nil),
				Description: "Last9 API token with delete scope (legacy - use delete_refresh_token instead)",
				Sensitive:   true,
			},
			"delete_refresh_token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("LAST9_DELETE_REFRESH_TOKEN", nil),
				Description: "Last9 refresh token for delete operations (generates access tokens with delete scope)",
				Sensitive:   true,
			},
			"org": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("LAST9_ORG", nil),
				Description: "Last9 organization slug",
			},
			"api_base_url": {
				Type:         schema.TypeString,
				Required:     true,
				DefaultFunc:  schema.EnvDefaultFunc("LAST9_API_BASE_URL", nil),
				Description:  "Last9 API base URL (required - set via LAST9_API_BASE_URL env var or provider config)",
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"last9_entity":                 resourceEntity(),
			"last9_alert":                  resourceAlert(),
			"last9_drop_rule":              resourceDropRule(),
			"last9_forward_rule":           resourceForwardRule(),
			"last9_scheduled_search_alert": resourceScheduledSearchAlert(),
			"last9_notification_channel":   resourceNotificationChannel(),
			// Note: notification_channel_attachment is not registered because the API
			// doesn't support reading child channels after creation. Attachments should
			// be managed via the entity's notification_channels field instead.
		},
		DataSourcesMap: map[string]*schema.Resource{
			"last9_entity":                   dataSourceEntity(),
			"last9_notification_destination": dataSourceNotificationDestination(),
		},
		ConfigureContextFunc: configureProvider,
	}
}

func configureProvider(ctx context.Context, d *schema.ResourceData) (any, diag.Diagnostics) {
	apiToken := d.Get("api_token").(string)
	refreshToken := d.Get("refresh_token").(string)
	deleteToken := d.Get("delete_token").(string)
	deleteRefreshToken := d.Get("delete_refresh_token").(string)
	org := d.Get("org").(string)
	apiBaseURL := d.Get("api_base_url").(string)

	// Check environment variables if not set in config
	if apiToken == "" {
		apiToken = os.Getenv("LAST9_API_TOKEN")
	}
	if refreshToken == "" {
		refreshToken = os.Getenv("LAST9_REFRESH_TOKEN")
	}
	if deleteToken == "" {
		deleteToken = os.Getenv("LAST9_DELETE_TOKEN")
	}
	if deleteRefreshToken == "" {
		deleteRefreshToken = os.Getenv("LAST9_DELETE_REFRESH_TOKEN")
	}
	if org == "" {
		org = os.Getenv("LAST9_ORG")
	}
	if apiBaseURL == "" {
		apiBaseURL = os.Getenv("LAST9_API_BASE_URL")
	}

	if org == "" {
		return nil, diag.FromErr(fmt.Errorf("org is required"))
	}
	if apiBaseURL == "" {
		return nil, diag.FromErr(fmt.Errorf("api_base_url is required - set via LAST9_API_BASE_URL environment variable or provider config"))
	}

	// Require either api_token or refresh_token
	if apiToken == "" && refreshToken == "" {
		return nil, diag.FromErr(fmt.Errorf("either api_token or refresh_token must be provided"))
	}

	config := &client.Config{
		APIToken:           apiToken,
		RefreshToken:       refreshToken,
		DeleteToken:        deleteToken,
		DeleteRefreshToken: deleteRefreshToken,
		Org:                org,
		BaseURL:            apiBaseURL,
	}

	apiClient, err := client.NewClient(config)
	if err != nil {
		return nil, diag.FromErr(fmt.Errorf("failed to create Last9 client: %w", err))
	}

	return apiClient, nil
}
