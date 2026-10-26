package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestReviewPrettyEmptyDashboardLinks(t *testing.T) {
	for _, clear := range []string{"[]", "[ ]", "[\n  ]"} {
		t.Run(clear, func(t *testing.T) {
			r := resourceDashboard()
			config := func(links string) map[string]interface{} {
				return map[string]interface{}{
					"name":       "Synthetic",
					"region":     "us-east-1",
					"links_json": links,
					"panel": []interface{}{map[string]interface{}{
						"key":           "p",
						"position":      0,
						"name":          "Section",
						"visualization": []interface{}{map[string]interface{}{"type": "section"}},
					}},
				}
			}
			prior := schema.TestResourceDataRaw(t, r.Schema, config(""))
			prior.SetId("synthetic")
			raw := config(clear)
			rc := terraform.NewResourceConfigRaw(raw)
			cv, err := schema.JSONMapToStateValue(raw, r.CoreConfigSchema())
			if err != nil {
				t.Fatal(err)
			}
			rc.CtyValue = cv
			d, err := r.Diff(context.Background(), prior.State(), rc, nil)
			if err != nil {
				t.Fatal(err)
			}
			if d != nil && d.Attributes["links_json"] != nil {
				t.Fatalf("cleared links must converge for %q; got diff %#v", clear, d.Attributes["links_json"])
			}
		})
	}
}
