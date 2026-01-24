package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccProvider_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(),
			},
		},
	})
}

func testAccProviderConfig() string {
	// Check for refresh tokens first (preferred), then fall back to access tokens (legacy)
	writeRefreshToken := os.Getenv("LAST9_WRITE_REFRESH_TOKEN")
	deleteRefreshToken := os.Getenv("LAST9_DELETE_REFRESH_TOKEN")
	// Legacy: direct access tokens
	refreshToken := os.Getenv("LAST9_REFRESH_TOKEN")
	apiToken := os.Getenv("LAST9_API_TOKEN")
	deleteToken := os.Getenv("LAST9_DELETE_TOKEN")
	org := os.Getenv("LAST9_ORG")
	apiBaseURL := os.Getenv("LAST9_API_BASE_URL")

	if org == "" {
		org = "test-org"
	}

	// Build provider config
	config := `provider "last9" {` + "\n"

	// Authentication: prefer write refresh token > refresh_token > api_token
	if writeRefreshToken != "" {
		config += `  refresh_token = "` + writeRefreshToken + `"` + "\n"
	} else if refreshToken != "" {
		config += `  refresh_token = "` + refreshToken + `"` + "\n"
	} else if apiToken != "" {
		config += `  api_token = "` + apiToken + `"` + "\n"
	} else {
		// Fallback for tests that don't set env vars
		config += `  refresh_token = "test-token"` + "\n"
	}

	// Delete token: prefer delete refresh token > delete_token
	if deleteRefreshToken != "" {
		config += `  delete_refresh_token = "` + deleteRefreshToken + `"` + "\n"
	} else if deleteToken != "" {
		config += `  delete_token = "` + deleteToken + `"` + "\n"
	}

	// Add org
	config += `  org = "` + org + `"` + "\n"

	// Add api_base_url if set
	if apiBaseURL != "" {
		config += `  api_base_url = "` + apiBaseURL + `"` + "\n"
	}

	config += `}` + "\n"

	return config
}
