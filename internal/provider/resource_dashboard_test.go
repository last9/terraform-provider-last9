package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/last9/terraform-provider-last9/internal/client"
)

// Acceptance tests

func TestAccDashboard_basic(t *testing.T) {
	resourceName := "last9_dashboard.test"
	region := testAccDashboardRegion()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckDashboardDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardConfig_basic(region),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDashboardExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "TF Test Basic Dashboard"),
					resource.TestCheckResourceAttr(resourceName, "region", region),
					resource.TestCheckResourceAttr(resourceName, "panel.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "panel.0.name", "Container Memory"),
					resource.TestCheckResourceAttr(resourceName, "panel.0.visualization.0.type", "stat"),
					resource.TestCheckResourceAttr(resourceName, "panel.0.query.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "panel.0.id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "created_by"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccDashboardImportID(resourceName, region),
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.#",
					"metadata.0.category",
					"metadata.0.type",
				},
			},
		},
	})
}

func TestAccDashboard_multiPanelWithSection(t *testing.T) {
	resourceName := "last9_dashboard.test"
	region := testAccDashboardRegion()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckDashboardDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardConfig_multiPanelWithSection(region),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDashboardExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "panel.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "panel.0.visualization.0.type", "section"),
					resource.TestCheckResourceAttr(resourceName, "panel.0.name", "Spend at a Glance"),
					resource.TestCheckResourceAttr(resourceName, "panel.1.visualization.0.type", "stat"),
					resource.TestCheckResourceAttr(resourceName, "panel.2.visualization.0.type", "bar"),
					resource.TestCheckResourceAttr(resourceName, "panel.2.visualization.0.bar_config.0.orientation", "vertical"),
					resource.TestCheckResourceAttr(resourceName, "panel.2.visualization.0.bar_config.0.stacked", "true"),
				),
			},
		},
	})
}

func TestAccDashboard_withVariables(t *testing.T) {
	resourceName := "last9_dashboard.test"
	region := testAccDashboardRegion()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckDashboardDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardConfig_withVariables(region),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDashboardExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "variable.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "variable.0.display_name", "Account"),
					resource.TestCheckResourceAttr(resourceName, "variable.0.type", "label"),
					resource.TestCheckResourceAttr(resourceName, "variable.0.source", "aws_account_id"),
					resource.TestCheckResourceAttr(resourceName, "variable.1.display_name", "Region"),
					resource.TestCheckResourceAttr(resourceName, "relative_time", "10080"),
				),
			},
		},
	})
}

func TestAccDashboard_update(t *testing.T) {
	resourceName := "last9_dashboard.test"
	region := testAccDashboardRegion()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckDashboardDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardConfig_basic(region),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDashboardExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "TF Test Basic Dashboard"),
				),
			},
			{
				Config: testAccDashboardConfig_basicUpdated(region),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDashboardExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "TF Test Basic Dashboard Updated"),
					resource.TestCheckResourceAttr(resourceName, "panel.0.name", "Container Memory Updated"),
				),
			},
		},
	})
}

// Validation tests (plan-time errors)

func TestAccDashboard_ValidationSectionWithQuery(t *testing.T) {
	region := testAccDashboardRegion()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccDashboardConfig_sectionWithQuery(region),
				ExpectError: regexp.MustCompile(`section panels cannot have query blocks`),
			},
		},
	})
}

func TestAccDashboard_ValidationNonSectionMissingLayout(t *testing.T) {
	region := testAccDashboardRegion()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccDashboardConfig_nonSectionMissingLayout(region),
				ExpectError: regexp.MustCompile(`require a layout block`),
			},
		},
	})
}

func TestAccDashboard_ValidationLabelVariableMissingSource(t *testing.T) {
	region := testAccDashboardRegion()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccDashboardConfig_labelVariableMissingSource(region),
				ExpectError: regexp.MustCompile(`source is required when type=label`),
			},
		},
	})
}

// Unit tests (no API)

func TestDashboard_ExpandPanels_BasicStat(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDashboard().Schema, map[string]interface{}{
		"region": "ap-south-1",
		"name":   "test",
		"panel": []interface{}{
			map[string]interface{}{
				"name":    "stat panel",
				"unit":    "bytes-iec",
				"layout":  []interface{}{map[string]interface{}{"x": 0, "y": 0, "w": 6, "h": 6}},
				"visualization": []interface{}{
					map[string]interface{}{
						"type":       "stat",
						"full_width": false,
					},
				},
				"query": []interface{}{
					map[string]interface{}{
						"name":             "A",
						"expr":             "avg(container_memory_usage_bytes)",
						"telemetry":        "metrics",
						"query_type":       "promql",
						"legend_type":      "auto",
						"legend_placement": "right",
					},
				},
			},
		},
	})

	panels := expandPanels(d.Get("panel").([]interface{}))
	if len(panels) != 1 {
		t.Fatalf("expected 1 panel, got %d", len(panels))
	}
	p := panels[0]
	if p.Name != "stat panel" || p.Unit != "bytes-iec" {
		t.Errorf("panel mismatch: %+v", p)
	}
	if p.Visualization == nil || p.Visualization.Type != "stat" {
		t.Errorf("visualization missing or wrong type")
	}
	if len(p.PopulatedQueries) != 1 || p.PopulatedQueries[0].Expr != "avg(container_memory_usage_bytes)" {
		t.Errorf("query expr wrong")
	}
	if p.Layout["w"] != 6 {
		t.Errorf("layout w wrong: %v", p.Layout["w"])
	}
}

func TestDashboard_ExpandPanels_SectionNoLayoutNoQueries(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDashboard().Schema, map[string]interface{}{
		"region": "ap-south-1",
		"name":   "test",
		"panel": []interface{}{
			map[string]interface{}{
				"name": "Section A",
				"visualization": []interface{}{
					map[string]interface{}{"type": "section", "full_width": true},
				},
			},
		},
	})

	panels := expandPanels(d.Get("panel").([]interface{}))
	if panels[0].Layout != nil {
		t.Errorf("section panel should have nil layout, got %v", panels[0].Layout)
	}
	if len(panels[0].PopulatedQueries) != 0 {
		t.Errorf("section panel should have 0 queries, got %d", len(panels[0].PopulatedQueries))
	}
}

func TestDashboard_ExpandPanels_BarWithConfig(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDashboard().Schema, map[string]interface{}{
		"region": "ap-south-1",
		"name":   "test",
		"panel": []interface{}{
			map[string]interface{}{
				"name":   "bar panel",
				"layout": []interface{}{map[string]interface{}{"x": 0, "y": 0, "w": 6, "h": 6}},
				"visualization": []interface{}{
					map[string]interface{}{
						"type": "bar",
						"bar_config": []interface{}{
							map[string]interface{}{
								"orientation": "horizontal",
								"stacked":     true,
							},
						},
					},
				},
				"query": []interface{}{
					map[string]interface{}{
						"name":       "A",
						"expr":       "sum by (svc) (rate(http_requests_total[5m]))",
						"telemetry":  "metrics",
						"query_type": "promql",
					},
				},
			},
		},
	})

	panels := expandPanels(d.Get("panel").([]interface{}))
	bc := panels[0].Visualization.BarConfig
	if bc == nil {
		t.Fatal("bar_config not expanded")
	}
	if bc.Orientation != "horizontal" || bc.Stacked == nil || !*bc.Stacked {
		t.Errorf("bar_config wrong: %+v", bc)
	}
}

func TestDashboard_ExpandVariables_LabelType(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDashboard().Schema, map[string]interface{}{
		"region": "ap-south-1",
		"name":   "test",
		"panel": []interface{}{
			map[string]interface{}{
				"name":   "x",
				"layout": []interface{}{map[string]interface{}{"x": 0, "y": 0, "w": 6, "h": 6}},
				"visualization": []interface{}{
					map[string]interface{}{"type": "stat"},
				},
				"query": []interface{}{
					map[string]interface{}{"name": "A", "expr": "1"},
				},
			},
		},
		"variable": []interface{}{
			map[string]interface{}{
				"display_name":   "Account",
				"target":         "account",
				"type":           "label",
				"source":         "aws_account_id",
				"matches":        []interface{}{`aws_cost_unblended_USD{cost_date!=""}`},
				"multiple":       true,
				"current_values": []interface{}{".*"},
			},
		},
	})

	vars := expandVariables(d.Get("variable").([]interface{}))
	if len(vars) != 1 {
		t.Fatalf("expected 1 var, got %d", len(vars))
	}
	v := vars[0]
	if v.Source != "aws_account_id" || v.Type != "label" || !v.Multiple {
		t.Errorf("var fields wrong: %+v", v)
	}
	if len(v.Matches) != 1 {
		t.Errorf("matches not expanded: %+v", v.Matches)
	}
}

func TestDashboard_FlattenLayout_RoundTrip(t *testing.T) {
	in := map[string]any{"x": float64(1), "y": float64(2), "w": float64(3), "h": float64(4)}
	out := flattenLayout(in)
	if len(out) != 1 {
		t.Fatalf("expected 1 layout, got %d", len(out))
	}
	m := out[0].(map[string]interface{})
	if m["x"].(int) != 1 || m["w"].(int) != 3 {
		t.Errorf("flatten failed: %+v", m)
	}
}

func TestDashboard_TableConfigJSON_OpaqueRoundTrip(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDashboard().Schema, map[string]interface{}{
		"region": "ap-south-1",
		"name":   "test",
		"panel": []interface{}{
			map[string]interface{}{
				"name":   "tbl",
				"layout": []interface{}{map[string]interface{}{"x": 0, "y": 0, "w": 6, "h": 6}},
				"visualization": []interface{}{
					map[string]interface{}{
						"type":              "table",
						"table_config_json": `{"density":"compact","columnConfig":[{"key":"svc","width":200}],"thresholds":[{"value":100,"color":"red"}]}`,
					},
				},
				"query": []interface{}{
					map[string]interface{}{"name": "A", "expr": "1", "telemetry": "metrics", "query_type": "promql"},
				},
			},
		},
	})

	panels := expandPanels(d.Get("panel").([]interface{}))
	if panels[0].Visualization.TableConfig == nil {
		t.Fatal("table_config not parsed")
	}
	blob := panels[0].Visualization.TableConfig.(map[string]interface{})
	if blob["density"] != "compact" {
		t.Errorf("density mismatch: %v", blob["density"])
	}
	cc := blob["columnConfig"].([]interface{})
	if len(cc) != 1 {
		t.Errorf("columnConfig not preserved: %v", cc)
	}
	first := cc[0].(map[string]interface{})
	if first["key"] != "svc" {
		t.Errorf("columnConfig.key wrong: %v", first)
	}
}

func TestDashboard_LegendSortAndMatrix_RoundTrip(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDashboard().Schema, map[string]interface{}{
		"region": "ap-south-1",
		"name":   "test",
		"panel": []interface{}{
			map[string]interface{}{
				"name":   "p",
				"layout": []interface{}{map[string]interface{}{"x": 0, "y": 0, "w": 6, "h": 6}},
				"visualization": []interface{}{
					map[string]interface{}{"type": "stat"},
				},
				"query": []interface{}{
					map[string]interface{}{
						"name":                  "A",
						"expr":                  "1",
						"telemetry":             "metrics",
						"query_type":            "promql",
						"legend_sort_field":     "value",
						"legend_sort_direction": "desc",
						"matrix_json":           `{"transform":"transpose"}`,
					},
				},
			},
		},
	})

	panels := expandPanels(d.Get("panel").([]interface{}))
	q := panels[0].PopulatedQueries[0]
	if q.LegendSort == nil || q.LegendSort.Field != "value" || q.LegendSort.Direction != "desc" {
		t.Errorf("legend_sort wrong: %+v", q.LegendSort)
	}
	if q.Matrix == nil {
		t.Fatal("matrix not parsed")
	}
	m := q.Matrix.(map[string]interface{})
	if m["transform"] != "transpose" {
		t.Errorf("matrix transform wrong: %v", m)
	}
}

func TestDashboard_JSONStringsEqual(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{`{"a":1,"b":2}`, `{"b":2,"a":1}`, true},
		{`{"a":1}`, `{"a":2}`, false},
		{`{}`, `{}`, true},
		{"", "", true},
		{`{"a":1}`, `not json`, false},
	}
	for _, c := range cases {
		if got := jsonStringsEqual(c.a, c.b); got != c.want {
			t.Errorf("jsonStringsEqual(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestDashboard_Validate_TimeXOR(t *testing.T) {
	cases := []struct {
		name    string
		rel     int
		from    int
		to      int
		wantErr string
	}{
		{"both_set", 60, 1700000000000, 1700003600000, "cannot be combined"},
		{"only_from", 0, 1700000000000, 0, "must be set together"},
		{"only_to", 0, 0, 1700003600000, "must be set together"},
		{"neither", 0, 0, 0, ""},
		{"only_relative", 60, 0, 0, ""},
		{"only_absolute_pair", 0, 1700000000000, 1700003600000, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := map[string]interface{}{
				"region":        "ap-south-1",
				"name":          "x",
				"relative_time": tc.rel,
				"absolute_from": tc.from,
				"absolute_to":   tc.to,
				"panel": []interface{}{
					map[string]interface{}{
						"name":   "p",
						"layout": []interface{}{map[string]interface{}{"x": 0, "y": 0, "w": 6, "h": 6}},
						"visualization": []interface{}{
							map[string]interface{}{"type": "stat"},
						},
						"query": []interface{}{map[string]interface{}{"name": "A", "expr": "1", "telemetry": "metrics", "query_type": "promql"}},
					},
				},
			}
			d := schema.TestResourceDataRaw(t, resourceDashboard().Schema, raw)
			err := validateDashboardData(d)
			if tc.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestDashboard_Validate_VersionRequiresTelemetry(t *testing.T) {
	cases := []struct {
		name      string
		version   int
		telemetry string
		queryType string
		wantErr   string
	}{
		{"v1_missing_telemetry", 1, "", "promql", "telemetry is required"},
		{"v1_missing_query_type", 1, "metrics", "", "query_type is required"},
		{"v1_mismatch", 1, "metrics", "log_ql", "is not valid for telemetry"},
		{"v1_match", 1, "metrics", "promql", ""},
		{"v0_unset_ok", 0, "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := map[string]interface{}{
				"region": "ap-south-1",
				"name":   "x",
				"panel": []interface{}{
					map[string]interface{}{
						"name":    "p",
						"version": tc.version,
						"layout":  []interface{}{map[string]interface{}{"x": 0, "y": 0, "w": 6, "h": 6}},
						"visualization": []interface{}{
							map[string]interface{}{"type": "stat"},
						},
						"query": []interface{}{
							map[string]interface{}{
								"name":       "A",
								"expr":       "1",
								"telemetry":  tc.telemetry,
								"query_type": tc.queryType,
							},
						},
					},
				},
			}
			d := schema.TestResourceDataRaw(t, resourceDashboard().Schema, raw)
			err := validateDashboardData(d)
			if tc.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestDashboard_FlattenVariables_NonStringValuesCoerced(t *testing.T) {
	vars := []*client.DashboardVariable{
		{
			DisplayName:   "Threshold",
			Target:        "t",
			Type:          "static",
			Values:        []interface{}{float64(42), true, "ok"},
			CurrentValues: []any{float64(1.5)},
		},
	}
	out := flattenVariables(vars)
	if len(out) != 1 {
		t.Fatalf("expected 1 var, got %d", len(out))
	}
	m := out[0].(map[string]interface{})
	values := m["values"].([]string)
	if len(values) != 3 {
		t.Fatalf("expected 3 values (none dropped), got %d: %v", len(values), values)
	}
	if values[0] != "42" || values[1] != "true" || values[2] != "ok" {
		t.Errorf("coerced values wrong: %v", values)
	}
	current := m["current_values"].([]string)
	if len(current) != 1 || current[0] != "1.5" {
		t.Errorf("current_values coerce wrong: %v", current)
	}
}

func TestDashboard_FlattenLayout_PreservesExtraKeys(t *testing.T) {
	in := map[string]any{
		"x":     float64(1),
		"y":     float64(2),
		"w":     float64(3),
		"h":     float64(4),
		"minH":  float64(2),
		"i":     "panel-id",
		"static": true,
	}
	out := flattenLayout(in)
	m := out[0].(map[string]interface{})
	extra, ok := m["extra_json"].(string)
	if !ok || extra == "" {
		t.Fatalf("extra_json not preserved: %v", m)
	}
	var blob map[string]interface{}
	if err := json.Unmarshal([]byte(extra), &blob); err != nil {
		t.Fatalf("extra_json invalid: %v", err)
	}
	if _, ok := blob["minH"]; !ok {
		t.Error("minH lost")
	}
	if _, ok := blob["i"]; !ok {
		t.Error("i lost")
	}
}

func TestDashboard_BuildRequest_SetsAbsoluteTime(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDashboard().Schema, map[string]interface{}{
		"region":        "ap-south-1",
		"name":          "test",
		"absolute_from": 1700000000000,
		"absolute_to":   1700003600000,
		"panel": []interface{}{
			map[string]interface{}{
				"name":   "p",
				"layout": []interface{}{map[string]interface{}{"x": 0, "y": 0, "w": 6, "h": 6}},
				"visualization": []interface{}{
					map[string]interface{}{"type": "stat"},
				},
				"query": []interface{}{map[string]interface{}{"name": "A", "expr": "1", "telemetry": "metrics", "query_type": "promql"}},
			},
		},
	})

	req := buildDashboardRequest(d)
	if req.Dashboard.Time == nil {
		t.Fatal("time not set")
	}
	if req.Dashboard.Time.From == nil || *req.Dashboard.Time.From != 1700000000000 {
		t.Errorf("from wrong: %v", req.Dashboard.Time.From)
	}
	if req.Dashboard.Time.To == nil || *req.Dashboard.Time.To != 1700003600000 {
		t.Errorf("to wrong: %v", req.Dashboard.Time.To)
	}
	if req.Dashboard.Time.RelativeTime != nil {
		t.Errorf("relative_time should be nil when absolute set, got %v", *req.Dashboard.Time.RelativeTime)
	}
}

func TestDashboard_BuildRequest_SetsRelativeTime(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDashboard().Schema, map[string]interface{}{
		"region":        "ap-south-1",
		"name":          "test",
		"relative_time": 10080,
		"panel": []interface{}{
			map[string]interface{}{
				"name":   "p",
				"layout": []interface{}{map[string]interface{}{"x": 0, "y": 0, "w": 6, "h": 6}},
				"visualization": []interface{}{
					map[string]interface{}{"type": "stat"},
				},
				"query": []interface{}{map[string]interface{}{"name": "A", "expr": "1"}},
			},
		},
	})

	req := buildDashboardRequest(d)
	if req.Dashboard.Time == nil || req.Dashboard.Time.RelativeTime == nil {
		t.Fatal("relative_time not set")
	}
	if *req.Dashboard.Time.RelativeTime != 10080 {
		t.Errorf("relative_time wrong: %d", *req.Dashboard.Time.RelativeTime)
	}
}

// Helpers

func testAccDashboardRegion() string {
	if r := os.Getenv("LAST9_TEST_REGION"); r != "" {
		return r
	}
	return "ap-south-1"
}

func testAccCheckDashboardExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no dashboard ID is set")
		}
		return nil
	}
}

func testAccCheckDashboardDestroy(s *terraform.State) error {
	provider := testAccProvider()
	providerConfig := provider.Meta()
	if providerConfig == nil {
		return nil
	}
	apiClient := providerConfig.(*client.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_dashboard" {
			continue
		}
		region := rs.Primary.Attributes["region"]
		_, err := apiClient.GetDashboard(rs.Primary.ID, region)
		if err == nil {
			return fmt.Errorf("dashboard %s still exists", rs.Primary.ID)
		}
		if !strings.Contains(err.Error(), "404") {
			return fmt.Errorf("unexpected error checking dashboard %s: %w", rs.Primary.ID, err)
		}
	}
	return nil
}

func testAccDashboardImportID(resourceName, region string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}
		return fmt.Sprintf("%s:%s", region, rs.Primary.ID), nil
	}
}

// Configs

func testAccDashboardConfig_basic(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_dashboard" "test" {
  region = %q
  name   = "TF Test Basic Dashboard"

  panel {
    name = "Container Memory"
    unit = "bytes-iec"

    layout {
      x = 0
      y = 0
      w = 6
      h = 6
    }

    visualization {
      type       = "stat"
      full_width = false
    }

    query {
      name             = "A"
      expr             = "avg(container_memory_usage_bytes)"
      telemetry        = "metrics"
      query_type       = "promql"
      legend_type      = "auto"
      legend_placement = "right"
    }
  }
}
`, region)
}

func testAccDashboardConfig_basicUpdated(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_dashboard" "test" {
  region = %q
  name   = "TF Test Basic Dashboard Updated"

  panel {
    name = "Container Memory Updated"
    unit = "bytes-iec"

    layout {
      x = 0
      y = 0
      w = 6
      h = 6
    }

    visualization {
      type = "stat"
    }

    query {
      name       = "A"
      expr       = "avg(container_memory_usage_bytes)"
      telemetry  = "metrics"
      query_type = "promql"
    }
  }
}
`, region)
}

func testAccDashboardConfig_multiPanelWithSection(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_dashboard" "test" {
  region = %q
  name   = "TF Test Multi Panel"

  panel {
    name = "Spend at a Glance"
    visualization {
      type       = "section"
      full_width = true
    }
  }

  panel {
    name = "Total Spend"
    unit = "USD"

    layout {
      x = 0
      y = 0
      w = 3
      h = 6
    }

    visualization {
      type = "stat"
    }

    query {
      name       = "A"
      expr       = "sum(aws_cost_unblended_USD)"
      telemetry  = "metrics"
      query_type = "promql"
    }
  }

  panel {
    name = "Cost by Service"
    unit = "USD"

    layout {
      x = 0
      y = 1
      w = 12
      h = 8
    }

    visualization {
      type       = "bar"
      full_width = true

      bar_config {
        orientation = "vertical"
        stacked     = true
      }
    }

    query {
      name             = "A"
      expr             = "sum by (aws_service) (aws_cost_unblended_USD)"
      telemetry        = "metrics"
      query_type       = "promql"
      legend_type      = "custom"
      legend_value     = "{{aws_service}}"
      legend_placement = "bottom"
    }
  }
}
`, region)
}

func testAccDashboardConfig_withVariables(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_dashboard" "test" {
  region        = %q
  name          = "TF Test With Variables"
  relative_time = 10080

  variable {
    display_name = "Account"
    target       = "account"
    type         = "label"
    source       = "aws_account_id"
    matches      = ["aws_cost_unblended_USD{cost_date!=\"\"}"]
    multiple     = true
    current_values = [".*"]
  }

  variable {
    display_name = "Region"
    target       = "region"
    type         = "label"
    source       = "aws_region"
    matches      = ["aws_cost_unblended_USD{cost_date!=\"\", aws_account_id=~\"$account\"}"]
    multiple     = true
    current_values = [".*"]
  }

  panel {
    name = "Total Spend"
    unit = "USD"

    layout {
      x = 0
      y = 0
      w = 6
      h = 6
    }

    visualization {
      type = "stat"
    }

    query {
      name       = "A"
      expr       = "sum(aws_cost_unblended_USD{aws_account_id=~\"$account\", aws_region=~\"$region\"})"
      telemetry  = "metrics"
      query_type = "promql"
    }
  }
}
`, region)
}

func testAccDashboardConfig_sectionWithQuery(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_dashboard" "test" {
  region = %q
  name   = "TF Test Bad Section"

  panel {
    name = "Bad Section"

    visualization {
      type = "section"
    }

    query {
      name = "A"
      expr = "1"
    }
  }
}
`, region)
}

func testAccDashboardConfig_nonSectionMissingLayout(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_dashboard" "test" {
  region = %q
  name   = "TF Test Missing Layout"

  panel {
    name = "stat no layout"

    visualization {
      type = "stat"
    }

    query {
      name = "A"
      expr = "1"
    }
  }
}
`, region)
}

func testAccDashboardConfig_labelVariableMissingSource(region string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_dashboard" "test" {
  region = %q
  name   = "TF Test Bad Var"

  variable {
    display_name = "X"
    target       = "x"
    type         = "label"
  }

  panel {
    name = "p"

    layout {
      x = 0
      y = 0
      w = 6
      h = 6
    }

    visualization {
      type = "stat"
    }

    query {
      name = "A"
      expr = "1"
    }
  }
}
`, region)
}
