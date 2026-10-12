package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/last9/terraform-provider-last9/internal/client"
)

// The GET boundary is synthetic; Read, Terraform state and request expansion are real.
func TestReviewPanelRefreshPreservesEveryServerPanel(t *testing.T) {
	for _, inserted := range []bool{false, true} {
		label := "unchanged_control"
		if inserted {
			label = "ui_insert_before_existing"
		}
		t.Run(label, func(t *testing.T) {
			r := resourceDashboard()
			d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
				"region": "us-east-1", "name": "Synthetic",
				"panel": []interface{}{map[string]interface{}{
					"id": "id-alpha", "name": "Alpha",
					"visualization": []interface{}{map[string]interface{}{"type": "section"}},
				}},
			})
			if _, hasKey := r.Schema["panel"].Elem.(*schema.Resource).Schema["key"]; hasKey {
				existing := d.Get("panel").(*schema.Set).List()
				existing[0].(map[string]interface{})["key"] = "alpha"
				if err := d.Set("panel", existing); err != nil {
					t.Fatal(err)
				}
			}
			d.SetId("dashboard-test")
			panels := []*client.DashboardPanel{{ID: "id-alpha", Name: "Alpha", Visualization: &client.DashboardPanelVisualization{Type: "section"}}}
			if inserted {
				panels = append([]*client.DashboardPanel{{ID: "id-new", Name: "New", Visualization: &client.DashboardPanelVisualization{Type: "section"}}}, panels...)
			}
			api := reviewClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("unexpected method %s", r.Method)
				}
				json.NewEncoder(w).Encode(client.DashboardResponse{Dashboard: &client.Dashboard{ID: d.Id(), Name: "Synthetic", Panels: panels}})
			})
			if diags := resourceDashboardRead(context.Background(), d, api); diags.HasError() {
				t.Fatal(diags)
			}
			// A refresh cannot discard a server panel or transfer alpha's identity to another ID.
			got := buildDashboardRequest(d).Dashboard.Panels
			if len(got) != len(panels) {
				t.Errorf("refresh lost panels: got %d, want %d; got=%+v", len(got), len(panels), got)
			}
			var values []interface{}
			switch v := d.Get("panel").(type) {
			case *schema.Set:
				values = v.List()
			case []interface{}:
				values = v
			}
			for _, raw := range values {
				p := raw.(map[string]interface{})
				if p["key"] == "alpha" && p["id"] != "id-alpha" {
					t.Errorf("alpha identity reassigned to %v", p["id"])
				}
			}
		})
	}
}
