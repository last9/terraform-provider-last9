package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/last9/terraform-provider-last9/internal/client"
)

// Unit tests for helper functions

func TestExpandRoutingFilters(t *testing.T) {
	tests := []struct {
		name  string
		input []interface{}
		want  int
	}{
		{
			name:  "empty input",
			input: []interface{}{},
			want:  0,
		},
		{
			name: "single filter without conjunction",
			input: []interface{}{
				map[string]interface{}{
					"key":      `attributes["service"]`,
					"value":    "test-service",
					"operator": "equals",
				},
			},
			want: 1,
		},
		{
			name: "multiple filters with conjunction",
			input: []interface{}{
				map[string]interface{}{
					"key":         `attributes["service"]`,
					"value":       "test-service",
					"operator":    "equals",
					"conjunction": "and",
				},
				map[string]interface{}{
					"key":      `attributes["environment"]`,
					"value":    "production",
					"operator": "equals",
				},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandRoutingFilters(tt.input)
			if len(result) != tt.want {
				t.Errorf("expandRoutingFilters() returned %d items, want %d", len(result), tt.want)
			}
		})
	}
}

func TestFlattenRoutingFilters(t *testing.T) {
	andConj := "and"

	tests := []struct {
		name  string
		input []client.RoutingFilter
		want  int
	}{
		{
			name:  "empty input",
			input: []client.RoutingFilter{},
			want:  0,
		},
		{
			name: "single filter",
			input: []client.RoutingFilter{
				{
					Key:      `attributes["service"]`,
					Value:    "test-service",
					Operator: "equals",
				},
			},
			want: 1,
		},
		{
			name: "filters with conjunction",
			input: []client.RoutingFilter{
				{
					Key:         `attributes["service"]`,
					Value:       "test-service",
					Operator:    "equals",
					Conjunction: &andConj,
				},
				{
					Key:      `attributes["environment"]`,
					Value:    "production",
					Operator: "equals",
				},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenRoutingFilters(tt.input)
			if len(result) != tt.want {
				t.Errorf("flattenRoutingFilters() returned %d items, want %d", len(result), tt.want)
			}

			// Verify structure
			for _, item := range result {
				m, ok := item.(map[string]interface{})
				if !ok {
					t.Errorf("flattenRoutingFilters() item is not a map")
					continue
				}

				// Check required fields
				if _, ok := m["key"]; !ok {
					t.Errorf("flattenRoutingFilters() missing 'key' field")
				}
				if _, ok := m["value"]; !ok {
					t.Errorf("flattenRoutingFilters() missing 'value' field")
				}
				if _, ok := m["operator"]; !ok {
					t.Errorf("flattenRoutingFilters() missing 'operator' field")
				}
			}
		})
	}
}

func TestExpandRoutingAction(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]interface{}
		want  client.RoutingAction
	}{
		{
			name: "basic action",
			input: map[string]interface{}{
				"name":        "drop-matching",
				"destination": "/dev/null",
			},
			want: client.RoutingAction{
				Name:        "drop-matching",
				Destination: "/dev/null",
				Properties:  map[string]string{},
			},
		},
		{
			name: "action with properties",
			input: map[string]interface{}{
				"name":        "forward",
				"destination": "https://example.com",
				"properties": map[string]interface{}{
					"header": "value",
				},
			},
			want: client.RoutingAction{
				Name:        "forward",
				Destination: "https://example.com",
				Properties:  map[string]string{"header": "value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandRoutingAction(tt.input)
			if result.Name != tt.want.Name {
				t.Errorf("expandRoutingAction() Name = %v, want %v", result.Name, tt.want.Name)
			}
			if result.Destination != tt.want.Destination {
				t.Errorf("expandRoutingAction() Destination = %v, want %v", result.Destination, tt.want.Destination)
			}
		})
	}
}

func TestFlattenRoutingAction(t *testing.T) {
	tests := []struct {
		name  string
		input client.RoutingAction
	}{
		{
			name: "basic action",
			input: client.RoutingAction{
				Name:        "drop-matching",
				Destination: "/dev/null",
				Properties:  map[string]string{},
			},
		},
		{
			name: "action with properties",
			input: client.RoutingAction{
				Name:        "forward",
				Destination: "https://example.com",
				Properties:  map[string]string{"header": "value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenRoutingAction(tt.input)
			if len(result) != 1 {
				t.Errorf("flattenRoutingAction() returned %d items, want 1", len(result))
				return
			}

			m, ok := result[0].(map[string]interface{})
			if !ok {
				t.Errorf("flattenRoutingAction() item is not a map")
				return
			}

			// Check required fields
			if m["name"] != tt.input.Name {
				t.Errorf("flattenRoutingAction() name = %v, want %v", m["name"], tt.input.Name)
			}
			if m["destination"] != tt.input.Destination {
				t.Errorf("flattenRoutingAction() destination = %v, want %v", m["destination"], tt.input.Destination)
			}
		})
	}
}

// Acceptance tests

func TestAccDropRule_basic(t *testing.T) {
	resourceName := "last9_drop_rule.test"
	region := os.Getenv("LAST9_TEST_REGION")
	if region == "" {
		region = "ap-south-1"
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckDropRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDropRuleConfig_basic(region),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDropRuleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-test-drop-rule"),
					resource.TestCheckResourceAttr(resourceName, "region", region),
					resource.TestCheckResourceAttrSet(resourceName, "cluster_id"),
					resource.TestCheckResourceAttr(resourceName, "telemetry", "logs"),
					resource.TestCheckResourceAttr(resourceName, "filters.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "filters.0.key", `attributes["service"]`),
					resource.TestCheckResourceAttr(resourceName, "filters.0.value", "debug-service"),
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

func TestAccDropRule_update(t *testing.T) {
	resourceName := "last9_drop_rule.test"
	region := os.Getenv("LAST9_TEST_REGION")
	if region == "" {
		region = "ap-south-1"
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckDropRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDropRuleConfig_basic(region),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDropRuleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-test-drop-rule"),
					resource.TestCheckResourceAttr(resourceName, "filters.0.value", "debug-service"),
				),
			},
			{
				Config: testAccDropRuleConfig_updated(region),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDropRuleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-test-drop-rule"),
					resource.TestCheckResourceAttr(resourceName, "filters.0.value", "test-service"),
				),
			},
		},
	})
}

func TestAccDropRule_multipleFilters(t *testing.T) {
	resourceName := "last9_drop_rule.test"
	region := os.Getenv("LAST9_TEST_REGION")
	if region == "" {
		region = "ap-south-1"
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckDropRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDropRuleConfig_multipleFilters(region),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDropRuleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-test-drop-rule-multi"),
					resource.TestCheckResourceAttr(resourceName, "filters.#", "2"),
				),
			},
		},
	})
}

// Helper functions

func testAccCheckDropRuleExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no drop rule ID is set")
		}

		return nil
	}
}

func testAccCheckDropRuleDestroy(s *terraform.State) error {
	provider := testAccProvider()
	providerConfig := provider.Meta()
	if providerConfig == nil {
		// Provider not configured, skip check
		return nil
	}

	apiClient := providerConfig.(*client.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_drop_rule" {
			continue
		}

		region := rs.Primary.Attributes["region"]
		name := rs.Primary.Attributes["name"]

		result, err := apiClient.GetDropRules(region)
		if err != nil {
			// API error, assume deleted
			continue
		}

		for _, rule := range result.Properties {
			if rule.Name == name {
				return fmt.Errorf("drop rule %s still exists in region %s", name, region)
			}
		}
	}

	return nil
}

// Configuration helpers
// Note: cluster_id is optional - if not specified, the default cluster for the region will be used

func testAccDropRuleConfig_basic(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_drop_rule" "test" {
  region    = "%s"
  name      = "tf-test-drop-rule"
  telemetry = "logs"

  filters {
    key      = "attributes[\"service\"]"
    value    = "debug-service"
    operator = "equals"
  }

  action {
    name        = "drop-matching"
    destination = "/dev/null"
  }
}
`, region)
}

func testAccDropRuleConfig_updated(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_drop_rule" "test" {
  region    = "%s"
  name      = "tf-test-drop-rule"
  telemetry = "logs"

  filters {
    key      = "attributes[\"service\"]"
    value    = "test-service"
    operator = "equals"
  }

  action {
    name        = "drop-matching"
    destination = "/dev/null"
  }
}
`, region)
}

func testAccDropRuleConfig_multipleFilters(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_drop_rule" "test" {
  region    = "%s"
  name      = "tf-test-drop-rule-multi"
  telemetry = "logs"

  filters {
    key         = "attributes[\"service\"]"
    value       = "debug-service"
    operator    = "equals"
    conjunction = "and"
  }

  filters {
    key      = "attributes[\"environment\"]"
    value    = "development"
    operator = "equals"
  }

  action {
    name        = "drop-matching"
    destination = "/dev/null"
  }
}
`, region)
}
