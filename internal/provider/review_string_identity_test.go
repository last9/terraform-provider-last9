package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// The panel set hash must distinguish every edit the JSON diff suppression
// does: SDK diffSet skips set elements whose hashes match, so a coarser hash
// silently drops the edit.
func TestReviewPanelHashPreservesStringIdentity(t *testing.T) {
	type panelEdit struct {
		expr          string
		valueMappings string
		threshold     string
	}
	base := panelEdit{expr: "1", valueMappings: `[{"value":"1","text":"Down"}]`, threshold: "1"}
	config := func(e panelEdit) map[string]interface{} {
		return map[string]interface{}{"region": "us-east-1", "name": "synthetic", "panel": []interface{}{map[string]interface{}{
			"key": "p", "position": 0, "name": "P",
			"field_overrides_json": fmt.Sprintf(`[{"matcher":{"type":"byName","value":"x"},"properties":{"thresholds":[{"value":%s,"color":"red"}]}}]`, e.threshold),
			"query":                []interface{}{map[string]interface{}{"name": "A", "expr": e.expr, "telemetry": "metrics", "query_type": "promql"}},
			"layout":               []interface{}{map[string]interface{}{"x": 0, "y": 0, "w": 6, "h": 6}},
			"visualization":        []interface{}{map[string]interface{}{"type": "stat", "value_mappings_json": e.valueMappings}},
		}}}
	}
	with := func(edit func(*panelEdit)) panelEdit {
		e := base
		edit(&e)
		return e
	}
	for _, tc := range []struct {
		name string
		next panelEdit
		want bool
	}{
		{"unchanged", base, false},
		{"numeric_spelling_in_json", with(func(e *panelEdit) { e.threshold = "1.0" }), false},
		{"numeric_value_in_json", with(func(e *panelEdit) { e.threshold = "2" }), true},
		{"numeric_looking_string_in_json", with(func(e *panelEdit) { e.valueMappings = `[{"value":"1.0","text":"Down"}]` }), true},
		{"different_string_in_json", with(func(e *panelEdit) { e.valueMappings = `[{"value":"2","text":"Down"}]` }), true},
		{"added_empty_string_in_json", with(func(e *panelEdit) { e.valueMappings = `[{"value":"1","text":"Down","color":""}]` }), true},
		{"numeric_looking_plain_attribute", with(func(e *panelEdit) { e.expr = "1.0" }), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := resourceDashboard()
			state := schema.TestResourceDataRaw(t, r.Schema, config(base))
			state.SetId("synthetic")
			raw := config(tc.next)
			rc := terraform.NewResourceConfigRaw(raw)
			cv, err := schema.JSONMapToStateValue(raw, r.CoreConfigSchema())
			if err != nil {
				t.Fatal(err)
			}
			rc.CtyValue = cv
			diff, err := r.Diff(context.Background(), state.State(), rc, nil)
			if err != nil {
				t.Fatal(err)
			}
			changed := false
			if diff != nil {
				for k, v := range diff.Attributes {
					if strings.HasPrefix(k, "panel.") && v.Old != v.New {
						changed = true
					}
				}
			}
			if changed != tc.want {
				t.Fatalf("panel diff=%v, want=%v; diff=%#v", changed, tc.want, diff)
			}
		})
	}
}
