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
				Description: "Last9 API token with delete scope (required for destroy operations)",
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
				Optional:     true,
				DefaultFunc:  schema.EnvDefaultFunc("LAST9_API_BASE_URL", "https://api.last9.io"),
				Description:  "Last9 API base URL",
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"last9_dashboard":              resourceDashboard(),
			"last9_entity":                 resourceEntity(),
			"last9_alert":                  resourceAlert(),
			"last9_macro":                  resourceMacro(),
			"last9_policy":                 resourcePolicy(),
			"last9_drop_rule":              resourceDropRule(),
			"last9_forward_rule":           resourceForwardRule(),
			"last9_scheduled_search_alert": resourceScheduledSearchAlert(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"last9_dashboard":                dataSourceDashboard(),
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
	if org == "" {
		org = os.Getenv("LAST9_ORG")
	}

	if org == "" {
		return nil, diag.FromErr(fmt.Errorf("org is required"))
	}

	// Require either api_token or refresh_token
	if apiToken == "" && refreshToken == "" {
		return nil, diag.FromErr(fmt.Errorf("either api_token or refresh_token must be provided"))
	}

	config := &client.Config{
		APIToken:     apiToken,
		RefreshToken: refreshToken,
		DeleteToken:  deleteToken,
		Org:          org,
		BaseURL:      apiBaseURL,
	}

	apiClient, err := client.NewClient(config)
	if err != nil {
		return nil, diag.FromErr(fmt.Errorf("failed to create Last9 client: %w", err))
	}

	return apiClient, nil
}
