package provider

import (
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func newEntityChannelsBlock(severity string, channels []string) map[string]interface{} {
	raw := make([]interface{}, len(channels))
	for i, c := range channels {
		raw[i] = c
	}
	return map[string]interface{}{"severity": severity, "channels": raw}
}

// TestEntitySeverityChannels_DuplicateSeverityErrors verifies two blocks for
// the same severity are rejected -- the API has exactly one binding set per
// (entity, severity), so two blocks claiming the same severity is
// inherently ambiguous about what the desired list actually is.
func TestEntitySeverityChannels_DuplicateSeverityErrors(t *testing.T) {
	_, err := parseEntitySeverityChannels([]interface{}{
		newEntityChannelsBlock("breach", []string{"Channel A"}),
		newEntityChannelsBlock("breach", []string{"Channel B"}),
	})
	if err == nil {
		t.Fatal("expected error for duplicate severity blocks, got nil")
	}
}

// TestReconcileEntityNotificationChannels_FullyReconciles verifies the
// entity resource DOES detach, unlike last9_alert's reconcile: the entity
// is the sole real owner of a (entity, severity, channel) binding (per the
// Last9 backend schema, service_fqid=entity id with no alert-rule column
// at all), so removing a channel from an entity's config must actually
// stop it from notifying.
func TestReconcileEntityNotificationChannels_FullyReconciles(t *testing.T) {
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

	prev := map[string][]string{"breach": {"Channel A", "Channel B"}}
	want := map[string][]string{"breach": {"Channel A"}} // Channel B removed

	if err := reconcileEntityNotificationChannels(c, "entity-1", prev, want); err != nil {
		t.Fatalf("reconcileEntityNotificationChannels() error = %v", err)
	}

	if len(fake.detached) != 1 || fake.detached[0] != 100002 {
		t.Errorf("expected binding 100002 (Channel B) detached, got: %v", fake.detached)
	}
	if _, stillBound := fake.live["Channel B"]; stillBound {
		t.Error("Channel B should have been detached from the entity")
	}
	if _, stillBound := fake.live["Channel A"]; !stillBound {
		t.Error("Channel A should remain bound")
	}
}

// TestReconcileEntityNotificationChannels_SeverityRemovedEntirely verifies
// that removing a whole severity's block from config detaches every
// channel at that severity, even though wantBySeverity has no key for it
// at all (only prevBySeverity does) -- reconcileEntityNotificationChannels
// must iterate the union of both maps' keys, not just wantBySeverity's.
func TestReconcileEntityNotificationChannels_SeverityRemovedEntirely(t *testing.T) {
	catalog := []client.NotificationDestination{{ID: 1, Name: "Channel A", Type: "slack"}}
	live := map[string]client.NotificationDestination{
		"Channel A": {ID: 100001, Name: "Channel A", ServiceFqid: "entity-1", Severity: "threat"},
	}
	fake := newFakeNotificationServer(catalog, live)
	server := httptest.NewServer(fake.handler())
	defer server.Close()

	c, err := client.NewClient(&client.Config{APIToken: "t", Org: "test-org", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	prev := map[string][]string{"threat": {"Channel A"}}
	want := map[string][]string{} // threat block removed entirely

	if err := reconcileEntityNotificationChannels(c, "entity-1", prev, want); err != nil {
		t.Fatalf("reconcileEntityNotificationChannels() error = %v", err)
	}

	if len(fake.detached) != 1 || fake.detached[0] != 100001 {
		t.Errorf("expected binding 100001 (Channel A, threat) detached, got: %v", fake.detached)
	}
}

// TestSetEntityNotificationChannels_PreservesConfiguredSpellingAndSurfacesExternal
// verifies Read keeps the configured ID-or-name spelling for a channel
// that's still live, and also surfaces a channel bound outside Terraform
// (by its display name) since the entity resource is the authoritative
// owner and hiding it would be misleading, not protective (contrast with
// last9_alert.notification_channels, which must never add a channel it
// doesn't own -- see resourceAlertRead).
func TestSetEntityNotificationChannels_PreservesConfiguredSpellingAndSurfacesExternal(t *testing.T) {
	catalog := []client.NotificationDestination{
		{ID: 1, Name: "Channel A", Type: "slack"},
		{ID: 2, Name: "Channel B", Type: "generic_webhook"},
	}
	live := map[string]client.NotificationDestination{
		"Channel A": {ID: 100001, Name: "Channel A", ServiceFqid: "entity-1", Severity: "breach"},
		// Channel B was attached outside Terraform (e.g. via the UI).
		"Channel B": {ID: 100002, Name: "Channel B", ServiceFqid: "entity-1", Severity: "breach"},
	}
	fake := newFakeNotificationServer(catalog, live)
	server := httptest.NewServer(fake.handler())
	defer server.Close()

	c, err := client.NewClient(&client.Config{APIToken: "t", Org: "test-org", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	d := schema.TestResourceDataRaw(t, resourceEntity().Schema, map[string]interface{}{
		"name":         "test",
		"type":         "service",
		"external_ref": "test",
		"notification_channels": []interface{}{
			newEntityChannelsBlock("breach", []string{"1"}), // configured by ID
		},
	})
	d.SetId("entity-1")

	if diags := setEntityNotificationChannels(d, c, "entity-1"); diags.HasError() {
		t.Fatalf("setEntityNotificationChannels() error = %v", diags)
	}

	got, err := entitySeverityChannels(d)
	if err != nil {
		t.Fatalf("entitySeverityChannels() error = %v", err)
	}
	channels := got["breach"]
	if len(channels) != 2 {
		t.Fatalf("got %v, want 2 channels (Channel A by configured ID + externally-bound Channel B)", channels)
	}
	foundConfigured, foundExternal := false, false
	for _, c := range channels {
		if c == "1" {
			foundConfigured = true // preserved the configured "1", not rewritten to "Channel A"
		}
		if c == "Channel B" {
			foundExternal = true
		}
	}
	if !foundConfigured {
		t.Errorf("expected configured spelling \"1\" preserved, got: %v", channels)
	}
	if !foundExternal {
		t.Errorf("expected externally-bound Channel B surfaced, got: %v", channels)
	}
}
