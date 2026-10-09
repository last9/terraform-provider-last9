package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func testRecurringMuteSchedule() map[string]interface{} {
	return map[string]interface{}{
		"enabled":  true,
		"timezone": "UTC",
		"windows": []interface{}{map[string]interface{}{
			"weekdays":   []interface{}{"mon", "wed"},
			"start_time": "09:00",
			"end_time":   "17:00",
		}},
	}
}

func TestRecurringMuteScheduleRequestShapes(t *testing.T) {
	for _, tc := range []struct {
		name  string
		patch *client.RecurringMuteSchedulePatch
		want  string
	}{
		{"absent", nil, ``},
		{"clear", &client.RecurringMuteSchedulePatch{}, `null`},
		{"replace", &client.RecurringMuteSchedulePatch{Schedule: expandRecurringMuteSchedule([]interface{}{testRecurringMuteSchedule()})}, `{"enabled":true,"timezone":"UTC","windows":[{"weekdays":["mon","wed"],"start_time":"09:00","end_time":"17:00"}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(&client.AlertUpdateRequest{RecurringMuteSchedule: tc.patch})
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]json.RawMessage
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatal(err)
			}
			value, ok := got["recurring_mute_schedule"]
			if tc.want == "" {
				if ok {
					t.Fatalf("absent schedule serialized as %s", value)
				}
				return
			}
			if !ok || string(value) != tc.want {
				t.Fatalf("recurring_mute_schedule = %s, want %s", value, tc.want)
			}
		})
	}
}

func TestResourceAlertRecurringMuteScheduleLifecycle(t *testing.T) {
	ctx := context.Background()
	schedule := testRecurringMuteSchedule()
	var createSchedule, updateSchedule json.RawMessage
	api := reviewClient(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/kpis"):
			fmt.Fprint(w, `{"id":"kpi-1","name":"CPU-abc"}`)
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/alert-rules"):
			var payload map[string]json.RawMessage
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			createSchedule = payload["recurring_mute_schedule"]
			fmt.Fprint(w, `{"id":"alert-1"}`)
		case req.Method == http.MethodPut && strings.HasSuffix(req.URL.Path, "/alert-rules/alert-1"):
			var payload map[string]json.RawMessage
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			updateSchedule = payload["recurring_mute_schedule"]
			fmt.Fprint(w, `{"id":"alert-1"}`)
		case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/alert-rules/alert-1"):
			fmt.Fprint(w, `{"id":"alert-1","rule_name":"CPU","primary_indicator":"CPU-abc","expression_args":{"CPU-abc":{"id":"kpi-1"}},"severity":"breach","recurring_mute_schedule":{"enabled":true,"timezone":"UTC","windows":[{"weekdays":["mon","wed"],"start_time":"09:00","end_time":"17:00"}]}}`)
		case req.Method == http.MethodDelete && strings.HasSuffix(req.URL.Path, "/kpis/kpi-1"):
			fmt.Fprint(w, `{}`)
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	r := resourceAlert()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		"entity_id":               "entity-1",
		"name":                    "CPU",
		"query":                   "avg(cpu_usage)",
		"severity":                "breach",
		"greater_than":            1.0,
		"less_than":               0.0,
		"equal_to":                0.0,
		"not_equal":               0.0,
		"bad_minutes":             1,
		"total_minutes":           1,
		"recurring_mute_schedule": []interface{}{schedule},
	})
	if diags := resourceAlertCreate(ctx, d, api); diags.HasError() {
		t.Fatal(diags)
	}
	if string(createSchedule) != `{"enabled":true,"timezone":"UTC","windows":[{"weekdays":["mon","wed"],"start_time":"09:00","end_time":"17:00"}]}` {
		t.Fatalf("create recurring_mute_schedule = %s", createSchedule)
	}
	if got := d.Get("recurring_mute_schedule").([]interface{}); len(got) != 1 {
		t.Fatalf("read did not retain schedule: %#v", got)
	}

	d.Set("description", "unrelated update")
	if diags := resourceAlertUpdate(ctx, d, api); diags.HasError() {
		t.Fatal(diags)
	}
	if string(updateSchedule) != string(createSchedule) {
		t.Fatalf("unrelated update lost schedule: got %s want %s", updateSchedule, createSchedule)
	}

	d.Set("recurring_mute_schedule", []interface{}{})
	if diags := resourceAlertUpdate(ctx, d, api); diags.HasError() {
		t.Fatal(diags)
	}
	if string(updateSchedule) != "null" {
		t.Fatalf("clearing schedule sent %s, want null", updateSchedule)
	}
}
