package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/last9/terraform-provider-last9/internal/client"
)

// These acceptance tests run only with TF_ACC=1 and real LAST9_* credentials.
// The framework runs a plan after each apply step and fails on a non-empty
// plan, which guards against query/resultant_query round-trip diffs.

func testAccTelemetryRegion() string {
	if r := os.Getenv("LAST9_TEST_REGION"); r != "" {
		return r
	}
	return "ap-south-1"
}

func TestAccLogsToMetrics_basic(t *testing.T) {
	region := testAccTelemetryRegion()
	resourceName := "last9_logs_to_metrics.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckTelemetryToMetricsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccLogsToMetricsConfig(region),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-l2m"),
					resource.TestCheckResourceAttr(resourceName, "query_type", "logjson-aggregate"),
					resource.TestCheckResourceAttr(resourceName, "metric_name", "tf_acc_l2m"),
					resource.TestCheckResourceAttrSet(resourceName, "resultant_query"),
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

func TestAccTracesToMetrics_basic(t *testing.T) {
	region := testAccTelemetryRegion()
	resourceName := "last9_traces_to_metrics.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckTelemetryToMetricsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTracesToMetricsConfig(region),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-t2m"),
					resource.TestCheckResourceAttr(resourceName, "query_type", "tracejson-aggregate"),
					resource.TestCheckResourceAttr(resourceName, "metric_name", "tf_acc_t2m"),
					resource.TestCheckResourceAttrSet(resourceName, "resultant_query"),
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

func testAccCheckTelemetryToMetricsDestroy(s *terraform.State) error {
	provider := testAccProvider()
	providerConfig := provider.Meta()
	if providerConfig == nil {
		return nil
	}
	apiClient := providerConfig.(*client.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_logs_to_metrics" && rs.Type != "last9_traces_to_metrics" {
			continue
		}

		region, ruleID, _, err := parseTelemetryToMetricsID(rs.Primary.ID)
		if err != nil {
			continue
		}

		rules, err := apiClient.GetScheduledSearchRules(region, ruleTypeStreamingAggregation)
		if err != nil {
			continue
		}
		for _, r := range rules {
			if r.ID == ruleID {
				return fmt.Errorf("telemetry-to-metrics rule %s still exists in region %s", ruleID, region)
			}
		}
	}

	return nil
}

func testAccLogsToMetricsConfig(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_logs_to_metrics" "test" {
  region           = %[1]q
  name             = "tf-acc-l2m"
  query_type       = "logjson-aggregate"
  metric_name      = "tf_acc_l2m"
  search_frequency = 60

  query = jsonencode([
    { type = "filter", query = { "$and" = [{ "$neq" = ["ServiceName", ""] }] } },
    {
      type       = "aggregate"
      aggregates = [{ function = { "$count" = [] }, as = "count" }]
      groupby    = { "ServiceName" = "service_name" }
    }
  ])
}
`, region)
}

func testAccTracesToMetricsConfig(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_traces_to_metrics" "test" {
  region           = %[1]q
  name             = "tf-acc-t2m"
  query_type       = "tracejson-aggregate"
  metric_name      = "tf_acc_t2m"
  search_frequency = 60

  query = jsonencode([
    { type = "filter", query = { "$and" = [{ "$neq" = ["ServiceName", ""] }] } },
    {
      type       = "aggregate"
      aggregates = [{ function = { "$count" = [] }, as = "count" }]
      groupby    = { "ServiceName" = "service_name" }
    }
  ])
}
`, region)
}
