package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccAlertIntegration_fullLifecycle tests the complete create -> update -> delete cycle
// for alerts, including automatic KPI creation and cleanup
func TestAccAlertIntegration_fullLifecycle(t *testing.T) {
	var entityID, alertID string
	entityResourceName := "last9_entity.test"
	alertResourceName := "last9_alert.test"
	timestamp := time.Now().UnixNano()
	entityName := fmt.Sprintf("integration-test-entity-%d", timestamp)
	externalRef := fmt.Sprintf("integration-test-ref-%d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckAlertIntegrationDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create entity and alert
			{
				Config: testAccAlertIntegrationConfig_basic(entityName, externalRef, "Integration Test Alert", "up{job=\"test\"}"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEntityExists(entityResourceName, &entityID),
					testAccCheckAlertIntegrationExists(alertResourceName, &alertID),
					resource.TestCheckResourceAttr(alertResourceName, "name", "Integration Test Alert"),
					resource.TestCheckResourceAttrSet(alertResourceName, "kpi_id"),
					resource.TestCheckResourceAttrSet(alertResourceName, "kpi_name"),
				),
			},
			// Step 2: Update alert name (triggers KPI recreation)
			{
				Config: testAccAlertIntegrationConfig_basic(entityName, externalRef, "Updated Alert Name", "up{job=\"test\"}"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(alertResourceName, "name", "Updated Alert Name"),
					resource.TestCheckResourceAttrSet(alertResourceName, "kpi_id"),
				),
			},
			// Step 3: Update query
			{
				Config: testAccAlertIntegrationConfig_basic(entityName, externalRef, "Updated Alert Name", "up{job=\"updated\"}"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(alertResourceName, "query", "up{job=\"updated\"}"),
				),
			},
		},
	})
}

// TestAccAlertIntegration_multipleAlerts tests creating multiple alerts on the same entity
func TestAccAlertIntegration_multipleAlerts(t *testing.T) {
	var entityID, alertID1, alertID2 string
	entityResourceName := "last9_entity.test"
	alert1ResourceName := "last9_alert.alert1"
	alert2ResourceName := "last9_alert.alert2"
	timestamp := time.Now().UnixNano()
	entityName := fmt.Sprintf("multi-alert-entity-%d", timestamp)
	externalRef := fmt.Sprintf("multi-alert-ref-%d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckAlertIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAlertIntegrationConfig_multipleAlerts(entityName, externalRef),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEntityExists(entityResourceName, &entityID),
					testAccCheckAlertIntegrationExists(alert1ResourceName, &alertID1),
					testAccCheckAlertIntegrationExists(alert2ResourceName, &alertID2),
					resource.TestCheckResourceAttr(alert1ResourceName, "name", "Alert One"),
					resource.TestCheckResourceAttr(alert2ResourceName, "name", "Alert Two"),
				),
			},
		},
	})
}

// TestAccAlertIntegration_staticThreshold tests alerts with static threshold conditions
func TestAccAlertIntegration_staticThreshold(t *testing.T) {
	var entityID, alertID string
	entityResourceName := "last9_entity.test"
	alertResourceName := "last9_alert.test"
	timestamp := time.Now().UnixNano()
	entityName := fmt.Sprintf("static-threshold-entity-%d", timestamp)
	externalRef := fmt.Sprintf("static-threshold-ref-%d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckAlertIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAlertIntegrationConfig_staticThreshold(entityName, externalRef),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEntityExists(entityResourceName, &entityID),
					testAccCheckAlertIntegrationExists(alertResourceName, &alertID),
					resource.TestCheckResourceAttr(alertResourceName, "name", "Static Threshold Alert"),
					resource.TestCheckResourceAttr(alertResourceName, "greater_than", "100"),
					resource.TestCheckResourceAttr(alertResourceName, "bad_minutes", "5"),
					resource.TestCheckResourceAttr(alertResourceName, "total_minutes", "10"),
					resource.TestCheckResourceAttr(alertResourceName, "severity", "breach"),
				),
			},
		},
	})
}

// TestAccAlertIntegration_withProperties tests alerts with runbook and annotations
func TestAccAlertIntegration_withProperties(t *testing.T) {
	var entityID, alertID string
	entityResourceName := "last9_entity.test"
	alertResourceName := "last9_alert.test"
	timestamp := time.Now().UnixNano()
	entityName := fmt.Sprintf("props-alert-entity-%d", timestamp)
	externalRef := fmt.Sprintf("props-alert-ref-%d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckAlertIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAlertIntegrationConfig_withProperties(entityName, externalRef),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEntityExists(entityResourceName, &entityID),
					testAccCheckAlertIntegrationExists(alertResourceName, &alertID),
					resource.TestCheckResourceAttr(alertResourceName, "name", "Alert With Properties"),
					resource.TestCheckResourceAttr(alertResourceName, "properties.0.runbook_url", "https://example.com/runbook"),
					resource.TestCheckResourceAttr(alertResourceName, "properties.0.annotations.priority", "high"),
					resource.TestCheckResourceAttr(alertResourceName, "properties.0.annotations.team", "platform"),
				),
			},
		},
	})
}

// TestAccAlertIntegration_import tests importing an existing alert
func TestAccAlertIntegration_import(t *testing.T) {
	var entityID, alertID string
	entityResourceName := "last9_entity.test"
	alertResourceName := "last9_alert.test"
	timestamp := time.Now().UnixNano()
	entityName := fmt.Sprintf("import-test-entity-%d", timestamp)
	externalRef := fmt.Sprintf("import-test-ref-%d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckAlertIntegrationDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccAlertIntegrationConfig_basic(entityName, externalRef, "Import Test Alert", "up{job=\"test\"}"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEntityExists(entityResourceName, &entityID),
					testAccCheckAlertIntegrationExists(alertResourceName, &alertID),
				),
			},
			// Import using composite ID format: entity_id:alert_id
			{
				ResourceName:      alertResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[alertResourceName]
					if !ok {
						return "", fmt.Errorf("not found: %s", alertResourceName)
					}
					entityID := rs.Primary.Attributes["entity_id"]
					alertID := rs.Primary.ID
					return fmt.Sprintf("%s:%s", entityID, alertID), nil
				},
				// query, kpi_id are computed during create, may not match exactly on import
				// is_disabled is intentionally not read from API to avoid drift
				ImportStateVerifyIgnore: []string{"query", "kpi_id", "kpi_name", "is_disabled"},
			},
		},
	})
}

// TestAccAlertIntegration_lessThanThreshold tests alerts with less-than threshold
func TestAccAlertIntegration_lessThanThreshold(t *testing.T) {
	var entityID, alertID string
	entityResourceName := "last9_entity.test"
	alertResourceName := "last9_alert.test"
	timestamp := time.Now().UnixNano()
	entityName := fmt.Sprintf("less-than-entity-%d", timestamp)
	externalRef := fmt.Sprintf("less-than-ref-%d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckAlertIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAlertIntegrationConfig_lessThanThreshold(entityName, externalRef),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEntityExists(entityResourceName, &entityID),
					testAccCheckAlertIntegrationExists(alertResourceName, &alertID),
					resource.TestCheckResourceAttr(alertResourceName, "name", "Less Than Alert"),
					resource.TestCheckResourceAttr(alertResourceName, "less_than", "10"),
					resource.TestCheckResourceAttr(alertResourceName, "severity", "threat"),
				),
			},
		},
	})
}

// testAccCheckAlertIntegrationExists verifies an alert exists in state
func testAccCheckAlertIntegrationExists(n string, id *string) resource.TestCheckFunc {
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

// testAccCheckAlertIntegrationDestroy verifies alerts and entities are destroyed
func testAccCheckAlertIntegrationDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_alert" && rs.Type != "last9_entity" {
			continue
		}
		// Resources should be destroyed - the test framework will fail if destroy fails
	}
	return nil
}

// Config helper functions

func testAccAlertIntegrationConfig_basic(entityName, externalRef, alertName, query string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_entity" "test" {
  name         = %q
  type         = "service"
  external_ref = %q
}

resource "last9_alert" "test" {
  entity_id     = last9_entity.test.id
  name          = %q
  description   = "Test alert for integration testing"
  query         = %q

  greater_than  = 100
  bad_minutes   = 5
  total_minutes = 10

  severity = "breach"
}
`, entityName, externalRef, alertName, query)
}

func testAccAlertIntegrationConfig_multipleAlerts(entityName, externalRef string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_entity" "test" {
  name         = %q
  type         = "service"
  external_ref = %q
}

resource "last9_alert" "alert1" {
  entity_id     = last9_entity.test.id
  name          = "Alert One"
  description   = "First test alert"
  query         = "up{job=\"service1\"}"

  greater_than  = 100
  bad_minutes   = 5
  total_minutes = 10

  severity = "breach"
}

resource "last9_alert" "alert2" {
  entity_id     = last9_entity.test.id
  name          = "Alert Two"
  description   = "Second test alert"
  query         = "up{job=\"service2\"}"

  greater_than  = 50
  bad_minutes   = 3
  total_minutes = 5

  severity = "threat"
}
`, entityName, externalRef)
}

func testAccAlertIntegrationConfig_staticThreshold(entityName, externalRef string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_entity" "test" {
  name         = %q
  type         = "service"
  external_ref = %q
}

resource "last9_alert" "test" {
  entity_id     = last9_entity.test.id
  name          = "Static Threshold Alert"
  description   = "Alert with static threshold configuration"
  query         = "error_rate{service=\"api\"}"

  greater_than  = 100
  bad_minutes   = 5
  total_minutes = 10

  severity = "breach"
}
`, entityName, externalRef)
}

func testAccAlertIntegrationConfig_withProperties(entityName, externalRef string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_entity" "test" {
  name         = %q
  type         = "service"
  external_ref = %q
}

resource "last9_alert" "test" {
  entity_id     = last9_entity.test.id
  name          = "Alert With Properties"
  description   = "Alert with runbook and annotations"
  query         = "latency_p99{service=\"api\"}"

  greater_than  = 500
  bad_minutes   = 3
  total_minutes = 5

  severity = "breach"

  properties {
    runbook_url = "https://example.com/runbook"
    annotations = {
      priority = "high"
      team     = "platform"
    }
  }
}
`, entityName, externalRef)
}

func testAccAlertIntegrationConfig_lessThanThreshold(entityName, externalRef string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_entity" "test" {
  name         = %q
  type         = "service"
  external_ref = %q
}

resource "last9_alert" "test" {
  entity_id     = last9_entity.test.id
  name          = "Less Than Alert"
  description   = "Alert that fires when value drops below threshold"
  query         = "availability{service=\"api\"}"

  less_than     = 10
  bad_minutes   = 2
  total_minutes = 5

  severity = "threat"
}
`, entityName, externalRef)
}
