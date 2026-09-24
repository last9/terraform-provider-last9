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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func TestReviewEntityRefreshDoesNotClaimAlertBindings(t *testing.T) {
	for _, configured := range []bool{false, true} {
		name := "omitted_entity_block_preserves_alert_binding"
		if configured {
			name = "explicit_entity_owner_control"
		}
		t.Run(name, func(t *testing.T) {
			fake := newFakeNotificationServer([]client.NotificationDestination{{ID: 1, Name: "Channel A"}}, map[string]client.NotificationDestination{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/entities/entity-1") {
					_ = json.NewEncoder(w).Encode(client.Entity{ID: "entity-1", Name: "Synthetic", Type: "service", ExternalRef: "synthetic"})
					return
				}
				fake.handler().ServeHTTP(w, r)
			}))
			defer server.Close()
			c, err := client.NewClient(&client.Config{APIToken: "synthetic", Org: "test-org", BaseURL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			// Existing configuration manages bindings on last9_alert, not last9_entity.
			if err := reconcileNotificationChannels(c, "entity-1", "breach", []string{"Channel A"}); err != nil {
				t.Fatal(err)
			}
			raw := map[string]interface{}{"name": "Synthetic", "type": "service", "external_ref": "synthetic"}
			if configured {
				raw["notification_channels"] = []interface{}{newEntityChannelsBlock("breach", []string{"Channel A"})}
			}
			resource := resourceEntity()
			d := schema.TestResourceDataRaw(t, resource.Schema, raw)
			d.SetId("entity-1")
			if diags := resourceEntityRead(context.Background(), d, c); diags.HasError() {
				t.Fatal(diags)
			}
			state := d.State()
			diff, err := resource.Diff(context.Background(), state, terraform.NewResourceConfigRaw(raw), c)
			if err != nil {
				t.Fatal(err)
			}
			if diff != nil && !diff.Empty() {
				if _, diags := resource.Apply(context.Background(), state, diff, c); diags.HasError() {
					t.Fatal(diags)
				}
			}
			if _, ok := fake.live["Channel A"]; !ok {
				t.Fatalf("unchanged config removed alert paging: detached=%v", fake.detached)
			}
		})
	}
}

func TestReviewEntityEmptySeveritySurvivesRead(t *testing.T) {
	fake := newFakeNotificationServer(nil, map[string]client.NotificationDestination{})
	server := httptest.NewServer(fake.handler())
	defer server.Close()
	c, err := client.NewClient(&client.Config{APIToken: "synthetic", Org: "test-org", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	raw := map[string]interface{}{"name": "Synthetic", "type": "service", "external_ref": "synthetic", "notification_channels": []interface{}{newEntityChannelsBlock("breach", []string{})}}
	if diags := (&schema.Provider{ResourcesMap: map[string]*schema.Resource{"last9_entity": resourceEntity()}}).ValidateResource("last9_entity", terraform.NewResourceConfigRaw(raw)); diags.HasError() {
		t.Fatalf("invalid fixture: %v", diags)
	}
	d := schema.TestResourceDataRaw(t, resourceEntity().Schema, raw)
	d.SetId("entity-1")
	before := d.Get("notification_channels")
	if diags := setEntityNotificationChannels(d, c, "entity-1"); diags.HasError() {
		t.Fatal(diags)
	}
	if got := d.Get("notification_channels"); !reflect.DeepEqual(got, before) {
		t.Fatalf("successful read changed empty desired severity block: before=%#v after=%#v", before, got)
	}
}

func TestReviewEntityLookupFailureDoesNotRewriteID(t *testing.T) {
	for _, failure := range []string{"healthy", "503", "malformed"} {
		t.Run(failure, func(t *testing.T) {
			fake := newFakeNotificationServer([]client.NotificationDestination{{ID: 1, Name: "Channel A"}}, map[string]client.NotificationDestination{"Channel A": {ID: 100001, Name: "Channel A", ServiceFqid: "entity-1", Severity: "breach"}})
			gets := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					gets++
					if gets == 2 {
						if failure == "503" {
							http.Error(w, "synthetic outage", 503)
							return
						}
						if failure == "malformed" {
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
			d := schema.TestResourceDataRaw(t, resourceEntity().Schema, map[string]interface{}{"name": "Synthetic", "type": "service", "external_ref": "synthetic", "notification_channels": []interface{}{newEntityChannelsBlock("breach", []string{"1"})}})
			before := d.Get("notification_channels")
			diags := setEntityNotificationChannels(d, c, "entity-1")
			if failure == "healthy" && diags.HasError() {
				t.Fatal(diags)
			}
			if !diags.HasError() && !reflect.DeepEqual(before, d.Get("notification_channels")) {
				t.Fatalf("lookup %s returned success with rewritten ID: %#v", failure, d.Get("notification_channels"))
			}
		})
	}
}
