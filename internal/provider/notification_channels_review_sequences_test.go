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

// TestReviewResolutionFailurePropagates verifies a code-review finding: a
// transport error, non-2xx response, or malformed JSON from a channel
// lookup must never be treated the same as "channel doesn't exist" (that
// distinction is client.ErrNotificationChannelNotFound, checked via
// errors.Is). Silently swallowing any other error would let Read commit a
// state that's missing a channel it couldn't confirm one way or the other,
// or let reconcile skip attaching a wanted channel while still reporting
// success.
//
//   - read/*: resourceAlertRead must fail rather than silently drop
//     "Channel A" from notification_channels when the channel lookup used
//     to confirm it's still live errors out.
//   - attach/*: reconcileNotificationChannels must fail rather than
//     silently skip attaching "Channel A" when resolving it errors out.
func TestReviewResolutionFailurePropagates(t *testing.T) {
	for _, phase := range []string{"read", "attach"} {
		for _, failure := range []string{"healthy", "503", "malformed_json"} {
			t.Run(phase+"/"+failure, func(t *testing.T) {
				fake := newFakeNotificationServer([]client.NotificationDestination{{ID: 1, Name: "Channel A"}}, map[string]client.NotificationDestination{"Channel A": {ID: 100001, Name: "Channel A", ServiceFqid: "entity-1", Severity: "breach"}})
				gets := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.Contains(r.URL.Path, "/alert-rules/") {
						_ = json.NewEncoder(w).Encode(client.Alert{ID: "alert-a", Name: "Synthetic", Severity: "breach"})
						return
					}
					if r.Method == http.MethodGet {
						gets++
						// The binding list succeeds. Fail ONLY the subsequent channel lookup.
						if gets == 2 {
							if failure == "503" {
								http.Error(w, "synthetic temporary failure", 503)
								return
							}
							if failure == "malformed_json" {
								_, _ = w.Write([]byte("{"))
								return
							}
						}
					}
					fake.handler().ServeHTTP(w, r)
				}))
				defer server.Close()
				c, err := client.NewClient(&client.Config{APIToken: "synthetic", Org: "test-org", BaseURL: server.URL})
				if err != nil {
					t.Fatal(err)
				}
				if phase == "read" {
					d := schema.TestResourceDataRaw(t, resourceAlert().Schema, map[string]interface{}{"entity_id": "entity-1", "name": "Synthetic", "severity": "breach", "notification_channels": []interface{}{"Channel A"}})
					d.SetId("alert-a")
					diags := resourceAlertRead(context.Background(), d, c)
					owned := toStringSlice(d.Get("notification_channels").([]interface{}))
					// A failed lookup must abort refresh, not commit a false empty state.
					if !diags.HasError() && (len(owned) != 1 || owned[0] != "Channel A") {
						t.Fatalf("successful refresh lost ownership: %v; lookup response=%s", owned, failure)
					}
					if failure == "healthy" && diags.HasError() {
						t.Fatal(diags)
					}
				} else {
					err := reconcileNotificationChannels(c, "entity-1", "breach", []string{"Channel A"})
					// The channel is already bound (live map seeds it), so a
					// healthy run makes no attach call at all. On failure, the
					// lookup still runs (to resolve the config value against
					// the catalog) and must propagate the error rather than
					// silently treating "couldn't resolve" as "already fine."
					if failure != "healthy" && err == nil {
						t.Fatalf("expected error propagated from failed channel lookup, got nil")
					}
					if failure == "healthy" && err != nil {
						t.Fatal(err)
					}
				}
			})
		}
	}
}
