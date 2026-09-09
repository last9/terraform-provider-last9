package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/last9/terraform-provider-last9/internal/client"
)

// fakeNotificationServer serves ListNotificationDestinations from a fixed
// catalog and records every attach/detach call it receives, so a test can
// assert reconcileNotificationChannels made exactly the calls it should —
// no more (don't touch what's already correct) and no less (don't leave a
// removed channel bound).
type fakeNotificationServer struct {
	mu       sync.Mutex
	catalog  []client.NotificationDestination // full org channel list
	live     map[string]client.NotificationDestination
	attached []string // channel names attached during the test
	detached []int    // binding row IDs detached during the test
}

func newFakeNotificationServer(catalog []client.NotificationDestination, live map[string]client.NotificationDestination) *fakeNotificationServer {
	return &fakeNotificationServer{catalog: catalog, live: live}
}

func (f *fakeNotificationServer) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet:
			// ListNotificationDestinations is used both for the org-wide
			// catalog (name -> ID resolution) and for GetEntityNotificationBindings
			// (filtered client-side by service_fqid) — return every live binding
			// plus the catalog rows for channels with no binding yet.
			result := make([]client.NotificationDestination, 0, len(f.catalog))
			seen := map[int]bool{}
			for _, b := range f.live {
				result = append(result, b)
				seen[b.ID] = true
			}
			for _, c := range f.catalog {
				if !seen[c.ID] {
					result = append(result, c)
				}
			}
			json.NewEncoder(w).Encode(result)

		case r.Method == http.MethodPost:
			var body client.AttachNotificationSettingsRequest
			json.NewDecoder(r.Body).Decode(&body)
			// path: /notification_settings/{channelID}/attach
			var channelID int
			for _, c := range f.catalog {
				if r.URL.Path == "/api/v4/organizations/test-org/notification_settings/"+strconv.Itoa(c.ID)+"/attach" {
					channelID = c.ID
					name := c.Name
					f.attached = append(f.attached, name)
					f.live[name] = client.NotificationDestination{
						ID:          channelID + 100000, // distinct binding-row id from the master channel id
						Name:        name,
						ServiceFqid: body.EntityID,
						Severity:    body.Severity,
					}
					break
				}
			}
			json.NewEncoder(w).Encode(f.live)

		case r.Method == http.MethodDelete:
			for name, b := range f.live {
				if r.URL.Path == "/api/v4/organizations/test-org/notification_settings/"+strconv.Itoa(b.ID)+"/attach" {
					f.detached = append(f.detached, b.ID)
					delete(f.live, name)
					break
				}
			}
			w.WriteHeader(http.StatusOK)
		}
	}
}

// TestReconcileNotificationChannels_AttachesMissing verifies a channel named
// in wantChannels but not currently bound gets attached, and a channel
// that's already correctly bound is left alone (no redundant attach call).
func TestReconcileNotificationChannels_AttachesMissing(t *testing.T) {
	catalog := []client.NotificationDestination{
		{ID: 1, Name: "Channel A", Type: "slack"},
		{ID: 2, Name: "Channel B", Type: "generic_webhook"},
	}
	live := map[string]client.NotificationDestination{
		"Channel A": {ID: 100001, Name: "Channel A", ServiceFqid: "entity-1", Severity: "breach"},
	}
	fake := newFakeNotificationServer(catalog, live)
	server := httptest.NewServer(fake.handler())
	defer server.Close()

	c, err := client.NewClient(&client.Config{APIToken: "t", Org: "test-org", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if err := reconcileNotificationChannels(c, "entity-1", "breach", []string{"Channel A", "Channel B"}); err != nil {
		t.Fatalf("reconcileNotificationChannels() error = %v", err)
	}

	if len(fake.attached) != 1 || fake.attached[0] != "Channel B" {
		t.Errorf("expected only Channel B to be attached, got: %v", fake.attached)
	}
	if len(fake.detached) != 0 {
		t.Errorf("expected nothing detached, got: %v", fake.detached)
	}
}

// TestReconcileNotificationChannels_DetachesRemoved verifies a channel that
// is currently bound but no longer listed in wantChannels gets detached —
// this is the fix for a channel removed from config staying silently bound.
func TestReconcileNotificationChannels_DetachesRemoved(t *testing.T) {
	catalog := []client.NotificationDestination{
		{ID: 1, Name: "Channel A", Type: "slack"},
		{ID: 2, Name: "Channel B", Type: "generic_webhook"},
	}
	live := map[string]client.NotificationDestination{
		"Channel A": {ID: 100001, Name: "Channel A", ServiceFqid: "entity-1", Severity: "breach"},
		"Channel B": {ID: 100002, Name: "Channel B", ServiceFqid: "entity-1", Severity: "breach"},
	}
	fake := newFakeNotificationServer(catalog, live)
	server := httptest.NewServer(fake.handler())
	defer server.Close()

	c, err := client.NewClient(&client.Config{APIToken: "t", Org: "test-org", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Config now only wants Channel A -- Channel B was removed.
	if err := reconcileNotificationChannels(c, "entity-1", "breach", []string{"Channel A"}); err != nil {
		t.Fatalf("reconcileNotificationChannels() error = %v", err)
	}

	if len(fake.attached) != 0 {
		t.Errorf("expected nothing attached, got: %v", fake.attached)
	}
	if len(fake.detached) != 1 || fake.detached[0] != 100002 {
		t.Errorf("expected binding row 100002 (Channel B) detached, got: %v", fake.detached)
	}
	if _, stillBound := fake.live["Channel B"]; stillBound {
		t.Error("Channel B is still bound after being removed from config")
	}
}

// TestReconcileNotificationChannels_EmptyWantDetachesAll verifies the
// specific case that motivated this fix: notification_channels going from
// non-empty to [] must detach every existing binding, not be treated as
// "nothing to do".
func TestReconcileNotificationChannels_EmptyWantDetachesAll(t *testing.T) {
	catalog := []client.NotificationDestination{
		{ID: 1, Name: "Channel A", Type: "slack"},
	}
	live := map[string]client.NotificationDestination{
		"Channel A": {ID: 100001, Name: "Channel A", ServiceFqid: "entity-1", Severity: "breach"},
	}
	fake := newFakeNotificationServer(catalog, live)
	server := httptest.NewServer(fake.handler())
	defer server.Close()

	c, err := client.NewClient(&client.Config{APIToken: "t", Org: "test-org", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if err := reconcileNotificationChannels(c, "entity-1", "breach", []string{}); err != nil {
		t.Fatalf("reconcileNotificationChannels() error = %v", err)
	}

	if len(fake.live) != 0 {
		t.Errorf("expected all bindings removed, still live: %v", fake.live)
	}
}
