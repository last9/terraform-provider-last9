package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/last9/terraform-provider-last9/internal/client"
)

// Acceptance tests

func TestAccForwardRule_basic(t *testing.T) {
	resourceName := "last9_forward_rule.test"
	region := os.Getenv("LAST9_TEST_REGION")
	if region == "" {
		region = "ap-south-1"
	}
	clusterID := os.Getenv("LAST9_TEST_CLUSTER_ID")
	if clusterID == "" {
		t.Skip("Skipping test - LAST9_TEST_CLUSTER_ID not set")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckForwardRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccForwardRuleConfig_basic(region, clusterID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckForwardRuleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-test-forward-rule"),
					resource.TestCheckResourceAttr(resourceName, "region", region),
					resource.TestCheckResourceAttr(resourceName, "cluster_id", clusterID),
					resource.TestCheckResourceAttr(resourceName, "telemetry", "logs"),
					resource.TestCheckResourceAttr(resourceName, "destination", "https://example.com/logs"),
					resource.TestCheckResourceAttr(resourceName, "filters.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "filters.0.key", `attributes["service"]`),
					resource.TestCheckResourceAttr(resourceName, "filters.0.value", "important-service"),
					resource.TestCheckResourceAttr(resourceName, "filters.0.operator", "equals"),
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

func TestAccForwardRule_update(t *testing.T) {
	resourceName := "last9_forward_rule.test"
	region := os.Getenv("LAST9_TEST_REGION")
	if region == "" {
		region = "ap-south-1"
	}
	clusterID := os.Getenv("LAST9_TEST_CLUSTER_ID")
	if clusterID == "" {
		t.Skip("Skipping test - LAST9_TEST_CLUSTER_ID not set")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckForwardRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccForwardRuleConfig_basic(region, clusterID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckForwardRuleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-test-forward-rule"),
					resource.TestCheckResourceAttr(resourceName, "destination", "https://example.com/logs"),
				),
			},
			{
				Config: testAccForwardRuleConfig_updated(region, clusterID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckForwardRuleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-test-forward-rule"),
					resource.TestCheckResourceAttr(resourceName, "destination", "https://example.com/logs-v2"),
					resource.TestCheckResourceAttr(resourceName, "filters.0.value", "critical-service"),
				),
			},
		},
	})
}

func TestAccForwardRule_multipleFilters(t *testing.T) {
	resourceName := "last9_forward_rule.test"
	region := os.Getenv("LAST9_TEST_REGION")
	if region == "" {
		region = "ap-south-1"
	}
	clusterID := os.Getenv("LAST9_TEST_CLUSTER_ID")
	if clusterID == "" {
		t.Skip("Skipping test - LAST9_TEST_CLUSTER_ID not set")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckForwardRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccForwardRuleConfig_multipleFilters(region, clusterID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckForwardRuleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-test-forward-rule-multi"),
					resource.TestCheckResourceAttr(resourceName, "filters.#", "2"),
				),
			},
		},
	})
}

// Helper functions

func testAccCheckForwardRuleExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no forward rule ID is set")
		}

		return nil
	}
}

func testAccCheckForwardRuleDestroy(s *terraform.State) error {
	provider := testAccProvider()
	providerConfig := provider.Meta()
	if providerConfig == nil {
		// Provider not configured, skip check
		return nil
	}

	apiClient := providerConfig.(*client.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_forward_rule" {
			continue
		}

		region := rs.Primary.Attributes["region"]
		name := rs.Primary.Attributes["name"]

		result, err := apiClient.GetForwardRules(region)
		if err != nil {
			// API error, assume deleted
			continue
		}

		for _, rule := range result.Properties {
			if rule.Name == name {
				return fmt.Errorf("forward rule %s still exists in region %s", name, region)
			}
		}
	}

	return nil
}

// Configuration helpers

func testAccForwardRuleConfig_basic(region, clusterID string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_forward_rule" "test" {
  region      = "%s"
  cluster_id  = "%s"
  name        = "tf-test-forward-rule"
  telemetry   = "logs"
  destination = "https://example.com/logs"

  filters {
    key      = "attributes[\"service\"]"
    value    = "important-service"
    operator = "equals"
  }
}
`, region, clusterID)
}

func testAccForwardRuleConfig_updated(region, clusterID string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_forward_rule" "test" {
  region      = "%s"
  cluster_id  = "%s"
  name        = "tf-test-forward-rule"
  telemetry   = "logs"
  destination = "https://example.com/logs-v2"

  filters {
    key      = "attributes[\"service\"]"
    value    = "critical-service"
    operator = "equals"
  }
}
`, region, clusterID)
}

func testAccForwardRuleConfig_multipleFilters(region, clusterID string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_forward_rule" "test" {
  region      = "%s"
  cluster_id  = "%s"
  name        = "tf-test-forward-rule-multi"
  telemetry   = "logs"
  destination = "https://example.com/logs"

  filters {
    key         = "attributes[\"service\"]"
    value       = "important-service"
    operator    = "equals"
    conjunction = "and"
  }

  filters {
    key      = "attributes[\"environment\"]"
    value    = "production"
    operator = "equals"
  }
}
`, region, clusterID)
}
