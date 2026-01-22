package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/last9/terraform-provider-last9/internal/client"
)

// Unit tests for helper functions

func TestExpandPostProcessors(t *testing.T) {
	tests := []struct {
		name    string
		input   []interface{}
		wantErr bool
	}{
		{
			name: "valid single aggregate",
			input: []interface{}{
				map[string]interface{}{
					"type": "aggregate",
					"aggregates": []interface{}{
						map[string]interface{}{
							"function": `{"$count":[]}`,
							"as":       "result",
						},
					},
					"groupby": "{}",
				},
			},
			wantErr: false,
		},
		{
			name: "valid multiple aggregates",
			input: []interface{}{
				map[string]interface{}{
					"type": "aggregate",
					"aggregates": []interface{}{
						map[string]interface{}{
							"function": `{"$count":[]}`,
							"as":       "count",
						},
						map[string]interface{}{
							"function": `{"$sum":["attributes.bytes"]}`,
							"as":       "total_bytes",
						},
					},
					"groupby": `{"service":["attributes.service"]}`,
				},
			},
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   []interface{}{},
			wantErr: true,
		},
		{
			name: "invalid function JSON",
			input: []interface{}{
				map[string]interface{}{
					"type": "aggregate",
					"aggregates": []interface{}{
						map[string]interface{}{
							"function": `invalid json`,
							"as":       "result",
						},
					},
					"groupby": "{}",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid groupby JSON",
			input: []interface{}{
				map[string]interface{}{
					"type": "aggregate",
					"aggregates": []interface{}{
						map[string]interface{}{
							"function": `{"$count":[]}`,
							"as":       "result",
						},
					},
					"groupby": `invalid json`,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandPostProcessors(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("expandPostProcessors() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Errorf("expandPostProcessors() returned nil result")
			}
		})
	}
}

func TestFlattenPostProcessors(t *testing.T) {
	tests := []struct {
		name  string
		input []client.PostProcessor
		want  int // expected number of items
	}{
		{
			name:  "empty input",
			input: []client.PostProcessor{},
			want:  0,
		},
		{
			name: "single processor",
			input: []client.PostProcessor{
				{
					Type: "aggregate",
					Aggregates: []client.Aggregate{
						{
							Function: map[string]interface{}{"$count": []interface{}{}},
							As:       "result",
						},
					},
					Groupby: map[string]interface{}{},
				},
			},
			want: 1,
		},
		{
			name: "processor with multiple aggregates",
			input: []client.PostProcessor{
				{
					Type: "aggregate",
					Aggregates: []client.Aggregate{
						{
							Function: map[string]interface{}{"$count": []interface{}{}},
							As:       "count",
						},
						{
							Function: map[string]interface{}{"$sum": []interface{}{"attributes.bytes"}},
							As:       "total",
						},
					},
					Groupby: map[string]interface{}{"service": []interface{}{"attributes.service"}},
				},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenPostProcessors(tt.input)
			if len(result) != tt.want {
				t.Errorf("flattenPostProcessors() returned %d items, want %d", len(result), tt.want)
			}

			// Verify structure
			for _, item := range result {
				m, ok := item.(map[string]interface{})
				if !ok {
					t.Errorf("flattenPostProcessors() item is not a map")
					continue
				}

				// Check required fields
				if _, ok := m["type"]; !ok {
					t.Errorf("flattenPostProcessors() missing 'type' field")
				}
				if _, ok := m["aggregates"]; !ok {
					t.Errorf("flattenPostProcessors() missing 'aggregates' field")
				}
				if _, ok := m["groupby"]; !ok {
					t.Errorf("flattenPostProcessors() missing 'groupby' field")
				}

				// Verify groupby is valid JSON string
				groupbyStr, ok := m["groupby"].(string)
				if !ok {
					t.Errorf("flattenPostProcessors() groupby is not a string")
					continue
				}
				var groupby map[string]interface{}
				if err := json.Unmarshal([]byte(groupbyStr), &groupby); err != nil {
					t.Errorf("flattenPostProcessors() groupby is not valid JSON: %v", err)
				}
			}
		})
	}
}

// Acceptance tests

func TestAccScheduledSearchAlert_basic(t *testing.T) {
	var alertID string
	resourceName := "last9_scheduled_search_alert.test"
	region := os.Getenv("LAST9_TEST_REGION")
	if region == "" {
		region = "ap-south-1" // Default region
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckScheduledSearchAlertDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccScheduledSearchAlertConfig_basic(region),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScheduledSearchAlertExists(resourceName, &alertID),
					resource.TestCheckResourceAttr(resourceName, "name", "Test Error Count Alert"),
					resource.TestCheckResourceAttr(resourceName, "region", region),
					resource.TestCheckResourceAttr(resourceName, "query_type", "logjson-aggregate"),
					resource.TestCheckResourceAttr(resourceName, "search_frequency", "300"),
					resource.TestCheckResourceAttr(resourceName, "threshold.0.operator", ">"),
					resource.TestCheckResourceAttr(resourceName, "threshold.0.value", "100"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"alert_destinations"},
			},
		},
	})
}

func TestAccScheduledSearchAlert_update(t *testing.T) {
	var alertID string
	resourceName := "last9_scheduled_search_alert.test"
	region := os.Getenv("LAST9_TEST_REGION")
	if region == "" {
		region = "ap-south-1"
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckScheduledSearchAlertDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccScheduledSearchAlertConfig_basic(region),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScheduledSearchAlertExists(resourceName, &alertID),
					resource.TestCheckResourceAttr(resourceName, "name", "Test Error Count Alert"),
					resource.TestCheckResourceAttr(resourceName, "threshold.0.value", "100"),
				),
			},
			{
				Config: testAccScheduledSearchAlertConfig_updated(region),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScheduledSearchAlertExists(resourceName, &alertID),
					resource.TestCheckResourceAttr(resourceName, "name", "Test Error Count Alert"),
					resource.TestCheckResourceAttr(resourceName, "threshold.0.value", "200"),
					resource.TestCheckResourceAttr(resourceName, "search_frequency", "600"),
				),
			},
		},
	})
}

func TestAccScheduledSearchAlert_withGrouping(t *testing.T) {
	var alertID string
	resourceName := "last9_scheduled_search_alert.test"
	region := os.Getenv("LAST9_TEST_REGION")
	if region == "" {
		region = "ap-south-1"
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckScheduledSearchAlertDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccScheduledSearchAlertConfig_withGrouping(region),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScheduledSearchAlertExists(resourceName, &alertID),
					resource.TestCheckResourceAttr(resourceName, "name", "Test Grouped Alert"),
					resource.TestCheckResourceAttr(resourceName, "post_processor.0.type", "aggregate"),
				),
			},
		},
	})
}

// Helper functions

func testAccCheckScheduledSearchAlertExists(resourceName string, alertID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no scheduled search alert ID is set")
		}

		*alertID = rs.Primary.ID
		return nil
	}
}

func testAccCheckScheduledSearchAlertDestroy(s *terraform.State) error {
	provider := testAccProvider()
	providerConfig := provider.Meta()
	if providerConfig == nil {
		// Provider not configured, skip check
		return nil
	}

	apiClient := providerConfig.(*client.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_scheduled_search_alert" {
			continue
		}

		region := rs.Primary.Attributes["region"]
		name := rs.Primary.Attributes["name"]

		alerts, err := apiClient.GetScheduledSearchAlerts(region)
		if err != nil {
			// API error, assume deleted
			continue
		}

		for _, alert := range alerts {
			if alert.RuleName == name {
				return fmt.Errorf("scheduled search alert %s still exists in region %s", name, region)
			}
		}
	}

	return nil
}

// Configuration helpers

func testAccScheduledSearchAlertConfig_basic(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_notification_channel" "test" {
  name          = "TF Test Alert Channel"
  type          = "slack"
  destination   = "https://hooks.slack.com/services/T00000000/B00000000/scheduled-search-test"
  send_resolved = true
}

resource "last9_scheduled_search_alert" "test" {
  region         = "%s"
  name           = "Test Error Count Alert"
  query_type     = "logjson-aggregate"
  physical_index = "logs"
  telemetry      = "logs"

  query = jsonencode([
    {
      type  = "filter"
      query = {
        "$and" = [
          { "$eq" = ["SeverityText", "ERROR"] }
        ]
      }
    }
  ])

  post_processor {
    type = "aggregate"

    aggregates {
      function = jsonencode({ "$count" = [] })
      as       = "error_count"
    }

    groupby = "{}"
  }

  search_frequency = 300

  threshold {
    operator = ">"
    value    = 100
  }

  alert_destinations = [last9_notification_channel.test.id]
}
`, region)
}

func testAccScheduledSearchAlertConfig_updated(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_notification_channel" "test" {
  name          = "TF Test Alert Channel"
  type          = "slack"
  destination   = "https://hooks.slack.com/services/T00000000/B00000000/scheduled-search-test"
  send_resolved = true
}

resource "last9_scheduled_search_alert" "test" {
  region         = "%s"
  name           = "Test Error Count Alert"
  query_type     = "logjson-aggregate"
  physical_index = "logs"
  telemetry      = "logs"

  query = jsonencode([
    {
      type  = "filter"
      query = {
        "$and" = [
          { "$eq" = ["SeverityText", "ERROR"] }
        ]
      }
    }
  ])

  post_processor {
    type = "aggregate"

    aggregates {
      function = jsonencode({ "$count" = [] })
      as       = "error_count"
    }

    groupby = "{}"
  }

  search_frequency = 600

  threshold {
    operator = ">"
    value    = 200
  }

  alert_destinations = [last9_notification_channel.test.id]
}
`, region)
}

func testAccScheduledSearchAlertConfig_withGrouping(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_notification_channel" "test" {
  name          = "TF Test Grouped Alert Channel"
  type          = "slack"
  destination   = "https://hooks.slack.com/services/T00000000/B00000000/grouped-alert-test"
  send_resolved = true
}

resource "last9_scheduled_search_alert" "test" {
  region         = "%s"
  name           = "Test Grouped Alert"
  query_type     = "logjson-aggregate"
  physical_index = "logs"
  telemetry      = "logs"

  query = jsonencode([
    {
      type  = "filter"
      query = {
        "$and" = [
          { "$eq" = ["SeverityText", "ERROR"] },
          { "$eq" = ["attributes.service", "api-service"] }
        ]
      }
    }
  ])

  post_processor {
    type = "aggregate"

    aggregates {
      function = jsonencode({ "$count" = [] })
      as       = "error_count"
    }

    groupby = jsonencode({
      "endpoint" = ["attributes.endpoint"]
    })
  }

  search_frequency = 300

  threshold {
    operator = ">="
    value    = 50
  }

  alert_destinations = [last9_notification_channel.test.id]
}
`, region)
}
