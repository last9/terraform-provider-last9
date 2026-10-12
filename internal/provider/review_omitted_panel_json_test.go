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

func TestReviewRenamePreservesOmittedPanelJSON(t *testing.T) {
	for _, name := range []string{"explicit_value_control", "omitted_value", "renamed_panel"} {
		t.Run(name, func(t *testing.T) {
			const original = `[{"matcher":{"type":"regex","value":"status.*"},"properties":{"hidden":true}}]`
			r := resourceDashboard()
			config := func(panelName string) map[string]interface{} {
				return map[string]interface{}{"region": "us-east-1", "name": "After", "panel": []interface{}{map[string]interface{}{"key": "alpha", "position": 0, "name": panelName, "visualization": []interface{}{map[string]interface{}{"type": "section"}}}}}
			}
			before := config("Alpha")
			before["name"] = "Before"
			panel := before["panel"].([]interface{})[0].(map[string]interface{})
			panel["id"] = "id-alpha"
			panel["field_overrides_json"] = original
			d := schema.TestResourceDataRaw(t, r.Schema, before)
			d.SetId("synthetic")
			after := config("Alpha")
			if name == "explicit_value_control" {
				after["panel"].([]interface{})[0].(map[string]interface{})["field_overrides_json"] = original
			}
			if name == "renamed_panel" {
				after["panel"].([]interface{})[0].(map[string]interface{})["name"] = "Renamed"
			}
			diff, err := r.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(after), nil)
			if err != nil {
				t.Fatal(err)
			}
			var sent *client.Dashboard
			api := reviewClient(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodPut {
					var payload client.DashboardRequest
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatal(err)
					}
					sent = payload.Dashboard
				}
				if sent == nil {
					t.Fatal("read before update")
				}
				if err := json.NewEncoder(w).Encode(client.DashboardResponse{Dashboard: sent}); err != nil {
					t.Fatal(err)
				}
			})
			_, diags := r.Apply(context.Background(), d.State(), diff, api)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if sent == nil || len(sent.Panels) != 1 {
				t.Fatal("expected one updated panel")
			}
			if sent.Panels[0].ID != "id-alpha" {
				t.Fatalf("identity lost: %q", sent.Panels[0].ID)
			}
			if !jsonStringsEqual(string(sent.Panels[0].FieldOverrides), original) {
				t.Fatalf("omitted field overrides lost during update: got %q want %q", sent.Panels[0].FieldOverrides, original)
			}
		})
	}
}
