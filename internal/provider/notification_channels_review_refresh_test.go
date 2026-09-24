package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/last9/terraform-provider-last9/internal/client"
)

// TestReviewRefreshDoesNotLeakSiblingOrDrift guards against two bugs a code
// review found: (1) Read reporting "every live binding at this severity" as
// notification_channels would promote a sibling alert's channel into this
// alert's state on a shared entity, permanently inflating it beyond what
// this alert's own config ever asked for; (2) Read rewriting a configured
// numeric channel ID ("1") to its display name ("Channel A") would cause
// config drift on every refresh. Both are now moot for the DETACH direction
// since reconcileNotificationChannels is attach-only (see its docstring),
// but Read's reported state still matters: it's what shows up in `terraform
// plan` diffs and in TestResourceAlertImportState_AdoptsLiveBindings-style
// import flows, so it must not misrepresent what this alert actually owns.
//
//   - refresh_preserves_sibling: a sibling alert's channel on the same
//     entity/severity must never appear in this alert's refreshed
//     notification_channels.
//   - numeric_id_round_trip: configuring by ID must survive a refresh with
//     the same spelling ("1" stays "1", not rewritten to "Channel A").
func TestReviewRefreshDoesNotLeakSiblingOrDrift(t *testing.T) {
	for _, tc := range []struct {
		name       string
		configured []string
		sibling    bool
	}{
		{"refresh_preserves_sibling", []string{"Channel A"}, true},
		{"numeric_id_round_trip", []string{"1"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			catalog := []client.NotificationDestination{{ID: 1, Name: "Channel A", Type: "slack"}, {ID: 2, Name: "Channel B", Type: "slack"}}
			live := map[string]client.NotificationDestination{"Channel A": {ID: 100001, Name: "Channel A", ServiceFqid: "entity-1", Severity: "breach"}}
			if tc.sibling {
				live["Channel B"] = client.NotificationDestination{ID: 100002, Name: "Channel B", ServiceFqid: "entity-1", Severity: "breach"}
			}
			fake := newFakeNotificationServer(catalog, live)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "/alert-rules/") {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(client.Alert{ID: "alert-a", Name: "Synthetic", Severity: "breach"})
					return
				}
				fake.handler().ServeHTTP(w, r)
			}))
			defer server.Close()
			c, err := client.NewClient(&client.Config{APIToken: "synthetic", Org: "test-org", BaseURL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			raw := make([]interface{}, len(tc.configured))
			for i, v := range tc.configured {
				raw[i] = v
			}
			d := schema.TestResourceDataRaw(t, resourceAlert().Schema, map[string]interface{}{"entity_id": "entity-1", "name": "Synthetic", "severity": "breach", "notification_channels": raw})
			d.SetId("alert-a")

			if diags := resourceAlertRead(context.Background(), d, c); diags.HasError() {
				t.Fatal(diags)
			}
			got := toStringSlice(d.Get("notification_channels").([]interface{}))
			if !reflect.DeepEqual(got, tc.configured) {
				t.Errorf("notification_channels after refresh = %v, want %v (sibling leak or ID rewrite)", got, tc.configured)
			}
		})
	}
}

// TestResourceAlertImportState_AdoptsLiveBindings verifies that import seeds
// notification_channels from whatever is actually bound at the alert's
// severity. This is required by the ownership model resourceAlertRead now
// enforces: Read only ever keeps or drops entries already present in state,
// it never adds one (see the ownership note on resourceAlertRead) — so
// without this seed, an imported alert would show notification_channels as
// permanently empty despite having real bindings, since there is no prior
// config for Read to compare against.
func TestResourceAlertImportState_AdoptsLiveBindings(t *testing.T) {
	catalog := []client.NotificationDestination{{ID: 1, Name: "Channel A", Type: "slack"}}
	live := map[string]client.NotificationDestination{
		"Channel A": {ID: 100001, Name: "Channel A", ServiceFqid: "entity-1", Severity: "breach"},
	}
	fake := newFakeNotificationServer(catalog, live)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/alert-rules/") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(client.Alert{ID: "alert-a", Name: "Synthetic", Severity: "breach"})
			return
		}
		fake.handler().ServeHTTP(w, r)
	}))
	defer server.Close()

	c, err := client.NewClient(&client.Config{APIToken: "synthetic", Org: "test-org", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}

	d := schema.TestResourceDataRaw(t, resourceAlert().Schema, map[string]interface{}{})
	d.SetId("entity-1:alert-a")

	results, err := resourceAlertImportState(context.Background(), d, c)
	if err != nil {
		t.Fatalf("resourceAlertImportState() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}

	got := toStringSlice(results[0].Get("notification_channels").([]interface{}))
	if len(got) != 1 || got[0] != "Channel A" {
		t.Errorf("notification_channels after import = %v, want [Channel A]", got)
	}
}
