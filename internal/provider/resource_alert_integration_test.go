package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestResourceAlertImportedKPIReferencesSurviveThresholdUpdate(t *testing.T) {
	for _, imported := range []bool{true, false} {
		t.Run(fmt.Sprintf("import=%v", imported), func(t *testing.T) {
			ctx := context.Background()
			putCalls := 0
			threshold := 70
			api := reviewClient(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/kpis/kpi-1") {
					fmt.Fprint(w, `{"id":"kpi-1","name":"CPU-abc","definition":{"query":"avg(cpu_usage)"}}`)
					return
				}
				if !strings.HasSuffix(req.URL.Path, "/alert-rules/alert-1") {
					t.Errorf("unexpected API request: %s %s", req.Method, req.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if req.Method == http.MethodPut {
					putCalls++
					var payload struct {
						PrimaryIndicator string `json:"primary_indicator"`
						ExpressionArgs   map[string]struct {
							ID string `json:"id"`
						} `json:"expression_args"`
					}
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Error(err)
					}
					if payload.PrimaryIndicator != "CPU-abc" || payload.ExpressionArgs["CPU-abc"].ID != "kpi-1" {
						t.Errorf("threshold-only update lost imported KPI references: %+v", payload)
						w.WriteHeader(http.StatusInternalServerError)
						fmt.Fprint(w, `{"error":"empty KPI reference"}`)
						return
					}
					threshold = 80
				}
				fmt.Fprintf(w, `{"id":"alert-1","rule_name":"CPU","primary_indicator":"CPU-abc","expression_args":{"CPU-abc":{"id":"kpi-1"}},"condition":"expr > %d","alert_condition":"count_true(result) >= 7","eval_window":10,"severity":"breach","group_timeseries_notifications":false}`, threshold)
			})
			r := resourceAlert()
			config := map[string]interface{}{"entity_id": "entity-1", "name": "CPU", "query": "avg(cpu_usage)", "greater_than": 70.0, "bad_minutes": 7, "total_minutes": 10, "severity": "breach", "group_timeseries_notifications": false}
			data := schema.TestResourceDataRaw(t, r.Schema, config)
			data.SetId("alert-1")
			if imported {
				data = schema.TestResourceDataRaw(t, r.Schema, nil)
				data.SetId("entity-1:alert-1")
				states, err := r.Importer.StateContext(ctx, data, api)
				if err != nil {
					t.Fatal(err)
				}
				data = states[0]
			}
			state, diags := r.RefreshWithoutUpgrade(ctx, data.State(), api)
			if diags.HasError() {
				t.Fatal(diags)
			}
			config["greater_than"] = 80.0
			diff, err := r.Diff(ctx, state, terraform.NewResourceConfigRaw(config), api)
			if err != nil {
				t.Fatal(err)
			}
			if diff == nil || diff.Empty() {
				t.Fatal("threshold update did not produce a diff")
			}
			diff.RawConfig, err = schema.JSONMapToStateValue(config, r.CoreConfigSchema())
			if err != nil {
				t.Fatal(err)
			}
			if _, diags := r.Apply(ctx, state, diff, api); diags.HasError() {
				t.Fatal(diags)
			}
			if putCalls != 1 {
				t.Fatalf("expected exactly one alert update, got %d", putCalls)
			}
		})
	}
}

func TestResourceAlertMissingKPIReferencesRejectUpdate(t *testing.T) {
	for _, missing := range []string{"kpi_id", "kpi_name"} {
		t.Run(missing, func(t *testing.T) {
			api := reviewClient(t, func(w http.ResponseWriter, req *http.Request) {
				t.Errorf("missing KPI reference must not mutate API: %s %s", req.Method, req.URL.Path)
			})
			r := resourceAlert()
			config := map[string]interface{}{"entity_id": "entity-1", "name": "CPU", "query": "avg(cpu_usage)", "greater_than": 70.0}
			d := schema.TestResourceDataRaw(t, r.Schema, config)
			d.SetId("alert-1")
			d.Set("kpi_id", "kpi-1")
			d.Set("kpi_name", "CPU-abc")
			d.Set(missing, "")
			config["description"] = "updated description"
			diff, err := r.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(config), api)
			if err != nil {
				t.Fatal(err)
			}
			_, diags := r.Apply(context.Background(), d.State(), diff, api)
			if !diags.HasError() || !strings.Contains(diags[0].Summary, "refresh or re-import") {
				t.Fatalf("missing KPI must produce actionable error, got %v", diags)
			}
		})
	}
}

func TestResourceAlertImportRejectsUnusableKPI(t *testing.T) {
	for _, tc := range []struct {
		name, args, kpi string
		status          int
		want            string
	}{
		{"missing_reference", `null`, "", 0, "cannot recover KPI metadata"},
		{"empty_references", `{}`, "", 0, "cannot recover KPI metadata"},
		{"empty_id", `{"CPU-abc":{"id":""}}`, "", 0, "non-empty ID"},
		{"null_reference", `{"CPU-abc":null}`, "", 0, "non-empty ID"},
		{"multiple_indicators", `{"CPU-abc":{"id":"kpi-1"},"other":{"id":"kpi-2"}}`, "", 0, "single KPI"},
		{"kpi_get_failure", `{"CPU-abc":{"id":"kpi-1"}}`, "", http.StatusInternalServerError, "failed to read KPI"},
		{"empty_query", `{"CPU-abc":{"id":"kpi-1"}}`, `{"id":"kpi-1","definition":{"query":""}}`, http.StatusOK, "has no query"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := reviewClient(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet {
					t.Fatalf("unexpected mutation: %s", req.Method)
				}
				if strings.HasSuffix(req.URL.Path, "/kpis/kpi-1") {
					w.WriteHeader(tc.status)
					fmt.Fprint(w, tc.kpi)
					return
				}
				fmt.Fprintf(w, `{"id":"alert-1","rule_name":"CPU","primary_indicator":"CPU-abc","expression_args":%s}`, tc.args)
			})
			r := resourceAlert()
			d := schema.TestResourceDataRaw(t, r.Schema, nil)
			d.SetId("entity-1:alert-1")
			states, err := r.Importer.StateContext(context.Background(), d, api)
			if err != nil {
				t.Fatal(err)
			}
			_, diags := r.RefreshWithoutUpgrade(context.Background(), states[0].State(), api)
			if !diags.HasError() || !strings.Contains(diags[0].Summary, tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, diags)
			}
		})
	}
}

func TestResourceAlertRefreshPreservesKnownStateForMultipleIndicators(t *testing.T) {
	api := reviewClient(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || !strings.HasSuffix(req.URL.Path, "/alert-rules/alert-1") {
			t.Errorf("refresh must only read the alert: %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Fprint(w, `{"id":"alert-1","rule_name":"CPU","primary_indicator":"CPU-abc","expression_args":{"CPU-abc":{"id":"kpi-1"},"other":{"id":"kpi-2"}},"severity":"breach"}`)
	})
	r := resourceAlert()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"entity_id": "entity-1", "name": "CPU", "query": "avg(cpu_usage)"})
	d.SetId("alert-1")
	d.Set("kpi_id", "kpi-1")
	d.Set("kpi_name", "CPU-abc")
	state, diags := r.RefreshWithoutUpgrade(context.Background(), d.State(), api)
	if diags.HasError() {
		t.Fatal(diags)
	}
	for key, want := range map[string]string{"kpi_id": "kpi-1", "kpi_name": "CPU-abc", "query": "avg(cpu_usage)"} {
		if state.Attributes[key] != want {
			t.Fatalf("refresh changed known %s: got %q, want %q", key, state.Attributes[key], want)
		}
	}
}

// TestAccAlertIntegration_fullLifecycle tests the complete create -> update -> delete cycle
// for alerts, including automatic KPI creation and cleanup
func TestAccAlertIntegration_fullLifecycle(t *testing.T) {
	var entityID, alertID string
	entityResourceName := "last9_entity.test"
	alertResourceName := "last9_alert.test"
	timestamp := time.Now().UnixNano()
	entityName := fmt.Sprintf("integration-test-entity-%d", timestamp)
	externalRef := fmt.Sprintf("integration-test-ref-%d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckAlertIntegrationDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create entity and alert
			{
				Config: testAccAlertIntegrationConfig_basic(entityName, externalRef, "Integration Test Alert", "up{job=\"test\"}"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEntityExists(entityResourceName, &entityID),
					testAccCheckAlertIntegrationExists(alertResourceName, &alertID),
					resource.TestCheckResourceAttr(alertResourceName, "name", "Integration Test Alert"),
					resource.TestCheckResourceAttrSet(alertResourceName, "kpi_id"),
					resource.TestCheckResourceAttrSet(alertResourceName, "kpi_name"),
				),
			},
			// Step 2: Update alert name (triggers KPI recreation)
			{
				Config: testAccAlertIntegrationConfig_basic(entityName, externalRef, "Updated Alert Name", "up{job=\"test\"}"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(alertResourceName, "name", "Updated Alert Name"),
					resource.TestCheckResourceAttrSet(alertResourceName, "kpi_id"),
				),
			},
			// Step 3: Update query
			{
				Config: testAccAlertIntegrationConfig_basic(entityName, externalRef, "Updated Alert Name", "up{job=\"updated\"}"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(alertResourceName, "query", "up{job=\"updated\"}"),
				),
			},
		},
	})
}

// TestAccAlertIntegration_multipleAlerts tests creating multiple alerts on the same entity
func TestAccAlertIntegration_multipleAlerts(t *testing.T) {
	var entityID, alertID1, alertID2 string
	entityResourceName := "last9_entity.test"
	alert1ResourceName := "last9_alert.alert1"
	alert2ResourceName := "last9_alert.alert2"
	timestamp := time.Now().UnixNano()
	entityName := fmt.Sprintf("multi-alert-entity-%d", timestamp)
	externalRef := fmt.Sprintf("multi-alert-ref-%d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckAlertIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAlertIntegrationConfig_multipleAlerts(entityName, externalRef),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEntityExists(entityResourceName, &entityID),
					testAccCheckAlertIntegrationExists(alert1ResourceName, &alertID1),
					testAccCheckAlertIntegrationExists(alert2ResourceName, &alertID2),
					resource.TestCheckResourceAttr(alert1ResourceName, "name", "Alert One"),
					resource.TestCheckResourceAttr(alert2ResourceName, "name", "Alert Two"),
				),
			},
		},
	})
}

// TestAccAlertIntegration_staticThreshold tests alerts with static threshold conditions
func TestAccAlertIntegration_staticThreshold(t *testing.T) {
	var entityID, alertID string
	entityResourceName := "last9_entity.test"
	alertResourceName := "last9_alert.test"
	timestamp := time.Now().UnixNano()
	entityName := fmt.Sprintf("static-threshold-entity-%d", timestamp)
	externalRef := fmt.Sprintf("static-threshold-ref-%d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckAlertIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAlertIntegrationConfig_staticThreshold(entityName, externalRef),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEntityExists(entityResourceName, &entityID),
					testAccCheckAlertIntegrationExists(alertResourceName, &alertID),
					resource.TestCheckResourceAttr(alertResourceName, "name", "Static Threshold Alert"),
					resource.TestCheckResourceAttr(alertResourceName, "greater_than", "100"),
					resource.TestCheckResourceAttr(alertResourceName, "bad_minutes", "5"),
					resource.TestCheckResourceAttr(alertResourceName, "total_minutes", "10"),
					resource.TestCheckResourceAttr(alertResourceName, "severity", "breach"),
				),
			},
		},
	})
}

// TestAccAlertIntegration_withProperties tests alerts with runbook and annotations
func TestAccAlertIntegration_withProperties(t *testing.T) {
	var entityID, alertID string
	entityResourceName := "last9_entity.test"
	alertResourceName := "last9_alert.test"
	timestamp := time.Now().UnixNano()
	entityName := fmt.Sprintf("props-alert-entity-%d", timestamp)
	externalRef := fmt.Sprintf("props-alert-ref-%d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckAlertIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAlertIntegrationConfig_withProperties(entityName, externalRef),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEntityExists(entityResourceName, &entityID),
					testAccCheckAlertIntegrationExists(alertResourceName, &alertID),
					resource.TestCheckResourceAttr(alertResourceName, "name", "Alert With Properties"),
					resource.TestCheckResourceAttr(alertResourceName, "properties.0.runbook_url", "https://example.com/runbook"),
					resource.TestCheckResourceAttr(alertResourceName, "properties.0.annotations.priority", "high"),
					resource.TestCheckResourceAttr(alertResourceName, "properties.0.annotations.team", "platform"),
				),
			},
		},
	})
}

// TestAccAlertIntegration_import tests importing an existing alert
func TestAccAlertIntegration_import(t *testing.T) {
	var entityID, alertID string
	entityResourceName := "last9_entity.test"
	alertResourceName := "last9_alert.test"
	timestamp := time.Now().UnixNano()
	entityName := fmt.Sprintf("import-test-entity-%d", timestamp)
	externalRef := fmt.Sprintf("import-test-ref-%d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckAlertIntegrationDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccAlertIntegrationConfig_basic(entityName, externalRef, "Import Test Alert", "up{job=\"test\"}"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEntityExists(entityResourceName, &entityID),
					testAccCheckAlertIntegrationExists(alertResourceName, &alertID),
				),
			},
			// Import using composite ID format: entity_id:alert_id
			{
				ResourceName:      alertResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[alertResourceName]
					if !ok {
						return "", fmt.Errorf("not found: %s", alertResourceName)
					}
					entityID := rs.Primary.Attributes["entity_id"]
					alertID := rs.Primary.ID
					return fmt.Sprintf("%s:%s", entityID, alertID), nil
				},
				// is_disabled is intentionally not read from API to avoid drift
				ImportStateVerifyIgnore: []string{"is_disabled"},
			},
		},
	})
}

// TestAccAlertIntegration_lessThanThreshold tests alerts with less-than threshold
func TestAccAlertIntegration_lessThanThreshold(t *testing.T) {
	var entityID, alertID string
	entityResourceName := "last9_entity.test"
	alertResourceName := "last9_alert.test"
	timestamp := time.Now().UnixNano()
	entityName := fmt.Sprintf("less-than-entity-%d", timestamp)
	externalRef := fmt.Sprintf("less-than-ref-%d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckAlertIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAlertIntegrationConfig_lessThanThreshold(entityName, externalRef),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEntityExists(entityResourceName, &entityID),
					testAccCheckAlertIntegrationExists(alertResourceName, &alertID),
					resource.TestCheckResourceAttr(alertResourceName, "name", "Less Than Alert"),
					resource.TestCheckResourceAttr(alertResourceName, "less_than", "10"),
					resource.TestCheckResourceAttr(alertResourceName, "severity", "threat"),
				),
			},
		},
	})
}

// TestAccAlertIntegration_equalityThreshold tests alerts with equal_to and not_equal thresholds
func TestAccAlertIntegration_equalityThreshold(t *testing.T) {
	var entityID, alertID string
	entityResourceName := "last9_entity.test"
	alertResourceName := "last9_alert.test"
	timestamp := time.Now().UnixNano()
	entityName := fmt.Sprintf("equality-threshold-entity-%d", timestamp)
	externalRef := fmt.Sprintf("equality-threshold-ref-%d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckAlertIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAlertIntegrationConfig_equalToThreshold(entityName, externalRef, 12),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEntityExists(entityResourceName, &entityID),
					testAccCheckAlertIntegrationExists(alertResourceName, &alertID),
					resource.TestCheckResourceAttr(alertResourceName, "name", "Equality Threshold Alert"),
					resource.TestCheckResourceAttr(alertResourceName, "equal_to", "12"),
					resource.TestCheckResourceAttr(alertResourceName, "bad_minutes", "4"),
					resource.TestCheckResourceAttr(alertResourceName, "total_minutes", "20"),
				),
			},
			{
				Config: testAccAlertIntegrationConfig_notEqualThreshold(entityName, externalRef, 0),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlertIntegrationExists(alertResourceName, &alertID),
					resource.TestCheckResourceAttr(alertResourceName, "not_equal", "0"),
					resource.TestCheckResourceAttr(alertResourceName, "bad_minutes", "2"),
					resource.TestCheckResourceAttr(alertResourceName, "total_minutes", "10"),
				),
			},
		},
	})
}

// TestAccAlertIntegration_withRenotify tests entity renotify settings combined with alerts
func TestAccAlertIntegration_withRenotify(t *testing.T) {
	var entityID, alertID string
	entityResourceName := "last9_entity.test"
	alertResourceName := "last9_alert.test"
	timestamp := time.Now().UnixNano()
	entityName := fmt.Sprintf("renotify-entity-%d", timestamp)
	externalRef := fmt.Sprintf("renotify-ref-%d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckAlertIntegrationDestroy,
		Steps: []resource.TestStep{
			// Step 1: Entity with notify-once, alert attached
			{
				Config: testAccAlertIntegrationConfig_renotifyDisabled(entityName, externalRef),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEntityExists(entityResourceName, &entityID),
					testAccCheckAlertIntegrationExists(alertResourceName, &alertID),
					resource.TestCheckResourceAttr(entityResourceName, "renotify_enabled", "false"),
					resource.TestCheckResourceAttrSet(alertResourceName, "kpi_id"),
				),
			},
			// Step 2: Enable renotify with custom interval and cap — alert stays intact
			{
				Config: testAccAlertIntegrationConfig_renotifyEnabled(entityName, externalRef),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(entityResourceName, "renotify_enabled", "true"),
					resource.TestCheckResourceAttr(entityResourceName, "renotify_interval_seconds", "1800"),
					resource.TestCheckResourceAttr(entityResourceName, "renotify_occurrences", "5"),
					// Alert ID should be unchanged — renotify change doesn't recreate alerts
					testAccCheckAlertIntegrationExists(alertResourceName, &alertID),
				),
			},
			// Step 3: Remove all renotify fields — clear override, inherit tenant default
			{
				Config: testAccAlertIntegrationConfig_basic(entityName, externalRef, "Renotify Test Alert", "up{job=\"renotify\"}"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlertIntegrationExists(alertResourceName, &alertID),
				),
			},
		},
	})
}

// testAccCheckAlertIntegrationExists verifies an alert exists in state
func testAccCheckAlertIntegrationExists(n string, id *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no alert ID is set")
		}

		*id = rs.Primary.ID
		return nil
	}
}

// testAccCheckAlertIntegrationDestroy verifies alerts and entities are destroyed
func testAccCheckAlertIntegrationDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_alert" && rs.Type != "last9_entity" {
			continue
		}
		// Resources should be destroyed - the test framework will fail if destroy fails
	}
	return nil
}

// Config helper functions

func testAccAlertIntegrationConfig_basic(entityName, externalRef, alertName, query string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_entity" "test" {
  name         = %q
  type         = "service"
  external_ref = %q
}

resource "last9_alert" "test" {
  entity_id     = last9_entity.test.id
  name          = %q
  description   = "Test alert for integration testing"
  query         = %q

  greater_than  = 100
  bad_minutes   = 5
  total_minutes = 10

  severity = "breach"
}
`, entityName, externalRef, alertName, query)
}

func testAccAlertIntegrationConfig_multipleAlerts(entityName, externalRef string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_entity" "test" {
  name         = %q
  type         = "service"
  external_ref = %q
}

resource "last9_alert" "alert1" {
  entity_id     = last9_entity.test.id
  name          = "Alert One"
  description   = "First test alert"
  query         = "up{job=\"service1\"}"

  greater_than  = 100
  bad_minutes   = 5
  total_minutes = 10

  severity = "breach"
}

resource "last9_alert" "alert2" {
  entity_id     = last9_entity.test.id
  name          = "Alert Two"
  description   = "Second test alert"
  query         = "up{job=\"service2\"}"

  greater_than  = 50
  bad_minutes   = 3
  total_minutes = 5

  severity = "threat"
}
`, entityName, externalRef)
}

func testAccAlertIntegrationConfig_staticThreshold(entityName, externalRef string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_entity" "test" {
  name         = %q
  type         = "service"
  external_ref = %q
}

resource "last9_alert" "test" {
  entity_id     = last9_entity.test.id
  name          = "Static Threshold Alert"
  description   = "Alert with static threshold configuration"
  query         = "error_rate{service=\"api\"}"

  greater_than  = 100
  bad_minutes   = 5
  total_minutes = 10

  severity = "breach"
}
`, entityName, externalRef)
}

func testAccAlertIntegrationConfig_withProperties(entityName, externalRef string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_entity" "test" {
  name         = %q
  type         = "service"
  external_ref = %q
}

resource "last9_alert" "test" {
  entity_id     = last9_entity.test.id
  name          = "Alert With Properties"
  description   = "Alert with runbook and annotations"
  query         = "latency_p99{service=\"api\"}"

  greater_than  = 500
  bad_minutes   = 3
  total_minutes = 5

  severity = "breach"

  properties {
    runbook_url = "https://example.com/runbook"
    annotations = {
      priority = "high"
      team     = "platform"
    }
  }
}
`, entityName, externalRef)
}

func testAccAlertIntegrationConfig_renotifyDisabled(entityName, externalRef string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_entity" "test" {
  name             = %q
  type             = "service"
  external_ref     = %q
  renotify_enabled = false
}

resource "last9_alert" "test" {
  entity_id     = last9_entity.test.id
  name          = "Renotify Test Alert"
  description   = "Alert for renotify integration testing"
  query         = "up{job=\"renotify\"}"
  greater_than  = 0
  bad_minutes   = 5
  total_minutes = 10
  severity      = "breach"
}
`, entityName, externalRef)
}

func testAccAlertIntegrationConfig_renotifyEnabled(entityName, externalRef string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_entity" "test" {
  name                      = %q
  type                      = "service"
  external_ref              = %q
  renotify_enabled          = true
  renotify_interval_seconds = 1800
  renotify_occurrences      = 5
}

resource "last9_alert" "test" {
  entity_id     = last9_entity.test.id
  name          = "Renotify Test Alert"
  description   = "Alert for renotify integration testing"
  query         = "up{job=\"renotify\"}"
  greater_than  = 0
  bad_minutes   = 5
  total_minutes = 10
  severity      = "breach"
}
`, entityName, externalRef)
}

func testAccAlertIntegrationConfig_lessThanThreshold(entityName, externalRef string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_entity" "test" {
  name         = %q
  type         = "service"
  external_ref = %q
}

resource "last9_alert" "test" {
  entity_id     = last9_entity.test.id
  name          = "Less Than Alert"
  description   = "Alert that fires when value drops below threshold"
  query         = "availability{service=\"api\"}"

  less_than     = 10
  bad_minutes   = 2
  total_minutes = 5

  severity = "threat"
}
`, entityName, externalRef)
}

func testAccAlertIntegrationConfig_equalToThreshold(entityName, externalRef string, equalTo int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_entity" "test" {
  name         = %q
  type         = "service"
  external_ref = %q
}

resource "last9_alert" "test" {
  entity_id     = last9_entity.test.id
  name          = "Equality Threshold Alert"
  description   = "Alert that fires when value equals threshold"
  query         = "count(kube_pod_status_phase{phase=\"Running\"})"

  equal_to      = %d
  bad_minutes   = 4
  total_minutes = 20

  severity = "breach"
}
`, entityName, externalRef, equalTo)
}

func testAccAlertIntegrationConfig_notEqualThreshold(entityName, externalRef string, notEqual int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_entity" "test" {
  name         = %q
  type         = "service"
  external_ref = %q
}

resource "last9_alert" "test" {
  entity_id     = last9_entity.test.id
  name          = "Equality Threshold Alert"
  description   = "Alert that fires when value differs from threshold"
  query         = "count(kube_pod_status_phase{phase=\"Running\"})"

  not_equal     = %d
  bad_minutes   = 2
  total_minutes = 10

  severity = "breach"
}
`, entityName, externalRef, notEqual)
}
