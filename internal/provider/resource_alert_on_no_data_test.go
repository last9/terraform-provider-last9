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

func TestAlertOnNoDataRequestShapes(t *testing.T) {
	alert := "alert"
	for _, tc := range []struct {
		name string
		body any
		want string
	}{
		{name: "create set", body: &client.AlertCreateRequest{OnNoData: &alert}, want: `"alert"`},
		{name: "create unset", body: &client.AlertCreateRequest{}, want: ""},
		{name: "update set", body: &client.AlertUpdateRequest{OnNoData: &alert}, want: `"alert"`},
		{name: "update unset", body: &client.AlertUpdateRequest{}, want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(tc.body)
			if err != nil {
				t.Fatal(err)
			}
			var payload map[string]json.RawMessage
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatal(err)
			}
			got, ok := payload["on_no_data"]
			if tc.want == "" {
				if ok {
					t.Fatalf("unset on_no_data serialized as %s", got)
				}
				return
			}
			if !ok || string(got) != tc.want {
				t.Fatalf("on_no_data = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestAlertOnNoDataValidation(t *testing.T) {
	field := resourceAlert().Schema["on_no_data"]
	if field == nil || field.ValidateFunc == nil {
		t.Fatal("on_no_data must have schema validation")
	}
	if warnings, errors := field.ValidateFunc("alert", "on_no_data"); len(warnings) != 0 || len(errors) != 0 {
		t.Fatalf("alert rejected: warnings=%v errors=%v", warnings, errors)
	}
	if _, errors := field.ValidateFunc("ignore", "on_no_data"); len(errors) == 0 {
		t.Fatal("unsupported on_no_data value was accepted")
	}
}

func TestResourceAlertOnNoDataLifecycle(t *testing.T) {
	ctx := context.Background()
	var createValue, updateValue json.RawMessage
	readValue := any("alert")
	currentID := "alert-1"
	api := reviewClient(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/kpis"):
			fmt.Fprint(w, `{"id":"kpi-1","name":"CPU-abc"}`)
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/alert-rules"):
			var payload map[string]json.RawMessage
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			createValue = payload["on_no_data"]
			fmt.Fprint(w, `{"id":"alert-1"}`)
		case req.Method == http.MethodPut && strings.HasSuffix(req.URL.Path, "/alert-rules/alert-1"):
			var payload map[string]json.RawMessage
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			updateValue = payload["on_no_data"]
			readValue = nil
			currentID = "alert-2"
			fmt.Fprint(w, `{"id":"alert-2"}`)
		case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/alert-rules/"+currentID):
			payload, err := json.Marshal(map[string]any{
				"id": currentID, "rule_name": "CPU", "primary_indicator": "CPU-abc",
				"expression_args": map[string]any{"CPU-abc": map[string]string{"id": "kpi-1"}},
				"severity": "breach", "on_no_data": readValue,
			})
			if err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write(payload)
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
		}
	})

	d := schema.TestResourceDataRaw(t, resourceAlert().Schema, map[string]any{
		"entity_id": "entity-1", "name": "CPU", "query": "avg(cpu_usage)",
		"severity": "breach", "on_no_data": "alert",
	})
	if diags := resourceAlertCreate(ctx, d, api); diags.HasError() {
		t.Fatal(diags)
	}
	if string(createValue) != `"alert"` {
		t.Fatalf("create on_no_data = %s, want alert", createValue)
	}
	if got := d.Get("on_no_data"); got != "alert" {
		t.Fatalf("read on_no_data = %#v, want alert", got)
	}

	if err := d.Set("on_no_data", ""); err != nil {
		t.Fatal(err)
	}
	if diags := resourceAlertUpdate(ctx, d, api); diags.HasError() {
		t.Fatal(diags)
	}
	if updateValue != nil {
		t.Fatalf("unset update on_no_data = %s, want omitted", updateValue)
	}
	if d.Id() != "alert-2" {
		t.Fatalf("updated ID = %q, want alert-2", d.Id())
	}
	if got := d.Get("on_no_data"); got != "" {
		t.Fatalf("null read on_no_data = %#v, want empty", got)
	}
}
