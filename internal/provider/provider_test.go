package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestProvider(t *testing.T) {
	if err := New().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ *schema.Provider = New()
}

func testAccPreCheck(t *testing.T) {
	// Check for required environment variables
	// Accept either write refresh token or legacy refresh/api token
	writeRefreshToken := os.Getenv("LAST9_WRITE_REFRESH_TOKEN")
	refreshToken := os.Getenv("LAST9_REFRESH_TOKEN")
	apiToken := os.Getenv("LAST9_API_TOKEN")

	if writeRefreshToken == "" && refreshToken == "" && apiToken == "" {
		t.Skip("Skipping acceptance test - LAST9_WRITE_REFRESH_TOKEN, LAST9_REFRESH_TOKEN, or LAST9_API_TOKEN must be set")
	}

	if v := os.Getenv("LAST9_ORG"); v == "" {
		t.Skip("Skipping acceptance test - LAST9_ORG must be set")
	}

	if v := os.Getenv("LAST9_API_BASE_URL"); v == "" {
		t.Skip("Skipping acceptance test - LAST9_API_BASE_URL must be set")
	}
}

// testAccPreCheckWithDelete checks for environment variables including delete token
// Use this for tests that require delete operations
func testAccPreCheckWithDelete(t *testing.T) {
	testAccPreCheck(t)

	// Accept either delete refresh token or legacy delete token
	deleteRefreshToken := os.Getenv("LAST9_DELETE_REFRESH_TOKEN")
	deleteToken := os.Getenv("LAST9_DELETE_TOKEN")

	if deleteRefreshToken == "" && deleteToken == "" {
		t.Skip("Skipping acceptance test - LAST9_DELETE_REFRESH_TOKEN or LAST9_DELETE_TOKEN must be set for destroy tests")
	}
}

// testAccProviderFactories is a helper function to build a map of factory functions
// for instantiating a provider during acceptance tests.
func testAccProviderFactories() map[string]func() (*schema.Provider, error) {
	return map[string]func() (*schema.Provider, error){
		"last9": func() (*schema.Provider, error) {
			return New(), nil
		},
	}
}

// testAccProvider is a helper function that returns a configured provider instance
func testAccProvider() *schema.Provider {
	return New()
}
