package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	c, err := NewClient(&Config{
		APIToken: "test-token",
		Org:      "test-org",
		BaseURL:  server.URL,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return c, server
}

// TestFindNotificationDestinationByName exercises the name -> ID resolution
// step of the notification-channel attach fix: the alert-rules API has no
// notification_channels field, so a channel name from Terraform config must
// be resolved against ListNotificationDestinations before it can be attached.
func TestFindNotificationDestinationByName(t *testing.T) {
	destinations := []NotificationDestination{
		{ID: 111, Name: "Other Channel", Type: "slack"},
		{ID: 58698, Name: "CreditPlus - All Alerts", Type: "generic_webhook"},
	}

	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/organizations/test-org/notification_settings" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-LAST9-API-TOKEN") != "Bearer test-token" {
			t.Fatalf("unexpected auth header: %s", r.Header.Get("X-LAST9-API-TOKEN"))
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(destinations); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})
	defer server.Close()

	dest, err := c.FindNotificationDestinationByName("CreditPlus - All Alerts")
	if err != nil {
		t.Fatalf("FindNotificationDestinationByName() error = %v", err)
	}
	if dest.ID != 58698 {
		t.Errorf("got ID %d, want 58698", dest.ID)
	}

	if _, err := c.FindNotificationDestinationByName("Does Not Exist"); err == nil {
		t.Error("expected error for unknown channel name, got nil")
	}
}

// TestFindNotificationDestinationByName_PrefersMasterRow verifies that when
// a channel has both a master row (ServiceFqid empty) and a per-entity
// binding row (ServiceFqid set) sharing its name, the master row's ID is
// returned. The binding row's ID is that binding's own row ID — not the
// master channel ID that AttachNotificationSettings/DetachNotificationSettings
// expect — so returning it would call attach/detach against an ID no
// /notification_settings/{id}/attach route recognizes.
func TestFindNotificationDestinationByName_PrefersMasterRow(t *testing.T) {
	destinations := []NotificationDestination{
		// Binding row appears first in the list, to prove ordering isn't
		// what makes this pass.
		{ID: 200001, Name: "Channel A", ServiceFqid: "entity-1", Severity: "threat"},
		{ID: 1, Name: "Channel A", Type: "slack"}, // master row
	}

	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(destinations); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})
	defer server.Close()

	dest, err := c.FindNotificationDestinationByName("Channel A")
	if err != nil {
		t.Fatalf("FindNotificationDestinationByName() error = %v", err)
	}
	if dest.ID != 1 {
		t.Errorf("got ID %d, want master row ID 1 (not binding row ID 200001)", dest.ID)
	}
}

// TestFindNotificationDestinationByName_FallsBackToBindingRow verifies a
// channel that only exists as binding rows (no master row was returned by
// this page of the API response) is still resolvable, rather than erroring.
func TestFindNotificationDestinationByName_FallsBackToBindingRow(t *testing.T) {
	destinations := []NotificationDestination{
		{ID: 200001, Name: "Channel A", ServiceFqid: "entity-1", Severity: "threat"},
	}

	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(destinations); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})
	defer server.Close()

	dest, err := c.FindNotificationDestinationByName("Channel A")
	if err != nil {
		t.Fatalf("FindNotificationDestinationByName() error = %v", err)
	}
	if dest.ID != 200001 {
		t.Errorf("got ID %d, want fallback binding row ID 200001", dest.ID)
	}
}

// TestAttachNotificationSettings verifies the actual binding call: the
// request body must carry entity_id and severity (severity is mandatory —
// the live API returns 400 "severity required" without it), and the path
// must be /notification_settings/{channel_id}/attach relative to the
// client's already-versioned base path.
func TestAttachNotificationSettings(t *testing.T) {
	var gotBody AttachNotificationSettingsRequest

	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v4/organizations/test-org/notification_settings/58698/attach" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-LAST9-API-TOKEN") != "Bearer test-token" {
			t.Fatalf("unexpected auth header: %s", r.Header.Get("X-LAST9-API-TOKEN"))
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(NotificationDestination{
			ID:          58698,
			Name:        "CreditPlus - All Alerts",
			Type:        "generic_webhook",
			ServiceFqid: gotBody.EntityID,
			Severity:    gotBody.Severity,
		}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})
	defer server.Close()

	result, err := c.AttachNotificationSettings(58698, "entity-abc-123", "breach")
	if err != nil {
		t.Fatalf("AttachNotificationSettings() error = %v", err)
	}

	if gotBody.EntityID != "entity-abc-123" {
		t.Errorf("request body entity_id = %q, want %q", gotBody.EntityID, "entity-abc-123")
	}
	if gotBody.Severity != "breach" {
		t.Errorf("request body severity = %q, want %q", gotBody.Severity, "breach")
	}
	if result.ServiceFqid != "entity-abc-123" {
		t.Errorf("response service_fqid = %q, want %q", result.ServiceFqid, "entity-abc-123")
	}
}

// TestDetachNotificationSettings verifies the detach call: DELETE against
// /notification_settings/{binding_row_id}/attach — the row's own id (as
// returned by attach or listed via ListNotificationDestinations), NOT the
// channel's master ID used to attach. Also asserts it uses the regular
// write token, not the delete-scoped token: unlike DeleteEntity/DeleteAlert,
// this endpoint sits behind normal write auth server-side (see
// ValidateDeleteNotificationSetting in last9/last9, which checks entity
// ownership, not token scope), so requiring a delete_refresh_token here
// would be an unnecessary blast-radius increase.
func TestDetachNotificationSettings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v4/organizations/test-org/notification_settings/58742/attach" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		// Configured with a distinct write token and delete token below;
		// asserting on the write token here is what proves this call does
		// NOT use the delete-scoped path (c.Delete would send "Bearer
		// delete-token" instead).
		if got := r.Header.Get("X-LAST9-API-TOKEN"); got != "Bearer write-token" {
			t.Fatalf("expected write token (Bearer write-token), got: %s", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c, err := NewClient(&Config{
		APIToken:    "write-token",
		DeleteToken: "delete-token",
		Org:         "test-org",
		BaseURL:     server.URL,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if err := c.DetachNotificationSettings(58742); err != nil {
		t.Fatalf("DetachNotificationSettings() error = %v", err)
	}
}

// TestGetEntityNotificationBindings verifies the Read-side fix: it must
// return only the rows actually bound to the given entity (service_fqid
// match), not every channel in the org, and not the alert-rule's own
// (unused) notification_channels field.
// TestGetEntityNotificationBindings uses a fake server that mimics the real
// API's confirmed production behavior: GET /notification_settings with NO
// entity_id query param returns only master/global rows (empty
// ServiceFqid) and never a per-entity bound row, even when one exists —
// verified directly against a live tenant, where the unfiltered endpoint
// returned zero bound rows for an entity with a real, confirmed-live
// binding. Only GET /notification_settings?entity_id=<id> includes that
// entity's bound rows (mixed in with the masters). A fake server that (like
// an earlier version of this test) ignores the query string and always
// returns the full fixed list would pass even if GetEntityNotificationBindings
// silently reverted to calling the unfiltered endpoint, hiding exactly the
// bug this test exists to catch.
func TestGetEntityNotificationBindings(t *testing.T) {
	masters := []NotificationDestination{
		{ID: 1, Name: "CreditPlus - All Alerts", Severity: "", ServiceFqid: ""},
		{ID: 2, Name: "Some Other Channel", Severity: "", ServiceFqid: ""},
	}
	boundByEntity := map[string][]NotificationDestination{
		"entity-a": {
			{ID: 101, Name: "CreditPlus - All Alerts", ServiceFqid: "entity-a", Severity: "breach"},
			{ID: 103, Name: "Some Other Channel", ServiceFqid: "entity-a", Severity: "threat"},
		},
		"entity-b": {
			{ID: 102, Name: "CreditPlus - All Alerts", ServiceFqid: "entity-b", Severity: "breach"},
		},
	}

	var gotQuery string
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		entityID := r.URL.Query().Get("entity_id")
		result := append([]NotificationDestination{}, masters...)
		if entityID != "" {
			result = append(result, boundByEntity[entityID]...)
		}
		if err := json.NewEncoder(w).Encode(result); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})
	defer server.Close()

	bindings, err := c.GetEntityNotificationBindings("entity-a")
	if err != nil {
		t.Fatalf("GetEntityNotificationBindings() error = %v", err)
	}
	if gotQuery != "entity_id=entity-a" {
		t.Fatalf("request query = %q, want entity_id=entity-a -- GetEntityNotificationBindings must query the entity_id-filtered endpoint, not the unfiltered list", gotQuery)
	}
	if len(bindings) != 2 {
		t.Fatalf("got %d bindings, want 2", len(bindings))
	}
	for _, b := range bindings {
		if b.ServiceFqid != "entity-a" {
			t.Errorf("got binding for %q, want only entity-a", b.ServiceFqid)
		}
	}

	empty, err := c.GetEntityNotificationBindings("entity-with-no-bindings")
	if err != nil {
		t.Fatalf("GetEntityNotificationBindings() error = %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("got %d bindings for unbound entity, want 0", len(empty))
	}
}
