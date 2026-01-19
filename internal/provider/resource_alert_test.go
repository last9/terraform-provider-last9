package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAlert_basic(t *testing.T) {
	var alertID string
	resourceName := "last9_alert.test"
	entityID := os.Getenv("LAST9_TEST_ENTITY_ID")
	if entityID == "" {
		t.Skip("Skipping test - LAST9_TEST_ENTITY_ID not set")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckAlertDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAlertConfig_basic(entityID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlertExists(resourceName, &alertID),
					resource.TestCheckResourceAttr(resourceName, "name", "Test Alert"),
					resource.TestCheckResourceAttr(resourceName, "severity", "breach"),
					resource.TestCheckResourceAttr(resourceName, "mute", "false"),
					resource.TestCheckResourceAttr(resourceName, "is_disabled", "false"),
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

func TestAccAlert_staticThreshold(t *testing.T) {
	var alertID string
	resourceName := "last9_alert.test"
	entityID := os.Getenv("LAST9_TEST_ENTITY_ID")
	if entityID == "" {
		t.Skip("Skipping test - LAST9_TEST_ENTITY_ID not set")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckAlertDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAlertConfig_staticThreshold(entityID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlertExists(resourceName, &alertID),
					resource.TestCheckResourceAttr(resourceName, "name", "Static Threshold Alert"),
					resource.TestCheckResourceAttr(resourceName, "greater_than", "100"),
					resource.TestCheckResourceAttr(resourceName, "bad_minutes", "5"),
					resource.TestCheckResourceAttr(resourceName, "total_minutes", "10"),
				),
			},
		},
	})
}

func TestAccAlert_expression(t *testing.T) {
	var alertID string
	resourceName := "last9_alert.test"
	entityID := os.Getenv("LAST9_TEST_ENTITY_ID")
	if entityID == "" {
		t.Skip("Skipping test - LAST9_TEST_ENTITY_ID not set")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckAlertDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAlertConfig_expression(entityID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlertExists(resourceName, &alertID),
					resource.TestCheckResourceAttr(resourceName, "name", "Expression Alert"),
					resource.TestCheckResourceAttr(resourceName, "expression", "low_spike(0.5, throughput)"),
				),
			},
		},
	})
}

func testAccCheckAlertExists(n string, id *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no alert ID is set")
		}

		*id = rs.Primary.ID
		return nil
	}
}

func testAccCheckAlertDestroy(s *terraform.State) error {
	// Implement actual destroy check if needed
	return nil
}

func testAccAlertConfig_basic(entityID string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_alert" "test" {
  entity_id   = "%s"
  name        = "Test Alert"
  description = "Test alert description"
  indicator   = "throughput"
  
  greater_than  = 100
  bad_minutes   = 5
  total_minutes = 10
  
  severity = "breach"
}
`, entityID)
}

func testAccAlertConfig_staticThreshold(entityID string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_alert" "test" {
  entity_id   = "%s"
  name        = "Static Threshold Alert"
  description = "Alert with static threshold"
  indicator   = "error_rate"
  
  greater_than  = 100
  bad_minutes   = 5
  total_minutes = 10
  
  severity = "breach"
  
  properties {
    runbook_url = "https://example.com/runbook"
    annotations = {
      priority = "high"
      team     = "platform"
    }
  }
}
`, entityID)
}

func testAccAlertConfig_expression(entityID string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_alert" "test" {
  entity_id   = "%s"
  name        = "Expression Alert"
  description = "Alert with expression"
  indicator   = "throughput"
  
  expression = "low_spike(0.5, throughput)"
  severity   = "threat"
  
  properties {
    annotations = {
      priority = "medium"
    }
  }
}
`, entityID)
}
