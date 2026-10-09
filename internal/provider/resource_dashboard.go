package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func validateOptionalJSONString(v interface{}, p cty.Path) diag.Diagnostics {
	s, ok := v.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return nil
	}
	var blob interface{}
	if err := json.Unmarshal([]byte(s), &blob); err != nil {
		return diag.Diagnostics{{
			Severity:      diag.Error,
			Summary:       "invalid JSON",
			Detail:        fmt.Sprintf("expected valid JSON, got error: %v", err),
			AttributePath: p,
		}}
	}
	return nil
}

func jsonStringsEqual(a, b string) bool {
	if a == b {
		return true
	}
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return a == b
	}
	var av, bv interface{}
	if err := json.Unmarshal([]byte(a), &av); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &bv); err != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}

var (
	supportedVisualizationTypes  = []string{"timeseries", "stat", "bar", "table", "section", "markdown", "doughnut", "logs"}
	supportedTelemetries         = []string{"metrics", "logs", "traces"}
	supportedQueryTypes          = []string{"promql", "log_ql", "log_json", "log_raw", "trace_ql", "trace_json"}
	supportedLegendTypes         = []string{"auto", "custom"}
	supportedLegendPlacements    = []string{"bottom", "left", "right"}
	supportedBarOrientations     = []string{"vertical", "horizontal"}
	supportedTimeseriesDisplays  = []string{"line", "area", ""}
	supportedVariableTypes       = []string{"label", "static"}
	supportedPanelUnits          = []string{"", "percent", "seconds", "milliseconds", "nanoseconds", "bytes-iec", "bytes-si", "bytes/sec-iec", "bytes/sec-si"}
	supportedTelemetryQueryTypes = map[string]map[string]bool{
		"metrics": {"promql": true},
		"logs":    {"log_ql": true, "log_json": true, "log_raw": true},
		"traces":  {"trace_ql": true, "trace_json": true},
	}
)

func resourceDashboard() *schema.Resource {
	r := &schema.Resource{
		CreateContext: resourceDashboardCreate,
		ReadContext:   resourceDashboardRead,
		UpdateContext: resourceDashboardUpdate,
		DeleteContext: resourceDashboardDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceDashboardImportState,
		},
		Schema: map[string]*schema.Schema{
			"links_json": dashboardJSONSchema("array", "Dashboard navigation links."),
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Region used to look up active integrations for query rendering. Not stored with dashboard; safe to change without recreate.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the dashboard",
			},
			"relative_time": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Relative time window in minutes (e.g., 60 = last 1 hour). Mutually exclusive with absolute_from/absolute_to.",
			},
			"absolute_from": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Absolute time range start (Unix millis). Must be set with absolute_to. Mutually exclusive with relative_time.",
			},
			"absolute_to": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Absolute time range end (Unix millis). Must be set with absolute_from. Mutually exclusive with relative_time.",
			},
			"metadata": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"category": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "custom",
						},
						"type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "metrics",
						},
						"tags": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"variable": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"display_name": {Type: schema.TypeString, Required: true},
						"target":       {Type: schema.TypeString, Required: true},
						"type": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice(supportedVariableTypes, false),
						},
						"source": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Label name to fetch values from (required when type=label)",
						},
						"matches": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"multiple":    {Type: schema.TypeBool, Optional: true, Default: false},
						"internal":    {Type: schema.TypeBool, Optional: true, Default: false},
						"regex":       {Type: schema.TypeString, Optional: true},
						"include_all": {Type: schema.TypeBool, Optional: true, Default: false},
						"all_value":   {Type: schema.TypeString, Optional: true},
						"current_values": {
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "Selected variable values. Initial values honored on create. Field is Computed: server-side selections (e.g., made via UI dropdown) are read back into state on every refresh, so plan won't show drift on UI changes — but if you keep a value in HCL, the next apply will revert UI selections to your HCL value. Omit from HCL to let the UI own selection state.",
						},
						"values": {
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "Static values (used when type=static). Computed so empty server-side lists for label variables do not create plan drift when omitted from HCL.",
						},
					},
				},
			},
			"panel": {
				Type:       schema.TypeSet,
				ConfigMode: schema.SchemaConfigModeBlock,
				Required:   true,
				MinItems:   1,
				Set:        dashboardPanelKeyHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Stable Terraform identity for this panel. Keep unchanged when editing or moving the panel.",
						},
						"position": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntAtLeast(0),
							Description:  "Zero-based order in the dashboard panel array, including section panels.",
						},
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Server-assigned panel UUID (round-tripped to prevent churn on update)",
						},
						"name":                 {Type: schema.TypeString, Required: true},
						"collapsed":            {Type: schema.TypeBool, Optional: true, Default: false},
						"field_overrides_json": dashboardJSONSchema("array", "Ordered native field overrides, including regex/type matching, hidden fields, thresholds and field links."),
						"transformations_json": dashboardJSONSchema("array", "Ordered native panel transformations."),
						"links_json":           dashboardJSONSchema("array", "Panel navigation links."),
						"data_links_json":      dashboardJSONSchema("array", "Panel data links."),
						"datasource_id":        {Type: schema.TypeString, Optional: true, Computed: true},
						"telemetry": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validation.StringInSlice(supportedTelemetries, false),
						},
						"unit": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringInSlice(supportedPanelUnits, false),
							Description:  `Unit displayed on the panel axis. Recognised values: "percent" (0–100 values, shows %), "seconds", "milliseconds", "nanoseconds", "bytes-iec", "bytes-si", "bytes/sec-iec", "bytes/sec-si". Use "" for dimensionless counts (lines, commits, etc.). Grafana-style IDs (ms, short, binBps, percentunit, ops, etc.) are NOT recognised and will render as literal text.`,
						},
						"version": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     1,
							Description: "Panel schema version. Defaults to 1. The API only persists query.telemetry and query.query_type when version >= 1; if you set this to 0, those fields will be silently dropped server-side. Leave as default unless you have a specific reason.",
						},
						"layout": {
							Type:     schema.TypeSet,
							Optional: true,
							MaxItems: 1,
							Set:      dashboardConstantSetHash("layout"),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"x": {Type: schema.TypeInt, Required: true},
									"y": {Type: schema.TypeInt, Required: true},
									"w": {Type: schema.TypeInt, Required: true},
									"h": {Type: schema.TypeInt, Required: true},
									"extra_json": {
										Type:        schema.TypeString,
										Optional:    true,
										Computed:    true,
										Description: "Reserved for layout fields the API may add (e.g. minH, static, i). Round-tripped verbatim. Use jsonencode() if you need to set keys beyond x/y/w/h.",
										DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
											return jsonStringsEqual(old, new)
										},
										ValidateDiagFunc: validateOptionalJSONString,
									},
								},
							},
						},
						"visualization": {
							Type:       schema.TypeSet,
							ConfigMode: schema.SchemaConfigModeBlock,
							Required:   true,
							MaxItems:   1,
							Set: func(v interface{}) int {
								visualization, ok := v.(map[string]interface{})
								if !ok {
									return schema.HashString(fmt.Sprintf("invalid-visualization-%T", v))
								}
								return schema.HashString(fmt.Sprint(visualization["type"]))
							},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validation.StringInSlice(supportedVisualizationTypes, false),
									},
									"full_width":          {Type: schema.TypeBool, Optional: true, Default: false},
									"logs_config_json":    dashboardJSONSchema("object", "Raw logs display configuration. Use jsonencode({...})."),
									"value_mappings_json": dashboardJSONSchema("array", "Native value mappings."),
									"timeseries_config": {
										Type:     schema.TypeSet,
										Optional: true,
										MaxItems: 1,
										Set:      dashboardConstantSetHash("timeseries"),
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"display_type": {
													Type:         schema.TypeString,
													Optional:     true,
													ValidateFunc: validation.StringInSlice(supportedTimeseriesDisplays, false),
												},
											},
										},
									},
									"bar_config": {
										Type:     schema.TypeSet,
										Optional: true,
										MaxItems: 1,
										Set:      dashboardConstantSetHash("bar"),
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"orientation": {
													Type:         schema.TypeString,
													Optional:     true,
													ValidateFunc: validation.StringInSlice(supportedBarOrientations, false),
												},
												"stacked": {Type: schema.TypeBool, Optional: true, Default: false},
											},
										},
									},
									"stat_config": {
										Type:     schema.TypeSet,
										Optional: true,
										MaxItems: 1,
										Set:      dashboardConstantSetHash("stat"),
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"threshold": {
													Type:     schema.TypeSet,
													Optional: true,
													Set:      dashboardThresholdKeyHash,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"position": {Type: schema.TypeInt, Required: true, ValidateFunc: validation.IntAtLeast(0)},
															"value":    {Type: schema.TypeFloat, Required: true},
															"color":    {Type: schema.TypeString, Required: true},
														},
													},
												},
											},
										},
									},
									"markdown_config": {
										Type:     schema.TypeSet,
										Optional: true,
										MaxItems: 1,
										Set:      dashboardConstantSetHash("markdown"),
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"content": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Markdown content rendered by markdown panels.",
												},
											},
										},
									},
									"table_config_json": {
										Type:        schema.TypeString,
										Optional:    true,
										Computed:    true,
										Description: "Raw table_config JSON. Server treats this as opaque blob (columnConfig, density, thresholds, etc). Use jsonencode({...}). Invalid JSON fails at plan time.",
										DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
											return jsonStringsEqual(old, new)
										},
										ValidateDiagFunc: validateOptionalJSONString,
									},
								},
							},
						},
						"query": {
							Type:     schema.TypeSet,
							Optional: true,
							Set:      dashboardQueryKeyHash,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"position": {Type: schema.TypeInt, Required: true, ValidateFunc: validation.IntAtLeast(0)},
									"name":     {Type: schema.TypeString, Required: true},
									"expr":     {Type: schema.TypeString, Required: true},
									"type":     {Type: schema.TypeString, Optional: true, Default: "range"},
									"unit": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validation.StringInSlice(supportedPanelUnits, false),
										Description:  `Unit for this query's series. Same recognised values as panel.unit. Rarely needed — set panel.unit instead unless individual queries in the same panel need different units.`,
									},
									"telemetry": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validation.StringInSlice(supportedTelemetries, false),
									},
									"query_type": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validation.StringInSlice(supportedQueryTypes, false),
									},
									"legend_type": {
										Type:         schema.TypeString,
										Optional:     true,
										Default:      "auto",
										ValidateFunc: validation.StringInSlice(supportedLegendTypes, false),
									},
									"legend_value": {Type: schema.TypeString, Optional: true},
									"legend_placement": {
										Type:         schema.TypeString,
										Optional:     true,
										Default:      "bottom",
										ValidateFunc: validation.StringInSlice(supportedLegendPlacements, false),
									},
									"legend_sort_field": {
										Type:        schema.TypeString,
										Optional:    true,
										Computed:    true,
										Description: "Field to sort legend entries by. Pairs with legend_sort_direction.",
									},
									"legend_sort_direction": {
										Type:         schema.TypeString,
										Optional:     true,
										Computed:     true,
										Description:  "Sort direction (asc|desc) for legend entries. Pairs with legend_sort_field.",
										ValidateFunc: validation.StringInSlice([]string{"asc", "desc", ""}, false),
									},
									"matrix_json": {
										Type:        schema.TypeString,
										Optional:    true,
										Computed:    true,
										Description: "Raw matrix JSON for query result transformation. Opaque blob. Invalid JSON fails at plan time.",
										DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
											return jsonStringsEqual(old, new)
										},
										ValidateDiagFunc: validateOptionalJSONString,
									},
								},
							},
						},
					},
				},
			},
			// Computed
			"created_at": {Type: schema.TypeInt, Computed: true},
			"updated_at": {Type: schema.TypeInt, Computed: true},
			"created_by": {Type: schema.TypeString, Computed: true},
			"readonly":   {Type: schema.TypeBool, Computed: true},
		},
		CustomizeDiff: validateDashboard,
	}
	r.SchemaVersion = 1
	legacy := dashboardLegacyPanelSchema(r.Schema)
	r.StateUpgraders = []schema.StateUpgrader{{
		Version: 0,
		Type:    (&schema.Resource{Schema: legacy}).CoreConfigSchema().ImpliedType(),
		Upgrade: upgradeDashboardPanelStateV0,
	}}
	return r
}

type dashboardGetter interface {
	Get(key string) interface{}
}

func validateDashboard(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
	return validateDashboardData(d)
}

func validateDashboardData(d dashboardGetter) error {
	panels := dashboardPanelValues(d.Get("panel"))
	keys := make(map[string]struct{}, len(panels))
	positions := make(map[int]struct{}, len(panels))
	for i, p := range panels {
		pm, ok := p.(map[string]interface{})
		if !ok {
			return fmt.Errorf("panel[%d]: expected object, got %T", i, p)
		}
		key, knownKey := dashboardPanelKey(pm)
		if !knownKey || strings.TrimSpace(key) == "" {
			return fmt.Errorf("panel[%d]: key must be a non-empty stable identity", i)
		}
		if _, exists := keys[key]; exists {
			return fmt.Errorf("panel key %q must be unique", key)
		}
		keys[key] = struct{}{}
		position, knownPosition := dashboardPanelPosition(pm)
		if !knownPosition {
			return fmt.Errorf("panel[%d] %q: position must be a known integer", i, key)
		}
		if position < 0 {
			return fmt.Errorf("panel[%d] %q: position must be zero or greater", i, key)
		}
		if _, exists := positions[position]; exists {
			return fmt.Errorf("panel position %d must be unique", position)
		}
		positions[position] = struct{}{}
		viz := dashboardPanelValues(pm["visualization"])
		if len(viz) == 0 {
			return fmt.Errorf("panel[%d] %q: visualization is required", i, pm["name"])
		}
		vm, ok := viz[0].(map[string]interface{})
		if !ok {
			return fmt.Errorf("panel[%d] %q: visualization must be an object (got %T: %#v; panel=%#v)", i, key, viz[0], viz[0], pm)
		}
		vizType, ok := vm["type"].(string)
		if !ok || vizType == "" {
			return fmt.Errorf("panel[%d] %q: visualization type must be known", i, key)
		}
		queries := dashboardPanelValues(pm["query"])
		layout := dashboardPanelValues(pm["layout"])
		queryNames := make(map[string]struct{}, len(queries))
		for queryIndex, rawQuery := range queries {
			query, ok := rawQuery.(map[string]interface{})
			if !ok {
				return fmt.Errorf("panel[%d] %q query[%d]: expected object", i, key, queryIndex)
			}
			name, ok := query["name"].(string)
			if !ok || strings.TrimSpace(name) == "" {
				return fmt.Errorf("panel[%d] %q query[%d]: name must be a non-empty stable identity", i, key, queryIndex)
			}
			if _, exists := queryNames[name]; exists {
				return fmt.Errorf("panel[%d] %q query name %q must be unique", i, key, name)
			}
			queryNames[name] = struct{}{}
		}
		if err := validateDashboardOrderedBlocks(queries, fmt.Sprintf("panel[%d] %q query", i, key)); err != nil {
			return err
		}
		if visualizations := dashboardPanelValues(pm["visualization"]); len(visualizations) == 1 {
			if visualization, ok := visualizations[0].(map[string]interface{}); ok {
				if statConfigs := dashboardPanelValues(visualization["stat_config"]); len(statConfigs) > 0 {
					if stat, ok := statConfigs[0].(map[string]interface{}); ok {
						if err := validateDashboardOrderedBlocks(dashboardPanelValues(stat["threshold"]), fmt.Sprintf("panel[%d] %q threshold", i, key)); err != nil {
							return err
						}
					}
				}
			}
		}
		if err := validateDashboardLogs(pm, vm); err != nil {
			return fmt.Errorf("panel[%d] %q: %w", i, pm["name"], err)
		}

		if vizType == "section" {
			if len(queries) > 0 {
				return fmt.Errorf("panel[%d] %q: section panels cannot have query blocks", i, pm["name"])
			}
			if len(layout) > 0 {
				return fmt.Errorf("panel[%d] %q: section panels cannot have layout block", i, pm["name"])
			}
		} else if vizType == "markdown" {
			if markdownConfig := dashboardPanelValues(vm["markdown_config"]); len(markdownConfig) == 0 {
				return fmt.Errorf("panel[%d] %q: markdown panels require markdown_config", i, pm["name"])
			}
			if len(queries) > 0 {
				return fmt.Errorf("panel[%d] %q: markdown panels cannot have query blocks", i, pm["name"])
			}
			if len(layout) == 0 {
				return fmt.Errorf("panel[%d] %q: markdown panels require a layout block", i, pm["name"])
			}
		} else {
			if len(queries) == 0 {
				return fmt.Errorf("panel[%d] %q: %s panels require at least one query block", i, pm["name"], vizType)
			}
			if len(layout) == 0 {
				return fmt.Errorf("panel[%d] %q: %s panels require a layout block", i, pm["name"], vizType)
			}
		}

		version := pm["version"].(int)
		if version > 0 {
			for qi, q := range queries {
				qm := q.(map[string]interface{})
				telemetry := qm["telemetry"].(string)
				queryType := qm["query_type"].(string)
				if telemetry == "" {
					return fmt.Errorf("panel[%d].query[%d]: telemetry is required when panel version is set", i, qi)
				}
				if queryType == "" {
					return fmt.Errorf("panel[%d].query[%d]: query_type is required when panel version is set", i, qi)
				}
				if !supportedTelemetryQueryTypes[telemetry][queryType] {
					return fmt.Errorf("panel[%d].query[%d]: query_type %q is not valid for telemetry %q", i, qi, queryType, telemetry)
				}
			}
		}
	}
	for position := 0; position < len(panels); position++ {
		if _, exists := positions[position]; !exists {
			return fmt.Errorf("panel positions must be contiguous from 0; missing position %d", position)
		}
	}

	variables := d.Get("variable").([]interface{})
	for i, v := range variables {
		vm := v.(map[string]interface{})
		vt := vm["type"].(string)
		source := vm["source"].(string)
		values := vm["values"].([]interface{})
		if vt == "label" && source == "" {
			return fmt.Errorf("variable[%d] %q: source is required when type=label", i, vm["display_name"])
		}
		if vt == "static" && len(values) == 0 {
			return fmt.Errorf("variable[%d] %q: values is required when type=static", i, vm["display_name"])
		}
	}

	rel := d.Get("relative_time").(int)
	from := d.Get("absolute_from").(int)
	to := d.Get("absolute_to").(int)
	if rel > 0 && (from > 0 || to > 0) {
		return fmt.Errorf("relative_time cannot be combined with absolute_from/absolute_to")
	}
	if (from > 0) != (to > 0) {
		return fmt.Errorf("absolute_from and absolute_to must be set together")
	}

	return nil
}

func resourceDashboardCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	req := buildDashboardRequest(d)

	result, err := apiClient.CreateDashboard(req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create dashboard: %w", err))
	}
	if result.Dashboard == nil {
		return diag.FromErr(fmt.Errorf("create dashboard: empty response"))
	}
	d.SetId(result.Dashboard.ID)
	return resourceDashboardRead(ctx, d, m)
}

func resourceDashboardRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	region := d.Get("region").(string)

	result, err := apiClient.GetDashboard(d.Id(), region)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to read dashboard: %w", err))
	}
	if result.Dashboard == nil {
		d.SetId("")
		return nil
	}

	dash := result.Dashboard
	d.Set("name", dash.Name)
	d.Set("created_at", dash.CreatedAt)
	d.Set("updated_at", dash.UpdatedAt)
	d.Set("created_by", dash.CreatedBy)
	d.Set("readonly", dash.Readonly)
	d.Set("links_json", string(dash.Links))

	if dash.Time != nil {
		if dash.Time.RelativeTime != nil {
			d.Set("relative_time", *dash.Time.RelativeTime)
		}
		if dash.Time.From != nil {
			d.Set("absolute_from", *dash.Time.From)
		}
		if dash.Time.To != nil {
			d.Set("absolute_to", *dash.Time.To)
		}
	}

	d.Set("panel", flattenPanelsWithState(dash.Panels, dashboardPanelValues(d.Get("panel"))))
	d.Set("variable", flattenVariables(dash.Variables))

	if result.Metadata != nil {
		d.Set("metadata", flattenMetadata(result.Metadata))
	}

	return nil
}

func resourceDashboardUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	req := buildDashboardRequest(d)
	if req.Dashboard != nil {
		req.Dashboard.ID = d.Id()
	}

	if _, err := apiClient.UpdateDashboard(d.Id(), req); err != nil {
		return diag.FromErr(fmt.Errorf("failed to update dashboard: %w", err))
	}
	return resourceDashboardRead(ctx, d, m)
}

func resourceDashboardDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	if err := apiClient.DeleteDashboard(d.Id()); err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete dashboard: %w", err))
	}
	d.SetId("")
	return nil
}

func resourceDashboardImportState(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	parts := strings.SplitN(d.Id(), ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid import ID, expected 'region:dashboard_id', got: %s", d.Id())
	}
	d.Set("region", parts[0])
	d.SetId(parts[1])
	return []*schema.ResourceData{d}, nil
}

func buildDashboardRequest(d *schema.ResourceData) *client.DashboardRequest {
	dash := &client.Dashboard{
		Links:     dashboardJSONValue(d.Get("links_json")),
		Name:      d.Get("name").(string),
		Panels:    expandPanels(dashboardPanelValues(d.Get("panel"))),
		Variables: expandVariables(d.Get("variable").([]interface{})),
	}
	if v, ok := d.GetOk("relative_time"); ok && v.(int) > 0 {
		rt := int64(v.(int))
		dash.Time = &client.DashboardTime{RelativeTime: &rt}
	} else if from, _ := d.GetOk("absolute_from"); from != nil && from.(int) > 0 {
		f := int64(from.(int))
		t := int64(d.Get("absolute_to").(int))
		dash.Time = &client.DashboardTime{From: &f, To: &t}
	}

	req := &client.DashboardRequest{Dashboard: dash}
	if md := d.Get("metadata").([]interface{}); len(md) > 0 {
		req.Metadata = expandMetadata(md)
	}
	return req
}

func expandPanels(input []interface{}) []*client.DashboardPanel {
	ordered := append([]interface{}(nil), input...)
	sort.SliceStable(ordered, func(i, j int) bool {
		leftPanel, leftOK := ordered[i].(map[string]interface{})
		rightPanel, rightOK := ordered[j].(map[string]interface{})
		if leftOK != rightOK {
			return leftOK
		}
		if !leftOK {
			return false
		}
		left, leftKnown := dashboardPanelPosition(leftPanel)
		right, rightKnown := dashboardPanelPosition(rightPanel)
		if leftKnown != rightKnown {
			return leftKnown
		}
		if !leftKnown {
			return false
		}
		return left < right
	})
	panels := make([]*client.DashboardPanel, 0, len(ordered))
	for _, p := range ordered {
		pm := p.(map[string]interface{})
		panel := &client.DashboardPanel{
			Collapsed:       pm["collapsed"].(bool),
			FieldOverrides:  dashboardJSONValue(pm["field_overrides_json"]),
			Transformations: dashboardJSONValue(pm["transformations_json"]),
			Links:           dashboardJSONValue(pm["links_json"]),
			DataLinks:       dashboardJSONValue(pm["data_links_json"]),
			ID:              pm["id"].(string),
			Name:            pm["name"].(string),
			DatasourceID:    pm["datasource_id"].(string),
			Telemetry:       pm["telemetry"].(string),
			Unit:            pm["unit"].(string),
			Version:         pm["version"].(int),
		}

		if vizList := dashboardPanelValues(pm["visualization"]); len(vizList) > 0 {
			panel.Visualization = expandVisualization(vizList[0].(map[string]interface{}))
		}

		if layoutList := dashboardPanelValues(pm["layout"]); len(layoutList) > 0 {
			lm := layoutList[0].(map[string]interface{})
			layout := map[string]any{
				"x": lm["x"].(int),
				"y": lm["y"].(int),
				"w": lm["w"].(int),
				"h": lm["h"].(int),
			}
			if extra, _ := lm["extra_json"].(string); strings.TrimSpace(extra) != "" {
				var blob map[string]any
				if err := json.Unmarshal([]byte(extra), &blob); err == nil {
					for k, v := range blob {
						if _, reserved := layout[k]; !reserved {
							layout[k] = v
						}
					}
				}
			}
			panel.Layout = layout
		}

		panel.PopulatedQueries = expandQueries(dashboardPanelValues(pm["query"]))
		panels = append(panels, panel)
	}
	return panels
}

func dashboardPanelValues(value interface{}) []interface{} {
	switch panels := value.(type) {
	case *schema.Set:
		return panels.List()
	case []interface{}:
		return panels
	default:
		return nil
	}
}

func dashboardConstantSetHash(key string) schema.SchemaSetFunc {
	return func(interface{}) int { return schema.HashString(key) }
}

func dashboardQueryKeyHash(value interface{}) int {
	query, ok := value.(map[string]interface{})
	if !ok {
		return schema.HashString(fmt.Sprintf("invalid-query-%T", value))
	}
	return schema.HashString(fmt.Sprint(query["name"]))
}

func dashboardThresholdKeyHash(value interface{}) int {
	threshold, ok := value.(map[string]interface{})
	if !ok {
		return schema.HashString(fmt.Sprintf("invalid-threshold-%T", value))
	}
	position, ok := threshold["position"].(int)
	if !ok {
		return schema.HashString("unknown-threshold-position")
	}
	return schema.HashInt(position)
}

func validateDashboardOrderedBlocks(blocks []interface{}, label string) error {
	positions := make(map[int]struct{}, len(blocks))
	for i, raw := range blocks {
		block, ok := raw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("%s[%d]: expected object", label, i)
		}
		position, ok := block["position"].(int)
		if !ok || position < 0 {
			return fmt.Errorf("%s[%d]: position must be a known non-negative integer", label, i)
		}
		if _, exists := positions[position]; exists {
			return fmt.Errorf("%s position %d must be unique", label, position)
		}
		positions[position] = struct{}{}
	}
	for position := 0; position < len(blocks); position++ {
		if _, exists := positions[position]; !exists {
			return fmt.Errorf("%s positions must be contiguous from 0; missing position %d", label, position)
		}
	}
	return nil
}

func dashboardPanelKey(panel map[string]interface{}) (string, bool) {
	key, ok := panel["key"].(string)
	return key, ok
}

func dashboardPanelPosition(panel map[string]interface{}) (int, bool) {
	position, ok := panel["position"].(int)
	return position, ok
}

func expandVisualization(m map[string]interface{}) *client.DashboardPanelVisualization {
	viz := &client.DashboardPanelVisualization{
		ValueMappings: dashboardJSONValue(m["value_mappings_json"]),
		LogsConfig:    dashboardJSONValue(m["logs_config_json"]),
		Type:          m["type"].(string),
		FullWidth:     m["full_width"].(bool),
	}
	if l := dashboardPanelValues(m["timeseries_config"]); len(l) > 0 {
		c := l[0].(map[string]interface{})
		viz.TimeseriesConfig = &client.DashboardTimeseriesConfig{DisplayType: c["display_type"].(string)}
	}
	if l := dashboardPanelValues(m["bar_config"]); len(l) > 0 {
		c := l[0].(map[string]interface{})
		stacked := c["stacked"].(bool)
		viz.BarConfig = &client.DashboardBarConfig{
			Orientation: c["orientation"].(string),
			Stacked:     &stacked,
		}
	}
	if l := dashboardPanelValues(m["stat_config"]); len(l) > 0 {
		c := l[0].(map[string]interface{})
		viz.StatConfig = &client.DashboardStatConfig{
			Thresholds: expandStatThresholds(dashboardPanelValues(c["threshold"])),
		}
	}
	if l := dashboardPanelValues(m["markdown_config"]); len(l) > 0 {
		c := l[0].(map[string]interface{})
		viz.MarkdownConfig = &client.DashboardMarkdownConfig{Content: c["content"].(string)}
	}
	if s := strings.TrimSpace(m["table_config_json"].(string)); s != "" {
		var blob interface{}
		if err := json.Unmarshal([]byte(s), &blob); err == nil {
			viz.TableConfig = blob
		}
	}
	return viz
}

func expandStatThresholds(input []interface{}) []client.DashboardStatThreshold {
	input = orderByDashboardPosition(input)
	out := make([]client.DashboardStatThreshold, 0, len(input))
	for _, t := range input {
		tm := t.(map[string]interface{})
		out = append(out, client.DashboardStatThreshold{
			Value: tm["value"].(float64),
			Color: tm["color"].(string),
		})
	}
	return out
}

func expandQueries(input []interface{}) []*client.DashboardPanelQueryDetails {
	input = orderByDashboardPosition(input)
	out := make([]*client.DashboardPanelQueryDetails, 0, len(input))
	for _, q := range input {
		qm := q.(map[string]interface{})
		qd := &client.DashboardPanelQueryDetails{
			Name:            qm["name"].(string),
			Expr:            qm["expr"].(string),
			Type:            qm["type"].(string),
			Unit:            qm["unit"].(string),
			Telemetry:       qm["telemetry"].(string),
			QueryType:       qm["query_type"].(string),
			LegendPlacement: qm["legend_placement"].(string),
			Legend: client.DashboardPanelLegend{
				Type:  qm["legend_type"].(string),
				Value: qm["legend_value"].(string),
			},
		}
		sortField, _ := qm["legend_sort_field"].(string)
		sortDir, _ := qm["legend_sort_direction"].(string)
		if sortField != "" || sortDir != "" {
			qd.LegendSort = &client.DashboardPanelLegendSort{Field: sortField, Direction: sortDir}
		}
		if mj, _ := qm["matrix_json"].(string); strings.TrimSpace(mj) != "" {
			var blob interface{}
			if err := json.Unmarshal([]byte(mj), &blob); err == nil {
				qd.Matrix = blob
			}
		}
		out = append(out, qd)
	}
	return out
}

func orderByDashboardPosition(input []interface{}) []interface{} {
	ordered := append([]interface{}(nil), input...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left, lok := ordered[i].(map[string]interface{})
		right, rok := ordered[j].(map[string]interface{})
		if lok != rok {
			return lok
		}
		if !lok {
			return false
		}
		lp, lok := dashboardPanelPosition(left)
		rp, rok := dashboardPanelPosition(right)
		if lok != rok {
			return lok
		}
		if !lok {
			return false
		}
		return lp < rp
	})
	return ordered
}

func expandVariables(input []interface{}) []*client.DashboardVariable {
	out := make([]*client.DashboardVariable, 0, len(input))
	for _, v := range input {
		vm := v.(map[string]interface{})
		dv := &client.DashboardVariable{
			Regex:         vm["regex"].(string),
			IncludeAll:    vm["include_all"].(bool),
			AllValue:      vm["all_value"].(string),
			DisplayName:   vm["display_name"].(string),
			Target:        vm["target"].(string),
			Type:          vm["type"].(string),
			Source:        vm["source"].(string),
			Multiple:      vm["multiple"].(bool),
			Internal:      vm["internal"].(bool),
			Matches:       expandStringList(vm["matches"].([]interface{})),
			CurrentValues: toAnySlice(expandStringList(vm["current_values"].([]interface{}))),
		}
		if vals := vm["values"].([]interface{}); len(vals) > 0 {
			dv.Values = vals
		}
		out = append(out, dv)
	}
	return out
}

func expandMetadata(input []interface{}) *client.DashboardMetadata {
	if len(input) == 0 {
		return nil
	}
	m := input[0].(map[string]interface{})
	return &client.DashboardMetadata{
		Category: m["category"].(string),
		Type:     m["type"].(string),
		Tags:     expandStringList(m["tags"].([]interface{})),
	}
}

func coerceToStringSlice(in []interface{}) []string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		if x == nil {
			continue
		}
		switch v := x.(type) {
		case string:
			out = append(out, v)
		default:
			out = append(out, fmt.Sprint(v))
		}
	}
	return out
}

func toAnySlice(in []string) []any {
	out := make([]any, 0, len(in))
	for _, s := range in {
		out = append(out, s)
	}
	return out
}

func flattenPanelsWithState(panels []*client.DashboardPanel, current []interface{}) []interface{} {
	keysByID := make(map[string]string, len(current))
	keysByPosition := make(map[int]string, len(current))
	for _, raw := range current {
		panel, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		key, _ := dashboardPanelKey(panel)
		id, _ := panel["id"].(string)
		position, hasPosition := dashboardPanelPosition(panel)
		if id != "" && key != "" {
			keysByID[id] = key
		}
		if key != "" && hasPosition {
			keysByPosition[position] = key
		}
	}
	out := make([]interface{}, 0, len(panels))
	for position, p := range panels {
		if p == nil {
			continue
		}
		key := keysByID[p.ID]
		if key == "" {
			key = keysByPosition[position]
		}
		if key == "" {
			key = p.ID
		}
		pm := map[string]interface{}{
			"key":                  key,
			"position":             position,
			"collapsed":            p.Collapsed,
			"field_overrides_json": string(p.FieldOverrides),
			"transformations_json": string(p.Transformations),
			"links_json":           string(p.Links),
			"data_links_json":      string(p.DataLinks),
			"id":                   p.ID,
			"name":                 p.Name,
			"datasource_id":        p.DatasourceID,
			"telemetry":            p.Telemetry,
			"unit":                 p.Unit,
			"version":              p.Version,
			"visualization":        flattenVisualization(p.Visualization),
			"query":                flattenQueries(p.PopulatedQueries),
			"layout":               flattenLayout(p.Layout),
		}
		out = append(out, pm)
	}
	return out
}

func flattenLayout(layout map[string]any) []interface{} {
	if layout == nil {
		return []interface{}{}
	}
	out := map[string]interface{}{
		"x": toInt(layout["x"]),
		"y": toInt(layout["y"]),
		"w": toInt(layout["w"]),
		"h": toInt(layout["h"]),
	}
	extra := map[string]any{}
	for k, v := range layout {
		if k == "x" || k == "y" || k == "w" || k == "h" {
			continue
		}
		extra[k] = v
	}
	if len(extra) > 0 {
		if b, err := json.Marshal(extra); err == nil {
			out["extra_json"] = string(b)
		}
	}
	return []interface{}{out}
}

func toInt(v interface{}) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	default:
		return 0
	}
}

func flattenVisualization(viz *client.DashboardPanelVisualization) []interface{} {
	if viz == nil {
		return []interface{}{}
	}
	m := map[string]interface{}{
		"type":                viz.Type,
		"full_width":          viz.FullWidth,
		"logs_config_json":    string(viz.LogsConfig),
		"value_mappings_json": string(viz.ValueMappings),
	}
	if viz.TimeseriesConfig != nil {
		m["timeseries_config"] = []interface{}{
			map[string]interface{}{"display_type": viz.TimeseriesConfig.DisplayType},
		}
	}
	if viz.BarConfig != nil {
		stacked := false
		if viz.BarConfig.Stacked != nil {
			stacked = *viz.BarConfig.Stacked
		}
		m["bar_config"] = []interface{}{
			map[string]interface{}{
				"orientation": viz.BarConfig.Orientation,
				"stacked":     stacked,
			},
		}
	}
	if viz.StatConfig != nil {
		thresholds := make([]interface{}, 0, len(viz.StatConfig.Thresholds))
		for _, t := range viz.StatConfig.Thresholds {
			thresholds = append(thresholds, map[string]interface{}{
				"position": len(thresholds),
				"value":    t.Value,
				"color":    t.Color,
			})
		}
		m["stat_config"] = []interface{}{map[string]interface{}{"threshold": thresholds}}
	}
	if viz.MarkdownConfig != nil {
		m["markdown_config"] = []interface{}{
			map[string]interface{}{"content": viz.MarkdownConfig.Content},
		}
	}
	if viz.TableConfig != nil {
		if b, err := json.Marshal(viz.TableConfig); err == nil {
			m["table_config_json"] = string(b)
		}
	}
	return []interface{}{m}
}

func flattenQueries(queries []*client.DashboardPanelQueryDetails) []interface{} {
	out := make([]interface{}, 0, len(queries))
	for i, q := range queries {
		if q == nil {
			continue
		}
		qm := map[string]interface{}{
			"position":         i,
			"name":             q.Name,
			"expr":             q.Expr,
			"type":             q.Type,
			"unit":             q.Unit,
			"telemetry":        q.Telemetry,
			"query_type":       q.QueryType,
			"legend_type":      q.Legend.Type,
			"legend_value":     q.Legend.Value,
			"legend_placement": q.LegendPlacement,
		}
		if q.LegendSort != nil {
			qm["legend_sort_field"] = q.LegendSort.Field
			qm["legend_sort_direction"] = q.LegendSort.Direction
		}
		if q.Matrix != nil {
			if b, err := json.Marshal(q.Matrix); err == nil {
				qm["matrix_json"] = string(b)
			}
		}
		out = append(out, qm)
	}
	return out
}

func flattenVariables(variables []*client.DashboardVariable) []interface{} {
	out := make([]interface{}, 0, len(variables))
	for _, v := range variables {
		if v == nil {
			continue
		}
		values := coerceToStringSlice(v.Values)
		current := coerceToStringSlice(v.CurrentValues)
		out = append(out, map[string]interface{}{
			"display_name":   v.DisplayName,
			"regex":          v.Regex,
			"include_all":    v.IncludeAll,
			"all_value":      v.AllValue,
			"target":         v.Target,
			"type":           v.Type,
			"source":         v.Source,
			"matches":        v.Matches,
			"multiple":       v.Multiple,
			"internal":       v.Internal,
			"current_values": current,
			"values":         values,
		})
	}
	return out
}

func flattenMetadata(md *client.DashboardMetadata) []interface{} {
	if md == nil {
		return []interface{}{}
	}
	return []interface{}{
		map[string]interface{}{
			"category": md.Category,
			"type":     md.Type,
			"tags":     md.Tags,
		},
	}
}

func dashboardPanelKeyHash(v interface{}) int {
	panel, ok := v.(map[string]interface{})
	if !ok {
		return schema.HashString(fmt.Sprintf("invalid-panel-%T", v))
	}
	key, _ := dashboardPanelKey(panel)
	if key == "" {
		key, _ = panel["name"].(string)
	}
	return schema.HashString(key)
}

func dashboardLegacyPanelSchema(current map[string]*schema.Schema) map[string]*schema.Schema {
	legacy := make(map[string]*schema.Schema, len(current))
	for name, field := range current {
		copied := *field
		legacy[name] = &copied
	}
	panel := *legacy["panel"]
	panel.Type = schema.TypeList
	panel.Set = nil
	panelElem := *panel.Elem.(*schema.Resource)
	panelElem.Schema = make(map[string]*schema.Schema, len(panelElem.Schema)-2)
	for name, field := range panel.Elem.(*schema.Resource).Schema {
		if name == "key" || name == "position" {
			continue
		}
		panelElem.Schema[name] = dashboardLegacyNestedSchema(name, field)
	}
	panel.Elem = &panelElem
	legacy["panel"] = &panel
	return legacy
}

func dashboardLegacyNestedSchema(name string, field *schema.Schema) *schema.Schema {
	copied := *field
	switch name {
	case "layout", "visualization", "query", "timeseries_config", "bar_config", "stat_config", "markdown_config", "threshold":
		copied.Type = schema.TypeList
		copied.Set = nil
	}
	if resource, ok := field.Elem.(*schema.Resource); ok {
		legacyResource := *resource
		legacyResource.Schema = make(map[string]*schema.Schema, len(resource.Schema))
		for childName, child := range resource.Schema {
			if (name == "query" || name == "threshold") && childName == "position" {
				continue
			}
			legacyResource.Schema[childName] = dashboardLegacyNestedSchema(childName, child)
		}
		copied.Elem = &legacyResource
	}
	return &copied
}

func upgradeDashboardPanelStateV0(_ context.Context, state map[string]interface{}, _ interface{}) (map[string]interface{}, error) {
	panels, _ := state["panel"].([]interface{})
	seen := make(map[string]struct{}, len(panels))
	for position, raw := range panels {
		panel, ok := raw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot upgrade dashboard panel %d: expected object, got %T", position, raw)
		}
		key, _ := panel["id"].(string)
		if key == "" {
			key = fmt.Sprintf("legacy-panel-%d", position)
		}
		if _, duplicate := seen[key]; duplicate {
			return nil, fmt.Errorf("cannot safely upgrade dashboard panels with duplicate API id %q", key)
		}
		seen[key] = struct{}{}
		panel["key"] = key
		panel["position"] = position
		queries, _ := panel["query"].([]interface{})
		for queryPosition, rawQuery := range queries {
			query, ok := rawQuery.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("cannot upgrade dashboard panel %d query %d: expected object", position, queryPosition)
			}
			query["position"] = queryPosition
		}
		visualizations, _ := panel["visualization"].([]interface{})
		for _, rawVisualization := range visualizations {
			visualization, ok := rawVisualization.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("cannot upgrade dashboard panel %d visualization: expected object", position)
			}
			statConfigs, _ := visualization["stat_config"].([]interface{})
			for _, rawStat := range statConfigs {
				stat, ok := rawStat.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("cannot upgrade dashboard panel %d stat_config: expected object", position)
				}
				thresholds, _ := stat["threshold"].([]interface{})
				for thresholdPosition, rawThreshold := range thresholds {
					threshold, ok := rawThreshold.(map[string]interface{})
					if !ok {
						return nil, fmt.Errorf("cannot upgrade dashboard panel %d threshold %d: expected object", position, thresholdPosition)
					}
					threshold["position"] = thresholdPosition
				}
			}
		}
	}
	return state, nil
}
