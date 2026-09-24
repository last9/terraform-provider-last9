package provider

import (
	"context"
	"net/http"
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
// TestSetEntityNotificationChannels_PreservesConfiguredSpellingWithoutClaimingUndeclaredChannels
// verifies Read keeps the configured ID-or-name spelling for a channel
// that's still live, but does NOT surface a channel bound outside
// Terraform for this same, already-configured severity. Surfacing it was
// this test's original expectation, but a code review correctly pointed
// out that "the entity is the authoritative owner, so show everything
// live" doesn't hold when a severity's channels are actually managed via
// last9_alert.notification_channels (attach-only, still supported): the
// next apply would then delete that externally/alert-managed channel as
// "no longer configured." See setEntityNotificationChannels's docstring.
func TestSetEntityNotificationChannels_PreservesConfiguredSpellingWithoutClaimingUndeclaredChannels(t *testing.T) {
	catalog := []client.NotificationDestination{
		{ID: 1, Name: "Channel A", Type: "slack"},
		{ID: 2, Name: "Channel B", Type: "generic_webhook"},
	}
	live := map[string]client.NotificationDestination{
		"Channel A": {ID: 100001, Name: "Channel A", ServiceFqid: "entity-1", Severity: "breach"},
		// Channel B is bound at the same severity but never configured here
		// (e.g. attached via last9_alert or the UI) -- must not be adopted.
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
	if len(channels) != 1 || channels[0] != "1" {
		t.Fatalf("got %v, want [\"1\"] (configured spelling preserved, Channel B not adopted)", channels)
	}
}

// TestReviewAttachedEntityLookupFailureDoesNotRewriteID guards against a
// code-review finding: setEntityNotificationChannels's per-value
// resolveNotificationChannel failure handling treated ANY error (transport
// failure, non-2xx response, malformed JSON, or a genuine not-found) as
// "drop this configured value" -- the same defect already fixed on the
// alert Read path in resourceAlertRead (see
// TestReviewResolutionFailurePropagates), just not carried over to this
// newer entity path. With configured channel "1" and a live binding for
// "Channel A" (id 1), a failed lookup used to silently drop "1" from the
// resolved-channels loop, and the still-live "Channel A" name would then
// get added back by the "remaining" pass -- rewriting state from the
// configured ID "1" to the display name "Channel A" and manufacturing a
// spurious plan diff from a transient API failure, not an actual
// configuration change. (Named "Attached" to avoid colliding with the
// review's own copyable TestReviewEntityLookupFailureDoesNotRewriteID in
// notification_channels_review_entity_lifecycle_test.go, which covers the
// same defect via a slightly different harness — both are kept per the
// review's instruction to rename rather than delete either.)
func TestReviewAttachedEntityLookupFailureDoesNotRewriteID(t *testing.T) {
	for _, failure := range []string{"healthy", "503", "malformed_json"} {
		t.Run(failure, func(t *testing.T) {
			catalog := []client.NotificationDestination{{ID: 1, Name: "Channel A", Type: "slack"}}
			live := map[string]client.NotificationDestination{
				"Channel A": {ID: 100001, Name: "Channel A", ServiceFqid: "entity-1", Severity: "breach"},
			}
			fake := newFakeNotificationServer(catalog, live)
			gets := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					gets++
					// GET #1 is GetEntityNotificationBindings (must succeed so
					// the binding is visible); GET #2 is the per-value
					// resolveNotificationChannel("1") lookup -- fail only that one.
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

			diags := setEntityNotificationChannels(d, c, "entity-1")

			if failure == "healthy" {
				if diags.HasError() {
					t.Fatalf("setEntityNotificationChannels() error = %v", diags)
				}
				got, err := entitySeverityChannels(d)
				if err != nil {
					t.Fatalf("entitySeverityChannels() error = %v", err)
				}
				if channels := got["breach"]; len(channels) != 1 || channels[0] != "1" {
					t.Errorf("healthy control: notification_channels = %v, want [\"1\"]", channels)
				}
				return
			}

			// A failed lookup must abort the read, not silently rewrite "1"
			// to "Channel A" and report success.
			if !diags.HasError() {
				got, _ := entitySeverityChannels(d)
				t.Fatalf("expected error propagated from failed channel lookup (response=%s), got success with notification_channels = %v", failure, got["breach"])
			}
		})
	}
}

// TestResourceEntityImportState_AdoptsLiveBindings verifies import seeds
// notification_channels from every severity with a live binding, since
// setEntityNotificationChannels (used by ordinary Read) now only examines
// a severity already present in config -- without this explicit seed, a
// freshly imported entity would show notification_channels as empty
// forever despite having real bindings, because import starts with no
// config at all for Read to compare against.
func TestResourceEntityImportState_AdoptsLiveBindings(t *testing.T) {
	catalog := []client.NotificationDestination{
		{ID: 1, Name: "Channel A", Type: "slack"},
		{ID: 2, Name: "Channel B", Type: "slack"},
	}
	live := map[string]client.NotificationDestination{
		"Channel A": {ID: 100001, Name: "Channel A", ServiceFqid: "entity-1", Severity: "breach"},
		"Channel B": {ID: 100002, Name: "Channel B", ServiceFqid: "entity-1", Severity: "threat"},
	}
	fake := newFakeNotificationServer(catalog, live)
	server := httptest.NewServer(fake.handler())
	defer server.Close()

	c, err := client.NewClient(&client.Config{APIToken: "t", Org: "test-org", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	d := schema.TestResourceDataRaw(t, resourceEntity().Schema, map[string]interface{}{})
	d.SetId("entity-1")

	results, err := resourceEntityImportState(context.Background(), d, c)
	if err != nil {
		t.Fatalf("resourceEntityImportState() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}

	got, err := entitySeverityChannels(results[0])
	if err != nil {
		t.Fatalf("entitySeverityChannels() error = %v", err)
	}
	if channels := got["breach"]; len(channels) != 1 || channels[0] != "Channel A" {
		t.Errorf("breach severity after import = %v, want [Channel A]", channels)
	}
	if channels := got["threat"]; len(channels) != 1 || channels[0] != "Channel B" {
		t.Errorf("threat severity after import = %v, want [Channel B]", channels)
	}
}
