package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const unknownConfigValue = "74D93920-ED26-11E3-AC10-0800200C9A66"

func TestDataSourceUserRequiresSelectorAtPlan(t *testing.T) {
	provider := New()
	for name, config := range map[string]map[string]interface{}{
		"missing":  {},
		"id":       {"id": "u1"},
		"unknown":  {"id": unknownConfigValue},
		"conflict": {"id": "u1", "email": "u1@example.test"},
	} {
		t.Run(name, func(t *testing.T) {
			diags := provider.ValidateDataSource("last9_user", terraform.NewResourceConfigRaw(config))
			if gotError := diags.HasError(); gotError != (name == "missing" || name == "conflict") {
				t.Fatalf("validation errors=%v", diags)
			}
		})
	}
}

func TestDataSourceWithoutDefaultFails(t *testing.T) {
	c := reviewClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/organizations/synthetic/datasources" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprint(w, `[{"id":"first","name":"first","is_default":false}]`)
	})
	d := schema.TestResourceDataRaw(t, dataSourceDatasource().Schema, map[string]interface{}{})
	if diags := dataSourceDatasource().ReadContext(context.Background(), d, c); !diags.HasError() {
		t.Fatal("read succeeded without a default datasource")
	}
}

func TestDataSourceClusterKeepsConfiguredRegion(t *testing.T) {
	c := reviewClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("region") != "us-east-1" {
			t.Fatalf("region query=%q", r.URL.Query().Get("region"))
		}
		fmt.Fprint(w, `{"data":[{"id":"cluster-1","name":"cluster","region":"other-region","default":true}]}`)
	})
	resource := dataSourceCluster()
	d := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{"region": "us-east-1", "id": "cluster-1"})
	if diags := resource.ReadContext(context.Background(), d, c); diags.HasError() {
		t.Fatal(diags)
	}
	if got := d.Get("region"); got != "us-east-1" {
		t.Fatalf("region=%q, want configured region", got)
	}
}
