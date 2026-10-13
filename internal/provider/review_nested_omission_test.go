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
