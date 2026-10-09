package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func TestDashboardPanelStableIdentityLifecycle(t *testing.T) {
	stored := &client.Dashboard{}
	var updates int
	api := reviewClient(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/dashboards/"):
			var payload client.DashboardRequest
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			stored = payload.Dashboard
			stored.ID = "dashboard-identity"
			assignPanelIDs(stored)
			writeIdentityDashboard(t, w, stored)
		case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/dashboards/dashboard-identity"):
			writeIdentityDashboard(t, w, stored)
		case req.Method == http.MethodPut && strings.HasSuffix(req.URL.Path, "/dashboards/dashboard-identity"):
			updates++
			var payload client.DashboardRequest
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			stored = payload.Dashboard
			stored.ID = "dashboard-identity"
			assignPanelIDs(stored)
			assertIdentityUpdate(t, updates, stored.Panels)
			writeIdentityDashboard(t, w, stored)
		case req.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected dashboard request %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
		}
	})
	factory := func() (*schema.Provider, error) {
		p := New()
		p.ConfigureContextFunc = func(context.Context, *schema.ResourceData) (any, diag.Diagnostics) { return api, nil }
		return p, nil
	}
	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){"last9": factory},
		Steps: []resource.TestStep{
			{Config: panelIdentityHCL(`
panel {
  key = "alpha"
  position = 0
  name = "Alpha"
  field_overrides_json = "[]"
  visualization {
    type = "section"
  }
}
panel {
  key = "beta"
  position = 1
  name = "Beta"
  field_overrides_json = "[{\"owner\":\"beta\"}]"
  visualization {
    type = "section"
  }
}
panel {
  key = "gamma"
  position = 2
  name = "Gamma"
  field_overrides_json = "[{\"owner\":\"gamma\"}]"
  visualization {
    type = "section"
  }
}
	`)},
			{Config: panelIdentityHCL(`
panel {
  key = "beta"
  position = 0
  name = "Beta"
  field_overrides_json = "[{\"owner\":\"beta\"}]"
  visualization {
    type = "section"
  }
}
panel {
  key = "gamma"
  position = 1
  name = "Gamma"
  field_overrides_json = "[{\"owner\":\"gamma\"}]"
  visualization {
    type = "section"
  }
}
	`)},
			{Config: panelIdentityHCL(`
panel {
  key = "gamma"
  position = 0
  name = "Gamma"
  field_overrides_json = "[{\"owner\":\"gamma\"}]"
  visualization {
    type = "section"
  }
}
panel {
  key = "beta"
  position = 1
  name = "Beta"
  field_overrides_json = "[{\"owner\":\"beta\"}]"
  visualization {
    type = "section"
  }
}
	`)},
			{Config: panelIdentityHCL(`
panel {
  key = "gamma"
  position = 0
  name = "Gamma"
  field_overrides_json = "[{\"owner\":\"gamma\"}]"
  visualization {
    type = "section"
  }
}
panel {
  key = "beta"
  position = 1
  name = "Renamed Beta"
  field_overrides_json = "[{\"owner\":\"beta\"}]"
  visualization {
    type = "section"
  }
}
	`)},
		},
	})
}

func panelIdentityHCL(panels string) string {
	return fmt.Sprintf(`provider "last9" {
  org = "synthetic"
  api_base_url = "http://localhost"
}
resource "last9_dashboard" "test" {
  region = "us-east-1"
  name = "Identity test"
%s
}
`, panels)
}

func assignPanelIDs(dashboard *client.Dashboard) {
	for _, panel := range dashboard.Panels {
		if panel.ID == "" {
			panel.ID = "panel-" + strings.ToLower(strings.ReplaceAll(panel.Name, " ", "-"))
		}
	}
}

func writeIdentityDashboard(t *testing.T, w http.ResponseWriter, dashboard *client.Dashboard) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"dashboard": dashboard}); err != nil {
		t.Error(err)
	}
}

func assertIdentityUpdate(t *testing.T, update int, panels []*client.DashboardPanel) {
	t.Helper()
	check := func(index int, id, name, jsonValue string) {
		if index >= len(panels) {
			t.Fatalf("update %d has only %d panels", update, len(panels))
		}
		panel := panels[index]
		if panel.ID != id || panel.Name != name || string(panel.FieldOverrides) != jsonValue {
			t.Errorf("update %d panel[%d] = {id:%q name:%q overrides:%s}, want {id:%q name:%q overrides:%s}", update, index, panel.ID, panel.Name, panel.FieldOverrides, id, name, jsonValue)
		}
	}
	switch update {
	case 1: // removing the first panel must not transfer either survivor's ID or JSON
		check(0, "panel-beta", "Beta", `[{"owner":"beta"}]`)
		check(1, "panel-gamma", "Gamma", `[{"owner":"gamma"}]`)
	case 2: // API order follows explicit positions, not HCL/set order
		check(0, "panel-gamma", "Gamma", `[{"owner":"gamma"}]`)
		check(1, "panel-beta", "Beta", `[{"owner":"beta"}]`)
	case 3: // rename preserves identity when key is stable
		check(1, "panel-beta", "Renamed Beta", `[{"owner":"beta"}]`)
	default:
		t.Errorf("unexpected dashboard update %d", update)
	}
}
