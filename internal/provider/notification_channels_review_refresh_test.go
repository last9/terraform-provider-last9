package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/last9/terraform-provider-last9/internal/client"
)

// TestResourceAlertRead_LeavesNotificationChannelsUntouched verifies
// last9_alert's notification_channels is now fully inert: Read must not
// modify it at all, regardless of what's live at the entity/severity
// (including a channel with the same name, or another bound alongside it).
// This field previously read back live bindings under an ownership model
// (see the superseded TestReviewRefreshDoesNotLeakSiblingOrDrift for that
// history); it's now deprecated in favor of last9_entity.notification_channels
// (see resource_alert.go's schema description for why two resources
// managing the same entity/severity can't coexist safely), so Read simply
// leaves whatever the user configured untouched -- no lookups, no rewrites,
// no drift in either direction.
func TestResourceAlertRead_LeavesNotificationChannelsUntouched(t *testing.T) {
	for _, tc := range []struct {
		name       string
		configured []string
	}{
		{"configured_by_name", []string{"Channel A"}},
		{"configured_by_id", []string{"1"}},
		{"configured_empty", []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
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
			if len(got) != len(tc.configured) {
				t.Fatalf("notification_channels after refresh = %v, want unchanged %v (field is deprecated/inert)", got, tc.configured)
			}
			for i, v := range tc.configured {
				if got[i] != v {
					t.Errorf("notification_channels[%d] after refresh = %q, want unchanged %q", i, got[i], v)
				}
			}
		})
	}
}

// TestResourceAlertImportState_LeavesNotificationChannelsEmpty verifies
// import does NOT seed notification_channels from live bindings. That
// seeding existed to support the field's old attach-only ownership model;
// now that notification_channels on last9_alert is deprecated and fully
// inert (see resource_alert.go), import must not touch it at all -- there's
// nothing for this alert to "adopt," since channel management lives solely
// on last9_entity.
func TestResourceAlertImportState_LeavesNotificationChannelsEmpty(t *testing.T) {
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
	if len(got) != 0 {
		t.Errorf("notification_channels after import = %v, want empty (field is deprecated/inert)", got)
	}
}
