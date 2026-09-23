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

// TestReviewRefreshOwnershipAndIDs guards against the regression a code
// review found in the initial ownership-tracking fix: prevChannels was
// sourced from d.GetChange("notification_channels"), but Read populated
// notification_channels from the full live binding set at this severity —
// including a sibling alert's channels on a shared entity. One refresh
// therefore promoted the sibling's channel into this alert's "owned" state,
// and the next reconcile deleted it as no-longer-wanted. It also caught a
// second bug: Read rewrote a configured numeric channel ID ("1") to its
// display name ("Channel A") on every refresh, so reconcile's next call
// treated the two spellings as different channels and cycled
// attach-then-immediately-detach on unchanged configuration.
//
//   - ownership_control: no refresh — establishes the sibling survives when
//     prevChannels is exactly the as-configured value.
//   - refresh_preserves_sibling: same setup, but goes through a real
//     resourceAlertRead first (as Terraform does before every apply) to
//     prove the refreshed state doesn't leak the sibling channel into
//     prevChannels.
//   - numeric_id_round_trip: configuring by ID must survive a refresh with
//     the same spelling, and must not cause a spurious detach/attach cycle.
func TestReviewRefreshOwnershipAndIDs(t *testing.T) {
	for _, tc := range []struct {
		name       string
		configured []string
		refresh    bool
		sibling    bool
	}{
		{"ownership_control", []string{"Channel A"}, false, true},
		{"refresh_preserves_sibling", []string{"Channel A"}, true, true},
		{"numeric_id_round_trip", []string{"1"}, true, false},
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
			prior := tc.configured
			if tc.refresh {
				if diags := resourceAlertRead(context.Background(), d, c); diags.HasError() {
					t.Fatal(diags)
				}
				prior = toStringSlice(d.Get("notification_channels").([]interface{}))
				if tc.name == "numeric_id_round_trip" && !reflect.DeepEqual(prior, tc.configured) {
					t.Errorf("ID representation changed on read: got %v want %v", prior, tc.configured)
				}
			}
			// Terraform uses refreshed state as the old side of GetChange on update.
			if err := reconcileNotificationChannels(c, "entity-1", "breach", prior, tc.configured); err != nil {
				t.Fatal(err)
			}
			if len(fake.detached) != 0 {
				t.Errorf("unchanged config deleted live bindings: %v (refreshed prior=%v)", fake.detached, prior)
			}
			if tc.sibling {
				if _, ok := fake.live["Channel B"]; !ok {
					t.Error("sibling alert lost notification channel B")
				}
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
