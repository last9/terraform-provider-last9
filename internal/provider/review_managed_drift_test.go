package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func TestReviewManagedSeverityDrift(t *testing.T) {
	for _, tc := range []struct {
		name            string
		declared, extra bool
		want            []string
	}{
		{"exact_match_control", true, false, []string{"Channel A"}},
		{"undeclared_severity_control", false, true, nil},
		{"declared_empty_rejects_external_binding", true, true, nil},
		{"declared_subset_rejects_external_binding", true, true, []string{"Channel A"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeNotificationServer([]client.NotificationDestination{{ID: 1, Name: "Channel A"}, {ID: 2, Name: "Channel B"}}, map[string]client.NotificationDestination{})
			if len(tc.want) > 0 {
				fake.live["Channel A"] = client.NotificationDestination{ID: 100001, Name: "Channel A", ServiceFqid: "entity-1", Severity: "breach"}
			}
			if tc.extra {
				fake.live["Channel B"] = client.NotificationDestination{ID: 100002, Name: "Channel B", ServiceFqid: "entity-1", Severity: "breach"}
			}
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
			raw := map[string]interface{}{"name": "Synthetic", "type": "service", "external_ref": "synthetic"}
			if tc.declared {
				raw["notification_channels"] = []interface{}{newEntityChannelsBlock("breach", tc.want)}
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
			changed := diff != nil && !diff.Empty()
			if changed {
				if _, diags := resource.Apply(context.Background(), state, diff, c); diags.HasError() {
					t.Fatal(diags)
				}
			}
			_, extraStillLive := fake.live["Channel B"]
			if tc.declared && tc.extra && (!changed || extraStillLive) {
				t.Fatalf("explicit managed severity failed to remove drift: plan_changed=%v extra_live=%v detached=%v", changed, extraStillLive, fake.detached)
			}
			if !tc.declared && !extraStillLive {
				t.Fatal("undeclared severity lost externally managed binding")
			}
			if !tc.extra && changed {
				t.Fatal("exact-match control unexpectedly drifted")
			}
		})
	}
}
