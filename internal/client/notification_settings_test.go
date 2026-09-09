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
		json.NewEncoder(w).Encode(destinations)
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
		json.NewEncoder(w).Encode(NotificationDestination{
			ID:          58698,
			Name:        "CreditPlus - All Alerts",
			Type:        "generic_webhook",
			ServiceFqid: gotBody.EntityID,
			Severity:    gotBody.Severity,
		})
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

// TestGetEntityNotificationBindings verifies the Read-side fix: it must
// return only the rows actually bound to the given entity (service_fqid
// match), not every channel in the org, and not the alert-rule's own
// (unused) notification_channels field.
func TestGetEntityNotificationBindings(t *testing.T) {
	destinations := []NotificationDestination{
		{ID: 1, Name: "CreditPlus - All Alerts", ServiceFqid: "entity-a", Severity: "breach"},
		{ID: 2, Name: "CreditPlus - All Alerts", ServiceFqid: "entity-b", Severity: "breach"},
		{ID: 3, Name: "Some Other Channel", ServiceFqid: "entity-a", Severity: "threat"},
	}

	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(destinations)
	})
	defer server.Close()

	bindings, err := c.GetEntityNotificationBindings("entity-a")
	if err != nil {
		t.Fatalf("GetEntityNotificationBindings() error = %v", err)
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
