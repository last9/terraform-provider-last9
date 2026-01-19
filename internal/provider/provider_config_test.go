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
	apiToken := os.Getenv("LAST9_API_TOKEN")
	refreshToken := os.Getenv("LAST9_REFRESH_TOKEN")
	org := os.Getenv("LAST9_ORG")

	if org == "" {
		org = "test-org"
	}

	if refreshToken != "" {
		return `
provider "last9" {
  refresh_token = "` + refreshToken + `"
  org          = "` + org + `"
}
`
	} else if apiToken != "" {
		return `
provider "last9" {
  api_token = "` + apiToken + `"
  org      = "` + org + `"
}
`
	}

	return `
provider "last9" {
  refresh_token = "test-token"
  org          = "test-org"
}
`
}
