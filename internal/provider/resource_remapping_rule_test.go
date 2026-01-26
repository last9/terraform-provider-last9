package provider

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func TestAccRemappingRule_logsExtractPattern(t *testing.T) {
	resourceName := "last9_remapping_rule.test"
	region := getTestRegion()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckRemappingRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRemappingRuleConfig_logsExtractPattern(region),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRemappingRuleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "region", region),
					resource.TestCheckResourceAttr(resourceName, "type", "logs_extract"),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-test-extract-pattern"),
					resource.TestCheckResourceAttr(resourceName, "target_attributes", "log_attributes"),
					resource.TestCheckResourceAttr(resourceName, "extract_type", "pattern"),
					resource.TestCheckResourceAttr(resourceName, "action", "upsert"),
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

func TestAccRemappingRule_logsExtractJSON(t *testing.T) {
	resourceName := "last9_remapping_rule.test"
	region := getTestRegion()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckRemappingRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRemappingRuleConfig_logsExtractJSON(region),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRemappingRuleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "type", "logs_extract"),
					resource.TestCheckResourceAttr(resourceName, "extract_type", "json"),
					resource.TestCheckResourceAttr(resourceName, "prefix", "parsed_"),
				),
			},
		},
	})
}

func TestAccRemappingRule_logsMap(t *testing.T) {
	resourceName := "last9_remapping_rule.test"
	region := getTestRegion()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckRemappingRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRemappingRuleConfig_logsMap(region),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRemappingRuleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "type", "logs_map"),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-test-logs-map"),
					resource.TestCheckResourceAttr(resourceName, "target_attributes", "service"),
				),
			},
		},
	})
}

func TestAccRemappingRule_tracesMap(t *testing.T) {
	resourceName := "last9_remapping_rule.test"
	region := getTestRegion()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckRemappingRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRemappingRuleConfig_tracesMap(region),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRemappingRuleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "type", "traces_map"),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-test-traces-map"),
					resource.TestCheckResourceAttr(resourceName, "target_attributes", "service"),
				),
			},
		},
	})
}

func TestAccRemappingRule_update(t *testing.T) {
	resourceName := "last9_remapping_rule.test"
	region := getTestRegion()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckRemappingRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRemappingRuleConfig_logsMap(region),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRemappingRuleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "action", "upsert"),
				),
			},
			{
				Config: testAccRemappingRuleConfig_logsMapUpdated(region),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRemappingRuleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "action", "insert"),
				),
			},
		},
	})
}

func TestAccRemappingRule_validationExtractTypeRequired(t *testing.T) {
	region := getTestRegion()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccRemappingRuleConfig_invalidMissingExtractType(region),
				ExpectError: regexp.MustCompile(`extract_type is required for logs_extract type`),
			},
		},
	})
}

func TestAccRemappingRule_validationExtractTypeOnlyForExtract(t *testing.T) {
	region := getTestRegion()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccRemappingRuleConfig_invalidExtractTypeOnMap(region),
				ExpectError: regexp.MustCompile(`extract_type is only valid for logs_extract type`),
			},
		},
	})
}

func TestAccRemappingRule_validationInvalidTargetForLogsMap(t *testing.T) {
	region := getTestRegion()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccRemappingRuleConfig_invalidTargetForLogsMap(region),
				ExpectError: regexp.MustCompile(`target_attributes must be 'service', 'severity', or 'resource_deployment.environment' for logs_map type`),
			},
		},
	})
}

// Helper functions

func getTestRegion() string {
	region := os.Getenv("LAST9_TEST_REGION")
	if region == "" {
		region = "ap-south-1"
	}
	return region
}

func testAccCheckRemappingRuleExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no remapping rule ID is set")
		}

		return nil
	}
}

func testAccCheckRemappingRuleDestroy(s *terraform.State) error {
	provider := testAccProvider()
	providerConfig := provider.Meta()
	if providerConfig == nil {
		return nil
	}

	apiClient := providerConfig.(*client.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_remapping_rule" {
			continue
		}

		parts := strings.SplitN(rs.Primary.ID, ":", 3)
		if len(parts) != 3 {
			continue
		}

		region := parts[0]
		ruleType := parts[1]
		ruleName := parts[2]

		result, err := apiClient.GetRemappingRuleByType(region, ruleType)
		if err != nil {
			continue
		}

		if result != nil {
			for _, prop := range result.Properties {
				if prop.Name != nil && *prop.Name == ruleName {
					return fmt.Errorf("remapping rule %s still exists", ruleName)
				}
			}
		}
	}

	return nil
}

// Configuration helpers

func testAccRemappingRuleConfig_logsExtractPattern(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_remapping_rule" "test" {
  region            = "%s"
  type              = "logs_extract"
  name              = "tf-test-extract-pattern"
  remap_keys        = ["(?P<request_id>[a-f0-9-]+)"]
  target_attributes = "log_attributes"
  action            = "upsert"
  extract_type      = "pattern"
}
`, region)
}

func testAccRemappingRuleConfig_logsExtractJSON(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_remapping_rule" "test" {
  region            = "%s"
  type              = "logs_extract"
  name              = "tf-test-extract-json"
  remap_keys        = ["body"]
  target_attributes = "log_attributes"
  action            = "insert"
  extract_type      = "json"
  prefix            = "parsed_"
}
`, region)
}

func testAccRemappingRuleConfig_logsMap(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_remapping_rule" "test" {
  region            = "%s"
  type              = "logs_map"
  name              = "tf-test-logs-map"
  remap_keys        = ["attributes[\"app.name\"]"]
  target_attributes = "service"
  action            = "upsert"
}
`, region)
}

func testAccRemappingRuleConfig_logsMapUpdated(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_remapping_rule" "test" {
  region            = "%s"
  type              = "logs_map"
  name              = "tf-test-logs-map"
  remap_keys        = ["attributes[\"app.name\"]"]
  target_attributes = "service"
  action            = "insert"
}
`, region)
}

func testAccRemappingRuleConfig_tracesMap(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_remapping_rule" "test" {
  region            = "%s"
  type              = "traces_map"
  name              = "tf-test-traces-map"
  remap_keys        = ["resource.attributes[\"k8s.deployment.name\"]"]
  target_attributes = "service"
  action            = "upsert"
}
`, region)
}

func testAccRemappingRuleConfig_invalidMissingExtractType(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_remapping_rule" "test" {
  region            = "%s"
  type              = "logs_extract"
  name              = "tf-test-invalid"
  remap_keys        = ["body"]
  target_attributes = "log_attributes"
  action            = "upsert"
}
`, region)
}

func testAccRemappingRuleConfig_invalidExtractTypeOnMap(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_remapping_rule" "test" {
  region            = "%s"
  type              = "logs_map"
  name              = "tf-test-invalid"
  remap_keys        = ["attributes[\"app.name\"]"]
  target_attributes = "service"
  action            = "upsert"
  extract_type      = "pattern"
}
`, region)
}

func testAccRemappingRuleConfig_invalidTargetForLogsMap(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_remapping_rule" "test" {
  region            = "%s"
  type              = "logs_map"
  name              = "tf-test-invalid"
  remap_keys        = ["attributes[\"app.name\"]"]
  target_attributes = "log_attributes"
  action            = "upsert"
}
`, region)
}
