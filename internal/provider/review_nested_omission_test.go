package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func TestReviewDashboardNativeJSONOmissionAndClear(t *testing.T) {
	const links = `[{"title":"Docs","url":"https://example.test"}]`
	const overrides = `[{"matcher":{"type":"regex","value":"status.*"},"properties":{"hidden":true}}]`

	config := func(native string) map[string]interface{} {
		panel := map[string]interface{}{
			"key": "alpha", "position": 0, "name": "Alpha",
			"visualization": []interface{}{map[string]interface{}{"type": "section"}},
		}
		if native != "" {
			panel["links_json"] = native
			panel["field_overrides_json"] = native
		}
		return map[string]interface{}{
			"region": "us-east-1", "name": "Before", "links_json": native,
			"panel": []interface{}{panel},
		}
	}

	for _, tc := range []struct {
		name       string
		configured string
		want       string
	}{
		{name: "omitted", configured: "", want: links},
		{name: "clear", configured: "[]", want: "[]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := resourceDashboard()
			before := config(links)
			before["panel"].([]interface{})[0].(map[string]interface{})["field_overrides_json"] = overrides
			d := schema.TestResourceDataRaw(t, r.Schema, before)
			d.SetId("synthetic")

			after := config(tc.configured)
			after["name"] = "After"
			resourceConfig := terraform.NewResourceConfigRaw(after)
			rawConfig, err := schema.JSONMapToStateValue(after, r.CoreConfigSchema())
			if err != nil {
				t.Fatal(err)
			}
			resourceConfig.CtyValue = rawConfig
			diff, err := r.Diff(context.Background(), d.State(), resourceConfig, nil)
			if err != nil {
				t.Fatal(err)
			}
			diff.RawConfig = resourceConfig.CtyValue

			var sent *client.Dashboard
			stored := &client.Dashboard{ID: "synthetic", Links: json.RawMessage(links)}
			api := reviewClient(t, func(w http.ResponseWriter, req *http.Request) {
				switch req.Method {
				case http.MethodGet:
					if err := json.NewEncoder(w).Encode(client.DashboardResponse{Dashboard: stored}); err != nil {
						t.Fatal(err)
					}
					return
				case http.MethodPut:
					var payload client.DashboardRequest
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatal(err)
					}
					sent = payload.Dashboard
					stored = sent
				default:
					t.Fatalf("unexpected %s", req.Method)
				}
				if err := json.NewEncoder(w).Encode(client.DashboardResponse{Dashboard: stored}); err != nil {
					t.Fatal(err)
				}
			})
			if _, diags := r.Apply(context.Background(), d.State(), diff, api); diags.HasError() {
				t.Fatal(diags)
			}
			if got := string(sent.Links); got != tc.want {
				t.Fatalf("dashboard links sent %q, want %q", got, tc.want)
			}
			if got := string(sent.Panels[0].Links); got != tc.want {
				t.Fatalf("panel links sent %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReviewNestedOmission(t *testing.T) {
	for _, field := range []string{"logs_config_json", "extra_json"} {
		for _, explicit := range []bool{false, true} {
			t.Run(field+map[bool]string{false: "/omitted", true: "/explicit"}[explicit], func(t *testing.T) {
				config := func(preserve bool) map[string]interface{} {
					viz := map[string]interface{}{"type": "logs"}
					layout := map[string]interface{}{"x": 0, "y": 0, "w": 12, "h": 8}
					if field != "logs_config_json" || preserve {
						viz["logs_config_json"] = `{"columns":["body"],"row_limit":100}`
					}
					if field == "extra_json" && preserve {
						layout["extra_json"] = `{"minH":6,"static":true}`
					}
					return map[string]interface{}{"region": "us-east-1", "name": "Before", "panel": []interface{}{map[string]interface{}{"key": "alpha", "position": 0, "name": "Alpha", "version": 1, "visualization": []interface{}{viz}, "layout": []interface{}{layout}, "query": []interface{}{map[string]interface{}{"position": 0, "name": "A", "expr": "{}", "telemetry": "logs", "query_type": "log_ql"}}}}}
				}
				r := resourceDashboard()
				before := config(true)
				d := schema.TestResourceDataRaw(t, r.Schema, before)
				d.SetId("synthetic")
				after := config(explicit)
				after["name"] = "After"
				diff, err := r.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(after), nil)
				if err != nil {
					t.Fatalf("unrelated rename must plan with imported omitted field: %v", err)
				}
				var sent *client.Dashboard
				api := reviewClient(t, func(w http.ResponseWriter, req *http.Request) {
					if req.Method == http.MethodPut {
						var p client.DashboardRequest
						if err := json.NewDecoder(req.Body).Decode(&p); err != nil {
							t.Error(err)
							return
						}
						sent = p.Dashboard
					}
					if sent == nil {
						t.Error("missing update")
						return
					}
					if err := json.NewEncoder(w).Encode(client.DashboardResponse{Dashboard: sent}); err != nil {
						t.Error(err)
					}
				})
				_, diags := r.Apply(context.Background(), d.State(), diff, api)
				if diags.HasError() {
					t.Fatal(diags)
				}
				if field == "logs_config_json" {
					if len(sent.Panels[0].Visualization.LogsConfig) == 0 {
						t.Fatal("logs config lost")
					}
				} else {
					if sent.Panels[0].Layout["minH"] != float64(6) || sent.Panels[0].Layout["static"] != true {
						t.Fatalf("layout extras lost: %#v", sent.Panels[0].Layout)
					}
				}
			})
		}
	}
}
