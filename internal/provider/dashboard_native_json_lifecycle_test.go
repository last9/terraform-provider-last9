package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/last9/terraform-provider-last9/internal/client"
)

const (
	nativeLifecycleLinks      = `[{"title":"Runbook","url":"https://example.test/runbook"}]`
	nativeLifecycleDataLinks  = `[{"title":"Trace","url":"https://example.test/${__data.fields.trace_id}"}]`
	nativeLifecycleTransforms = `[{"id":"organize","options":{"excludeByName":{"Time":true}}}]`
	nativeLifecycleOverrides  = `[{"matcher":{"type":"regex","value":"status.*"},"properties":{"hidden":true}}]`
	nativeLifecycleMappings   = `[{"type":"value","value":"0","text":"DOWN"}]`
	nativeLifecycleLogsConfig = `{"columns":["timestamp","body"],"row_limit":1000}`
)

// TestDashboardNativeJSONImportLifecycle replays the live QA sequence through
// Terraform core: import a dashboard carrying native JSON fields, rename it
// with those fields omitted from HCL (they must be preserved and the plan must
// settle), edit a numeric-looking string mapping, then clear them explicitly.
// Each edit must reach the API and leave an empty plan, even though the API
// returns nothing for empty values.
func TestDashboardNativeJSONImportLifecycle(t *testing.T) {
	stored := nativeLifecycleSeed()
	var puts []*client.Dashboard
	api := reviewClient(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/dashboards/dash-1"):
		case req.Method == http.MethodPut && strings.HasSuffix(req.URL.Path, "/dashboards/dash-1"):
			var payload client.DashboardRequest
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			puts = append(puts, payload.Dashboard)
			stored = nativeLifecycleStore(t, payload.Dashboard)
		case req.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
			return
		default:
			t.Errorf("unexpected dashboard request %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(client.DashboardResponse{Dashboard: stored}); err != nil {
			t.Error(err)
		}
	})
	factory := func() (*schema.Provider, error) {
		p := New()
		p.ConfigureContextFunc = func(context.Context, *schema.ResourceData) (any, diag.Diagnostics) { return api, nil }
		return p, nil
	}

	clears := map[string]string{
		"ts":        `links_json = "[]"` + "\n    data_links_json = \"[]\"\n    transformations_json = \"[]\"\n    field_overrides_json = \"[]\"",
		"layout":    `extra_json = "{}"`,
		"mappings":  `value_mappings_json = "[]"`,
		"dashboard": `links_json = "[]"`,
	}
	editedMappings := `[{"type":"value","value":"0.0","text":"DOWN"}]`
	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){"last9": factory},
		Steps: []resource.TestStep{
			{
				Config:             nativeLifecycleHCL("Imported", nil),
				ResourceName:       "last9_dashboard.test",
				ImportState:        true,
				ImportStateId:      "us-east-1:dash-1",
				ImportStatePersist: true,
			},
			// Unrelated rename with every native field omitted from HCL. The
			// harness fails the step if the post-apply plan is not empty.
			{Config: nativeLifecycleHCL("Renamed", nil)},
			// Only the mapping string changes: "0" and "0.0" are distinct
			// values, so the panel hash must allow Terraform to apply the edit.
			{Config: nativeLifecycleHCL("Renamed", map[string]string{
				"mappings": fmt.Sprintf("value_mappings_json = %q", editedMappings),
			})},
			// Explicit clears of every clearable native field.
			{Config: nativeLifecycleHCL("Renamed", clears)},
		},
	})

	if len(puts) != 3 {
		t.Fatalf("expected rename, mapping edit, and clear updates, got %d", len(puts))
	}
	renamed, edited, cleared := puts[0], puts[1], puts[2]
	if renamed.Name != "Renamed" {
		t.Errorf("rename update sent name %q", renamed.Name)
	}
	assertNativeLifecycleFields(t, "rename", renamed, map[string]string{
		"dashboard.links":     nativeLifecycleLinks,
		"ts.links":            nativeLifecycleLinks,
		"ts.data_links":       nativeLifecycleDataLinks,
		"ts.transformations":  nativeLifecycleTransforms,
		"ts.field_overrides":  nativeLifecycleOverrides,
		"ts.value_mappings":   nativeLifecycleMappings,
		"ts.layout.minH":      "4",
		"logs.logs_config":    nativeLifecycleLogsConfig,
		"logs.field_override": "",
	})
	assertNativeLifecycleFields(t, "mapping edit", edited, map[string]string{
		"ts.value_mappings": editedMappings,
	})
	assertNativeLifecycleFields(t, "clear", cleared, map[string]string{
		"dashboard.links":     "[]",
		"ts.links":            "[]",
		"ts.data_links":       "[]",
		"ts.transformations":  "[]",
		"ts.field_overrides":  "[]",
		"ts.value_mappings":   "[]",
		"ts.layout.minH":      "",
		"logs.logs_config":    nativeLifecycleLogsConfig,
		"logs.field_override": "",
	})
}

// panel is Optional+Computed so CustomizeDiff can plan it; config must still
// declare at least one panel.
func TestDashboardRequiresPanelBlock(t *testing.T) {
	factory := func() (*schema.Provider, error) {
		p := New()
		p.ConfigureContextFunc = func(context.Context, *schema.ResourceData) (any, diag.Diagnostics) { return nil, nil }
		return p, nil
	}
	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){"last9": factory},
		Steps: []resource.TestStep{{
			Config: `provider "last9" {
  org = "synthetic"
  api_base_url = "http://localhost"
}

resource "last9_dashboard" "test" {
  region = "us-east-1"
  name = "No panels"
}
`,
			PlanOnly:    true,
			ExpectError: regexp.MustCompile("at least one panel block is required"),
		}},
	})
}

func assertNativeLifecycleFields(t *testing.T, label string, sent *client.Dashboard, want map[string]string) {
	t.Helper()
	panels := map[string]*client.DashboardPanel{}
	for _, panel := range sent.Panels {
		panels[panel.ID] = panel
	}
	ts, logs := panels["panel-ts"], panels["panel-logs"]
	if ts == nil || logs == nil || ts.Visualization == nil || logs.Visualization == nil {
		t.Fatalf("%s: update lost a panel or its server id: %#v", label, sent.Panels)
	}
	minH := ""
	if value, ok := ts.Layout["minH"]; ok {
		minH = fmt.Sprint(value)
	}
	got := map[string]string{
		"dashboard.links":     string(sent.Links),
		"ts.links":            string(ts.Links),
		"ts.data_links":       string(ts.DataLinks),
		"ts.transformations":  string(ts.Transformations),
		"ts.field_overrides":  string(ts.FieldOverrides),
		"ts.value_mappings":   string(ts.Visualization.ValueMappings),
		"ts.layout.minH":      minH,
		"logs.logs_config":    string(logs.Visualization.LogsConfig),
		"logs.field_override": string(logs.FieldOverrides),
	}
	for field, value := range want {
		if !jsonStringsEqual(got[field], value) || (got[field] == "") != (value == "") {
			t.Errorf("%s update sent %s = %q, want %q", label, field, got[field], value)
		}
	}
}

func nativeLifecycleSeed() *client.Dashboard {
	return &client.Dashboard{
		ID:    "dash-1",
		Name:  "Imported",
		Links: json.RawMessage(nativeLifecycleLinks),
		Panels: []*client.DashboardPanel{
			{
				ID: "panel-ts", Name: "TS A", Version: 1, Telemetry: "metrics",
				Links:           json.RawMessage(nativeLifecycleLinks),
				DataLinks:       json.RawMessage(nativeLifecycleDataLinks),
				Transformations: json.RawMessage(nativeLifecycleTransforms),
				FieldOverrides:  json.RawMessage(nativeLifecycleOverrides),
				Layout:          map[string]any{"x": 0, "y": 0, "w": 12, "h": 6, "minH": 4},
				Visualization: &client.DashboardPanelVisualization{
					Type: "timeseries", ValueMappings: json.RawMessage(nativeLifecycleMappings),
				},
				PopulatedQueries: []*client.DashboardPanelQueryDetails{{
					Name: "A", Expr: "up", Type: "range", Telemetry: "metrics", QueryType: "promql",
					Legend: client.DashboardPanelLegend{Type: "auto"}, LegendPlacement: "bottom",
				}},
			},
			{
				ID: "panel-logs", Name: "Logs", Version: 1, Telemetry: "logs",
				Layout: map[string]any{"x": 0, "y": 6, "w": 24, "h": 8},
				Visualization: &client.DashboardPanelVisualization{
					Type: "logs", LogsConfig: json.RawMessage(nativeLifecycleLogsConfig),
				},
				PopulatedQueries: []*client.DashboardPanelQueryDetails{{
					Name: "A", Expr: `{service="api"}`, Type: "range", Telemetry: "logs", QueryType: "log_raw",
					Legend: client.DashboardPanelLegend{Type: "auto"}, LegendPlacement: "bottom",
				}},
			},
		},
	}
}

// nativeLifecycleStore mimics the API: it persists the update and returns
// nothing for empty native JSON arrays/objects.
func nativeLifecycleStore(t *testing.T, sent *client.Dashboard) *client.Dashboard {
	t.Helper()
	encoded, err := json.Marshal(sent)
	if err != nil {
		t.Fatal(err)
	}
	var stored client.Dashboard
	if err := json.Unmarshal(encoded, &stored); err != nil {
		t.Fatal(err)
	}
	stored.ID = "dash-1"
	dropEmpty := func(fields ...*json.RawMessage) {
		for _, field := range fields {
			if dashboardJSONStringIsEmpty(strings.TrimSpace(string(*field))) {
				*field = nil
			}
		}
	}
	dropEmpty(&stored.Links)
	for _, panel := range stored.Panels {
		dropEmpty(&panel.Links, &panel.DataLinks, &panel.Transformations, &panel.FieldOverrides)
		if panel.Visualization != nil {
			dropEmpty(&panel.Visualization.ValueMappings, &panel.Visualization.LogsConfig)
		}
	}
	return &stored
}

// nativeLifecycleHCL renders the imported dashboard with native JSON fields
// omitted, except for the explicit settings in extra (keyed by block).
func nativeLifecycleHCL(name string, extra map[string]string) string {
	return fmt.Sprintf(`provider "last9" {
  org = "synthetic"
  api_base_url = "http://localhost"
}

resource "last9_dashboard" "test" {
  region = "us-east-1"
  name = %q
  %s
  panel {
    key = "panel-ts"
    position = 0
    name = "TS A"
    %s
    layout {
      x = 0
      y = 0
      w = 12
      h = 6
      %s
    }
    visualization {
      type = "timeseries"
      %s
    }
    query {
      position = 0
      name = "A"
      expr = "up"
      telemetry = "metrics"
      query_type = "promql"
    }
  }
  panel {
    key = "panel-logs"
    position = 1
    name = "Logs"
    layout {
      x = 0
      y = 6
      w = 24
      h = 8
    }
    visualization {
      type = "logs"
    }
    query {
      position = 0
      name = "A"
      expr = "{service=\"api\"}"
      telemetry = "logs"
      query_type = "log_raw"
    }
  }
}
`, name, extra["dashboard"], extra["ts"], extra["layout"], extra["mappings"])
}
