package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAlertSnoozeReadKeepsConfiguredUntil(t *testing.T) {
	const configured = 1893456000
	c := reviewClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"alert_snoozed_until":0}`)
	})
	resource := resourceAlertSnooze()
	config := map[string]interface{}{"entity_id": "entity-a", "until": configured}
	d := schema.TestResourceDataRaw(t, resource.Schema, config)
	d.SetId("entity-a")

	if diags := resource.ReadContext(context.Background(), d, c); diags.HasError() {
		t.Fatal(diags)
	}
	if got := d.Get("until"); got != configured {
		t.Fatalf("until=%v, want configured value %d", got, configured)
	}
	if got := d.Get("alert_snoozed_until"); got != 0 {
		t.Fatalf("alert_snoozed_until=%v, want 0", got)
	}
	diff, err := resource.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(config), c)
	if err != nil {
		t.Fatal(err)
	}
	if diff != nil && diff.Attributes["until"] != nil {
		t.Fatal("refresh of effective snooze created an until diff")
	}
}
