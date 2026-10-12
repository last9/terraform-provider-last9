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

func TestAlertRecurringMuteScheduleCanBeClearedWithEmptyAttribute(t *testing.T) {
	var schedule any = map[string]any{
		"enabled":  true,
		"timezone": "UTC",
		"windows": []any{map[string]any{
			"weekdays": []string{"mon"}, "start_time": "09:00", "end_time": "10:00",
		}},
	}
	var updates int
	api := reviewClient(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/kpis"):
			fmt.Fprint(w, `{"id":"kpi-1","name":"cpu-kpi","definition":{"query":"avg(cpu)"}}`)
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/alert-rules"):
			fmt.Fprint(w, `{"id":"alert-1"}`)
		case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/alert-rules/alert-1"):
			payload, err := json.Marshal(map[string]any{
				"id": "alert-1", "rule_name": "CPU", "primary_indicator": "cpu-kpi",
				"expression_args": map[string]any{"cpu-kpi": map[string]string{"id": "kpi-1"}},
				"severity":        "breach", "group_timeseries_notifications": true,
				"recurring_mute_schedule": schedule,
			})
			if err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write(payload)
		case req.Method == http.MethodPut && strings.HasSuffix(req.URL.Path, "/alert-rules/alert-1"):
			updates++
			var payload map[string]json.RawMessage
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			if string(payload["recurring_mute_schedule"]) != "null" {
				t.Errorf("expected explicit schedule clear as JSON null, got %s", payload["recurring_mute_schedule"])
			}
			schedule = nil
			fmt.Fprint(w, `{"id":"alert-1"}`)
		case req.Method == http.MethodDelete:
			fmt.Fprint(w, `{}`)
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
	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){"last9": factory},
		Steps: []resource.TestStep{
			{Config: recurringMuteScheduleHCL(`recurring_mute_schedule {
  enabled = true
  timezone = "UTC"
  windows {
    weekdays = ["mon"]
    start_time = "09:00"
    end_time = "10:00"
  }
}`)},
			{Config: recurringMuteScheduleHCL("")},
			{Config: recurringMuteScheduleHCL("recurring_mute_schedule = null")},
			{Config: recurringMuteScheduleHCL("recurring_mute_schedule = []"), Check: resource.TestCheckResourceAttr("last9_alert.test", "recurring_mute_schedule.#", "0")},
			{Config: recurringMuteScheduleHCL("recurring_mute_schedule = []"), Check: resource.TestCheckResourceAttr("last9_alert.test", "recurring_mute_schedule.#", "0")},
		},
	})
	if updates != 1 {
		t.Fatalf("expected exactly one schedule-clear update and no follow-up drift, got %d updates", updates)
	}
}

func recurringMuteScheduleHCL(schedule string) string {
	return fmt.Sprintf(`provider "last9" {
  org = "synthetic"
  api_base_url = "http://localhost"
}

resource "last9_alert" "test" {
  entity_id = "entity-1"
  name = "CPU"
  query = "avg(cpu)"
  severity = "breach"
  %s
}
`, schedule)
}
