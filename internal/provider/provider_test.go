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
	if v := os.Getenv("LAST9_REFRESH_TOKEN"); v == "" {
		if v := os.Getenv("LAST9_API_TOKEN"); v == "" {
			t.Skip("Skipping acceptance test - LAST9_REFRESH_TOKEN or LAST9_API_TOKEN must be set")
		}
	}

	if v := os.Getenv("LAST9_ORG"); v == "" {
		t.Skip("Skipping acceptance test - LAST9_ORG must be set")
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
