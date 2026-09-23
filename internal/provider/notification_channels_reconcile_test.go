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
			_ = json.NewEncoder(w).Encode(result)

		case r.Method == http.MethodPost:
			var body client.AttachNotificationSettingsRequest
			_ = json.NewDecoder(r.Body).Decode(&body)
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
			_ = json.NewEncoder(w).Encode(f.live)

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

	if err := reconcileNotificationChannels(c, "entity-1", "breach", []string{"Channel A"}, []string{"Channel A", "Channel B"}); err != nil {
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

	// Config now only wants Channel A -- Channel B was removed. Both were
	// previously declared by this alert, so both are eligible for detach.
	if err := reconcileNotificationChannels(c, "entity-1", "breach", []string{"Channel A", "Channel B"}, []string{"Channel A"}); err != nil {
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

	if err := reconcileNotificationChannels(c, "entity-1", "breach", []string{"Channel A"}, []string{}); err != nil {
		t.Fatalf("reconcileNotificationChannels() error = %v", err)
	}

	if len(fake.live) != 0 {
		t.Errorf("expected all bindings removed, still live: %v", fake.live)
	}
}

// TestReconcileNotificationChannels_PreservesSiblingAlertBinding verifies the
// P1 fix: reconciling one alert must not delete a binding that belongs to a
// sibling alert sharing the same entity and severity. The binding API has no
// per-alert ownership, so this alert only detaches channels IT previously
// declared (prevChannels) — never a binding it never owned, even if that
// binding isn't in this alert's wantChannels.
func TestReconcileNotificationChannels_PreservesSiblingAlertBinding(t *testing.T) {
	catalog := []client.NotificationDestination{
		{ID: 1, Name: "Channel A", Type: "slack"},
		{ID: 2, Name: "Channel B", Type: "generic_webhook"},
	}
	live := map[string]client.NotificationDestination{
		"Channel A": {ID: 100001, Name: "Channel A", ServiceFqid: "entity-1", Severity: "breach"},
		// Channel B was attached by a sibling breach alert on the same entity.
		"Channel B": {ID: 100002, Name: "Channel B", ServiceFqid: "entity-1", Severity: "breach"},
	}
	fake := newFakeNotificationServer(catalog, live)
	server := httptest.NewServer(fake.handler())
	defer server.Close()

	c, err := client.NewClient(&client.Config{APIToken: "t", Org: "test-org", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// This alert only ever declared Channel A and still wants only Channel A.
	// It must not touch Channel B, which it never owned.
	if err := reconcileNotificationChannels(c, "entity-1", "breach", []string{"Channel A"}, []string{"Channel A"}); err != nil {
		t.Fatalf("reconcileNotificationChannels() error = %v", err)
	}

	if len(fake.detached) != 0 {
		t.Errorf("expected sibling binding 100002 (Channel B) preserved, but detached: %v", fake.detached)
	}
	if _, stillBound := fake.live["Channel B"]; !stillBound {
		t.Error("Channel B (sibling alert's binding) was deleted")
	}
}

// TestReconcileNotificationChannels_SeverityScoped verifies the second P1
// fix: reconciling a breach alert must not see or touch a threat-severity
// binding for the same channel name on the same entity, in either direction
// — it must attach a breach binding even if a threat one already exists
// (attach_missing_severity), and must not detach the threat one when it's
// not in the breach alert's wantChannels (preserve_other_severity).
func TestReconcileNotificationChannels_SeverityScoped(t *testing.T) {
	catalog := []client.NotificationDestination{
		{ID: 1, Name: "Channel A", Type: "slack"},
	}
	live := map[string]client.NotificationDestination{
		// Channel A is bound only for threat, not breach.
		"Channel A": {ID: 100001, Name: "Channel A", ServiceFqid: "entity-1", Severity: "threat"},
	}
	fake := newFakeNotificationServer(catalog, live)
	server := httptest.NewServer(fake.handler())
	defer server.Close()

	c, err := client.NewClient(&client.Config{APIToken: "t", Org: "test-org", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Reconciling a breach alert wanting Channel A must attach it at breach
	// severity rather than treating the threat binding as satisfying it.
	if err := reconcileNotificationChannels(c, "entity-1", "breach", nil, []string{"Channel A"}); err != nil {
		t.Fatalf("reconcileNotificationChannels() error = %v", err)
	}
	if len(fake.attached) != 1 || fake.attached[0] != "Channel A" {
		t.Fatalf("expected Channel A attached at breach severity, got attached=%v", fake.attached)
	}

	// The pre-existing threat binding (row 100001) must survive untouched.
	// The fake server keys live bindings by name, so asserting via the
	// detach log (severity-agnostic) is the unambiguous check here: nothing
	// was ever detached, meaning row 100001 was never touched.
	if len(fake.detached) != 0 {
		t.Errorf("expected the pre-existing threat binding preserved, but detached: %v", fake.detached)
	}
}

// TestReconcileNotificationChannels_AcceptsChannelID verifies the P2 fix:
// notification_channels entries that are numeric strings (as produced by
// data.last9_notification_destination.*.id, per
// examples/entity-with-alerts/main.tf) resolve by ID instead of being
// rejected by a name-only lookup.
func TestReconcileNotificationChannels_AcceptsChannelID(t *testing.T) {
	catalog := []client.NotificationDestination{
		{ID: 1, Name: "Channel A", Type: "slack"},
	}
	live := map[string]client.NotificationDestination{}
	fake := newFakeNotificationServer(catalog, live)
	server := httptest.NewServer(fake.handler())
	defer server.Close()

	c, err := client.NewClient(&client.Config{APIToken: "t", Org: "test-org", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// "1" is the numeric ID of Channel A, not its name.
	if err := reconcileNotificationChannels(c, "entity-1", "breach", nil, []string{"1"}); err != nil {
		t.Fatalf("reconcileNotificationChannels() error = %v, want channel ID \"1\" to resolve to Channel A", err)
	}
	if len(fake.attached) != 1 || fake.attached[0] != "Channel A" {
		t.Errorf("expected Channel A attached via ID lookup, got: %v", fake.attached)
	}
}
