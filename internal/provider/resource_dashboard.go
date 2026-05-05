package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
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
	supportedVisualizationTypes = []string{"timeseries", "stat", "bar", "table", "section"}
	supportedTelemetries        = []string{"metrics", "logs", "traces"}
	supportedQueryTypes         = []string{"promql", "log_ql", "log_json", "trace_ql", "trace_json"}
	supportedLegendTypes        = []string{"auto", "custom"}
	supportedLegendPlacements   = []string{"bottom", "left", "right"}
	supportedBarOrientations    = []string{"vertical", "horizontal"}
	supportedTimeseriesDisplays = []string{"line", "area", ""}
	supportedVariableTypes      = []string{"label", "static"}
	supportedTelemetryQueryTypes = map[string]map[string]bool{
		"metrics": {"promql": true},
		"logs":    {"log_ql": true, "log_json": true},
		"traces":  {"trace_ql": true, "trace_json": true},
	}
)

func resourceDashboard() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDashboardCreate,
		ReadContext:   resourceDashboardRead,
		UpdateContext: resourceDashboardUpdate,
		DeleteContext: resourceDashboardDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceDashboardImportState,
		},
		Schema: map[string]*schema.Schema{
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
						"multiple": {Type: schema.TypeBool, Optional: true, Default: false},
						"internal": {Type: schema.TypeBool, Optional: true, Default: false},
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
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "Static values (used when type=static)",
						},
					},
				},
			},
			"panel": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Server-assigned panel UUID (round-tripped to prevent churn on update)",
						},
						"name":          {Type: schema.TypeString, Required: true},
						"datasource_id": {Type: schema.TypeString, Optional: true, Computed: true},
						"telemetry": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validation.StringInSlice(supportedTelemetries, false),
						},
						"unit":    {Type: schema.TypeString, Optional: true, Computed: true},
						"version": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     1,
							Description: "Panel schema version. Defaults to 1. The API only persists query.telemetry and query.query_type when version >= 1; if you set this to 0, those fields will be silently dropped server-side. Leave as default unless you have a specific reason.",
						},
						"layout": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
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
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validation.StringInSlice(supportedVisualizationTypes, false),
									},
									"full_width": {Type: schema.TypeBool, Optional: true, Default: false},
									"timeseries_config": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
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
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
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
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"threshold": {
													Type:     schema.TypeList,
													Optional: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"value": {Type: schema.TypeFloat, Required: true},
															"color": {Type: schema.TypeString, Required: true},
														},
													},
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
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {Type: schema.TypeString, Required: true},
									"expr": {Type: schema.TypeString, Required: true},
									"type": {Type: schema.TypeString, Optional: true, Default: "range"},
									"unit": {Type: schema.TypeString, Optional: true},
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
}

type dashboardGetter interface {
	Get(key string) interface{}
}

func validateDashboard(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
	return validateDashboardData(d)
}

func validateDashboardData(d dashboardGetter) error {
	panels := d.Get("panel").([]interface{})
	for i, p := range panels {
		pm := p.(map[string]interface{})
		viz := pm["visualization"].([]interface{})
		if len(viz) == 0 {
			return fmt.Errorf("panel[%d] %q: visualization is required", i, pm["name"])
		}
		vm := viz[0].(map[string]interface{})
		vizType := vm["type"].(string)
		queries := pm["query"].([]interface{})
		layout := pm["layout"].([]interface{})

		if vizType == "section" {
			if len(queries) > 0 {
				return fmt.Errorf("panel[%d] %q: section panels cannot have query blocks", i, pm["name"])
			}
			if len(layout) > 0 {
				return fmt.Errorf("panel[%d] %q: section panels cannot have layout block", i, pm["name"])
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

	d.Set("panel", flattenPanels(dash.Panels))
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
		Name:      d.Get("name").(string),
		Panels:    expandPanels(d.Get("panel").([]interface{})),
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
	panels := make([]*client.DashboardPanel, 0, len(input))
	for _, p := range input {
		pm := p.(map[string]interface{})
		panel := &client.DashboardPanel{
			ID:           pm["id"].(string),
			Name:         pm["name"].(string),
			DatasourceID: pm["datasource_id"].(string),
			Telemetry:    pm["telemetry"].(string),
			Unit:         pm["unit"].(string),
			Version:      pm["version"].(int),
		}

		if vizList := pm["visualization"].([]interface{}); len(vizList) > 0 {
			panel.Visualization = expandVisualization(vizList[0].(map[string]interface{}))
		}

		if layoutList := pm["layout"].([]interface{}); len(layoutList) > 0 {
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

		panel.PopulatedQueries = expandQueries(pm["query"].([]interface{}))
		panels = append(panels, panel)
	}
	return panels
}

func expandVisualization(m map[string]interface{}) *client.DashboardPanelVisualization {
	viz := &client.DashboardPanelVisualization{
		Type:      m["type"].(string),
		FullWidth: m["full_width"].(bool),
	}
	if l := m["timeseries_config"].([]interface{}); len(l) > 0 {
		c := l[0].(map[string]interface{})
		viz.TimeseriesConfig = &client.DashboardTimeseriesConfig{DisplayType: c["display_type"].(string)}
	}
	if l := m["bar_config"].([]interface{}); len(l) > 0 {
		c := l[0].(map[string]interface{})
		stacked := c["stacked"].(bool)
		viz.BarConfig = &client.DashboardBarConfig{
			Orientation: c["orientation"].(string),
			Stacked:     &stacked,
		}
	}
	if l := m["stat_config"].([]interface{}); len(l) > 0 {
		c := l[0].(map[string]interface{})
		viz.StatConfig = &client.DashboardStatConfig{
			Thresholds: expandStatThresholds(c["threshold"].([]interface{})),
		}
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

func expandVariables(input []interface{}) []*client.DashboardVariable {
	out := make([]*client.DashboardVariable, 0, len(input))
	for _, v := range input {
		vm := v.(map[string]interface{})
		dv := &client.DashboardVariable{
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

func flattenPanels(panels []*client.DashboardPanel) []interface{} {
	out := make([]interface{}, 0, len(panels))
	for _, p := range panels {
		if p == nil {
			continue
		}
		pm := map[string]interface{}{
			"id":            p.ID,
			"name":          p.Name,
			"datasource_id": p.DatasourceID,
			"telemetry":     p.Telemetry,
			"unit":          p.Unit,
			"version":       p.Version,
			"visualization": flattenVisualization(p.Visualization),
			"query":         flattenQueries(p.PopulatedQueries),
			"layout":        flattenLayout(p.Layout),
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
		"type":       viz.Type,
		"full_width": viz.FullWidth,
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
				"value": t.Value,
				"color": t.Color,
			})
		}
		m["stat_config"] = []interface{}{map[string]interface{}{"threshold": thresholds}}
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
	for _, q := range queries {
		if q == nil {
			continue
		}
		qm := map[string]interface{}{
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
