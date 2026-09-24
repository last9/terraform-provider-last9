package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/last9/terraform-provider-last9/internal/client"
)

// TestReviewResolutionFailurePropagates verifies a code-review finding: a
// transport error, non-2xx response, or malformed JSON from a channel
// lookup must never be treated the same as "channel doesn't exist" (that
// distinction is client.ErrNotificationChannelNotFound, checked via
// errors.Is). Silently swallowing any other error would let reconcile skip
// attaching a wanted channel while still reporting success.
//
// Originally also covered a "read" phase on last9_alert; that field
// (notification_channels) is now deprecated and fully inert (see
// resource_alert.go) and resourceAlertRead no longer touches it at all, so
// that phase no longer applies. Lookup-failure propagation on the read
// side is covered on the entity resource instead, by
// TestReviewEntityLookupFailureDoesNotRewriteID and
// TestReviewAttachedEntityLookupFailureDoesNotRewriteID.
func TestReviewResolutionFailurePropagates(t *testing.T) {
	for _, failure := range []string{"healthy", "503", "malformed_json"} {
		t.Run(failure, func(t *testing.T) {
			fake := newFakeNotificationServer([]client.NotificationDestination{{ID: 1, Name: "Channel A"}}, map[string]client.NotificationDestination{"Channel A": {ID: 100001, Name: "Channel A", ServiceFqid: "entity-1", Severity: "breach"}})
			gets := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			err = reconcileEntityNotificationChannelsAtSeverity(c, "entity-1", "breach", []string{"Channel A"})
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
		})
	}
}
