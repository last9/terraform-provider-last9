package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestDashboardMetadataOmissionPreservesServerValues(t *testing.T) {
	metadata := map[string]any{"_category": "custom", "_type": "logs", "tags": []any{"server-default"}}
	var dashboard map[string]any
	var updates int
	api := reviewClient(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/dashboards/"):
			var payload map[string]json.RawMessage
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			if value, ok := payload["metadata"]; ok && string(value) != "null" {
				t.Errorf("create with omitted metadata should not send an explicit value, got %s", value)
			}
			if err := json.Unmarshal(payload["dashboard"], &dashboard); err != nil {
				t.Error(err)
			}
			assertDashboardNestedOrder(t, dashboard)
			dashboard["id"] = "dashboard-1"
			panels := dashboard["panels"].([]interface{})
			panels[0].(map[string]interface{})["id"] = "stat-panel"
			writeDashboardMetadataResponse(t, w, dashboard, metadata)
		case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/dashboards/dashboard-1"):
			writeDashboardMetadataResponse(t, w, dashboard, metadata)
		case req.Method == http.MethodPut && strings.HasSuffix(req.URL.Path, "/dashboards/dashboard-1"):
			updates++
			var payload struct {
				Dashboard map[string]any  `json:"dashboard"`
				Metadata  json.RawMessage `json:"metadata"`
			}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			assertDashboardNestedOrder(t, payload.Dashboard)
			if len(payload.Metadata) == 0 || string(payload.Metadata) == "null" {
				t.Errorf("update %d erased metadata instead of preserving it: %s", updates, payload.Metadata)
			} else {
				var got map[string]any
				if err := json.Unmarshal(payload.Metadata, &got); err != nil {
					t.Error(err)
				}
				switch updates {
				case 1:
					if got["_category"] != "custom" || got["_type"] != "logs" || fmt.Sprint(got["tags"]) != "[server-default]" {
						t.Errorf("unrelated update did not preserve server metadata: %v", got)
					}
				case 2:
					if got["_category"] != "team" || got["_type"] != "metrics" || fmt.Sprint(got["tags"]) != "[edited]" {
						t.Errorf("explicit metadata change was not sent: %v", got)
					}
				default:
					t.Errorf("unexpected dashboard update %d", updates)
				}
				metadata = got
			}
			dashboard = payload.Dashboard
			dashboard["id"] = "dashboard-1"
			writeDashboardMetadataResponse(t, w, dashboard, metadata)
		case req.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected API request: %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
		}
	})

	factory := func() (*schema.Provider, error) {
		p := New()
		p.ConfigureContextFunc = func(_ context.Context, _ *schema.ResourceData) (any, diag.Diagnostics) {
			return api, nil
		}
		return p, nil
	}
	providerFactories := map[string]func() (*schema.Provider, error){"last9": factory}
	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: dashboardMetadataHCL("Initial", ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("last9_dashboard.test", "metadata.0.category", "custom"),
					resource.TestCheckResourceAttr("last9_dashboard.test", "metadata.0.type", "logs"),
					resource.TestCheckResourceAttr("last9_dashboard.test", "metadata.0.tags.0", "server-default"),
				),
			},
			{
				ResourceName:      "last9_dashboard.test",
				ImportState:       true,
				ImportStateId:     "us-east-1:dashboard-1",
				ImportStateVerify: true,
			},
			{
				Config: dashboardMetadataHCL("Initial", ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("last9_dashboard.test", "metadata.0.tags.0", "server-default"),
				),
			},
			{Config: dashboardMetadataHCL("Renamed", "")},
			{
				Config: dashboardMetadataHCL("Renamed", `metadata {
  category = "team"
  type = "metrics"
  tags = ["edited"]
}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("last9_dashboard.test", "metadata.0.category", "team"),
					resource.TestCheckResourceAttr("last9_dashboard.test", "metadata.0.tags.0", "edited"),
				),
			},
			{
				Config: dashboardMetadataHCL("Renamed", `metadata {
  category = "team"
  type = "metrics"
  tags = ["edited"]
}`),
			},
		},
	})
	if updates != 2 {
		t.Fatalf("expected one unrelated update and one explicit metadata update, got %d", updates)
	}
}

func writeDashboardMetadataResponse(t *testing.T, w http.ResponseWriter, dashboard, metadata map[string]any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"dashboard": dashboard, "metadata": metadata}); err != nil {
		t.Error(err)
	}
}

func dashboardMetadataHCL(name, metadata string) string {
	return fmt.Sprintf(`provider "last9" {
  org = "synthetic"
  api_base_url = "http://localhost"
}

resource "last9_dashboard" "test" {
  region = "us-east-1"
  name = %q
  panel {
    key = "stat-panel"
    position = 0
    name = "Panel"
    layout {
      x = 0
      y = 0
      w = 6
      h = 4
    }
    visualization {
      type = "stat"
      stat_config {
        threshold {
          position = 0
          value = 90
          color = "red"
        }
        threshold {
          position = 1
          value = 10
          color = "green"
        }
      }
    }
    query {
      position = 0
      name = "A"
      expr = "up"
      telemetry = "metrics"
      query_type = "promql"
    }
    query {
      position = 1
      name = "B"
      expr = "down"
      telemetry = "metrics"
      query_type = "promql"
    }
  }
  %s
}
`, name, metadata)
}

func assertDashboardNestedOrder(t *testing.T, dashboard map[string]any) {
	t.Helper()
	panels := dashboard["panels"].([]interface{})
	panel := panels[0].(map[string]interface{})
	queries := panel["queries"].([]interface{})
	if len(queries) != 2 || queries[0].(map[string]interface{})["name"] != "A" || queries[1].(map[string]interface{})["name"] != "B" {
		t.Errorf("query order was not preserved: %#v", queries)
	}
	visualization := panel["visualization"].(map[string]interface{})
	stat := visualization["stat_config"].(map[string]interface{})
	thresholds := stat["thresholds"].([]interface{})
	if len(thresholds) != 2 || thresholds[0].(map[string]interface{})["value"] != float64(90) || thresholds[1].(map[string]interface{})["value"] != float64(10) {
		t.Errorf("threshold order was not preserved: %#v", thresholds)
	}
}
