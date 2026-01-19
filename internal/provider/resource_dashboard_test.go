package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDashboard_basic(t *testing.T) {
	var dashboardID string
	resourceName := "last9_dashboard.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckDashboardDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDashboardExists(resourceName, &dashboardID),
					resource.TestCheckResourceAttr(resourceName, "name", "Test Dashboard"),
					resource.TestCheckResourceAttr(resourceName, "readonly", "false"),
					resource.TestCheckResourceAttr(resourceName, "panels.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDashboard_update(t *testing.T) {
	var dashboardID string
	resourceName := "last9_dashboard.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckDashboardDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDashboardExists(resourceName, &dashboardID),
					resource.TestCheckResourceAttr(resourceName, "name", "Test Dashboard"),
				),
			},
			{
				Config: testAccDashboardConfig_updated(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDashboardExists(resourceName, &dashboardID),
					resource.TestCheckResourceAttr(resourceName, "name", "Updated Dashboard"),
					resource.TestCheckResourceAttr(resourceName, "description", "Updated description"),
				),
			},
		},
	})
}

func testAccCheckDashboardExists(n string, id *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no dashboard ID is set")
		}

		*id = rs.Primary.ID
		return nil
	}
}

func testAccCheckDashboardDestroy(s *terraform.State) error {
	// Implement actual destroy check if needed
	return nil
}

func testAccDashboardConfig_basic() string {
	return testAccProviderConfig() + `
resource "last9_dashboard" "test" {
  name        = "Test Dashboard"
  description = "Test dashboard description"
  readonly    = false

  panels {
    title         = "Test Panel"
    query         = "sum(rate(http_requests_total[5m]))"
    visualization = "line"
  }

  tags = ["test"]
}
`
}

func testAccDashboardConfig_updated() string {
	return testAccProviderConfig() + `
resource "last9_dashboard" "test" {
  name        = "Updated Dashboard"
  description = "Updated description"
  readonly    = false

  panels {
    title         = "Test Panel"
    query         = "sum(rate(http_requests_total[5m]))"
    visualization = "line"
  }

  tags = ["test", "updated"]
}
`
}
