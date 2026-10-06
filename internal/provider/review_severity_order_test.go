package provider

import (
	"context"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func TestReviewSeverityRefreshOrder(t *testing.T) {
	for _, severities := range [][]string{{"breach"}, {"threat", "breach"}, {"breach", "threat"}} {
		t.Run(severities[0]+"_"+string(rune('0'+len(severities))), func(t *testing.T) {
			fake := newFakeNotificationServer(nil, map[string]client.NotificationDestination{})
			server := httptest.NewServer(fake.handler())
			defer server.Close()
			c, err := client.NewClient(&client.Config{APIToken: "synthetic", Org: "test-org", BaseURL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			blocks := make([]interface{}, 0, len(severities))
			for _, s := range severities {
				blocks = append(blocks, newEntityChannelsBlock(s, nil))
			}
			raw := map[string]interface{}{"name": "Synthetic", "type": "service", "external_ref": "synthetic", "notification_channels": blocks}
			resource := resourceEntity()
			for attempt := 0; attempt < 128; attempt++ {
				d := schema.TestResourceDataRaw(t, resource.Schema, raw)
				d.SetId("entity-1")
				before := d.Get("notification_channels")
				if diags := setEntityNotificationChannels(d, c, "entity-1"); diags.HasError() {
					t.Fatal(diags)
				}
				if !reflect.DeepEqual(before, d.Get("notification_channels")) {
					diff, err := resource.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(raw), c)
					if err != nil {
						t.Fatal(err)
					}
					if diff == nil || diff.Empty() {
						t.Fatal("control invalid: changed order did not affect SDK plan")
					}
					t.Fatalf("refresh %d reordered TypeList blocks without any remote/config change: before=%v after=%v diff=%v", attempt, before, d.Get("notification_channels"), diff.Attributes)
				}
			}
		})
	}
}
