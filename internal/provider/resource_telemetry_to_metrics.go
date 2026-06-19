package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

// Metric-only logs-to-metrics (L2M) and traces-to-metrics (T2M) rules are
// scheduled searches with rule_type "streaming_aggregation". Unlike
// last9_scheduled_search_alert (rule_type "scheduled_search"), these do not
// create an entity, KPI, or alert — they only emit a custom metric.
//
// The aggregation (filter + aggregate + groupby + window) lives entirely inside
// the `query` JSON pipeline; post_processor is sent empty, matching the product
// UI. The output metric's labels come from the pipeline's groupby stage.
const ruleTypeStreamingAggregation = "streaming_aggregation"

// metricNameRegexp mirrors the server-side Prometheus metric name validation.
var metricNameRegexp = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)

// telemetryToMetricsConfig captures the per-telemetry differences between the
// logs and traces resources, which otherwise share all CRUD logic.
type telemetryToMetricsConfig struct {
	telemetry            string
	allowedQueryTypes    []string
	defaultQueryType     string
	defaultPhysicalIndex string
}

func resourceLogsToMetrics() *schema.Resource {
	return telemetryToMetricsResource(telemetryToMetricsConfig{
		telemetry:            "logs",
		allowedQueryTypes:    []string{"logql-aggregate", "logjson-aggregate"},
		defaultQueryType:     "logjson-aggregate",
		defaultPhysicalIndex: "logs",
	}, "Manages a Last9 logs-to-metrics (L2M) rule that aggregates logs into a custom metric.")
}

func resourceTracesToMetrics() *schema.Resource {
	return telemetryToMetricsResource(telemetryToMetricsConfig{
		telemetry:            "traces",
		allowedQueryTypes:    []string{"tracejson-aggregate"},
		defaultQueryType:     "tracejson-aggregate",
		defaultPhysicalIndex: "traces",
	}, "Manages a Last9 traces-to-metrics (T2M) rule that aggregates traces into a custom metric.")
}

func telemetryToMetricsResource(cfg telemetryToMetricsConfig, description string) *schema.Resource {
	return &schema.Resource{
		Description:   description,
		CreateContext: makeTelemetryToMetricsCreate(cfg),
		ReadContext:   telemetryToMetricsRead,
		UpdateContext: makeTelemetryToMetricsUpdate(cfg),
		DeleteContext: telemetryToMetricsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Region for the rule.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the rule.",
			},
			"query_type": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      cfg.defaultQueryType,
				Description:  fmt.Sprintf("Query type. One of: %s.", strings.Join(cfg.allowedQueryTypes, ", ")),
				ValidateFunc: validation.StringInSlice(cfg.allowedQueryTypes, false),
			},
			"physical_index": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     cfg.defaultPhysicalIndex,
				Description: "Physical index to query.",
			},
			"query": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The full aggregation. For logjson-aggregate / tracejson-aggregate, a JSON pipeline array (filter stage + aggregate stage with groupby and window). For logql-aggregate, a logQL string with an aggregation. The output metric's labels come from the aggregate stage's groupby.",
				// The server may re-serialize the JSON pipeline (key order, spacing);
				// suppress diffs that are semantically equal JSON.
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return jsonStringsEqual(old, new)
				},
			},
			"resultant_query": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The resolved query executed by the rule. Defaults to the value of query; only set this if the executed pipeline must differ from the authored one.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return jsonStringsEqual(old, new)
				},
			},
			"metric_name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Name of the output metric. Must start with a letter, underscore, or colon and contain only letters, numbers, underscores, and colons.",
				ValidateFunc: validation.StringMatch(metricNameRegexp, "metric_name must start with a letter, underscore, or colon and contain only letters, numbers, underscores, and colons"),
			},
			"search_frequency": {
				Type:         schema.TypeInt,
				Required:     true,
				Description:  "How often to run the aggregation, in seconds (minimum 60, maximum 86400).",
				ValidateFunc: validation.IntBetween(60, 86400),
			},
		},
	}
}

func makeTelemetryToMetricsCreate(cfg telemetryToMetricsConfig) schema.CreateContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		apiClient := m.(*client.Client)
		region := d.Get("region").(string)

		rule := buildTelemetryToMetricsRule(cfg, d)

		result, err := apiClient.CreateScheduledSearchAlert(region, rule)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to create %s-to-metrics rule: %w", cfg.telemetry, err))
		}

		d.SetId(fmt.Sprintf("%s:%s:%s", region, result.ID, result.RuleName))

		return telemetryToMetricsRead(ctx, d, m)
	}
}

// parseTelemetryToMetricsID splits the composite resource ID into its parts.
// SplitN with a limit of 3 keeps any ":" embedded in the rule name intact.
func parseTelemetryToMetricsID(id string) (region, ruleID, name string, err error) {
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid ID format: %s (expected region:id:name)", id)
	}
	return parts[0], parts[1], parts[2], nil
}

func telemetryToMetricsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	region, ruleID, _, err := parseTelemetryToMetricsID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Metric-only rules are only returned when rule_type is requested explicitly.
	rules, err := apiClient.GetScheduledSearchRules(region, ruleTypeStreamingAggregation)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read telemetry-to-metrics rules: %w", err))
	}

	// Match on the unique rule ID, not the name: logs and traces rules share the
	// streaming_aggregation rule_type, and names are not guaranteed unique.
	var rule *client.ScheduledSearchAlertFull
	for i := range rules {
		if rules[i].ID == ruleID {
			rule = &rules[i]
			break
		}
	}

	if rule == nil {
		d.SetId("")
		return nil
	}

	d.Set("region", region)
	d.Set("name", rule.RuleName)
	d.Set("query_type", rule.QueryType)
	d.Set("physical_index", rule.PhysicalIndex)
	d.Set("query", rule.Properties.Query)
	d.Set("resultant_query", rule.Properties.ResultantQuery)
	d.Set("metric_name", rule.Properties.MetricName)
	d.Set("search_frequency", rule.Properties.SearchFrequency)

	return nil
}

func makeTelemetryToMetricsUpdate(cfg telemetryToMetricsConfig) schema.UpdateContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		apiClient := m.(*client.Client)

		region, ruleID, _, err := parseTelemetryToMetricsID(d.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		rule := buildTelemetryToMetricsRule(cfg, d)

		result, err := apiClient.UpdateScheduledSearchAlert(region, ruleID, rule)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to update %s-to-metrics rule: %w", cfg.telemetry, err))
		}

		d.SetId(fmt.Sprintf("%s:%s:%s", region, result.ID, result.RuleName))

		return telemetryToMetricsRead(ctx, d, m)
	}
}

func telemetryToMetricsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	region, ruleID, _, err := parseTelemetryToMetricsID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := apiClient.DeleteScheduledSearchAlert(region, ruleID); err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete telemetry-to-metrics rule: %w", err))
	}

	d.SetId("")
	return nil
}

// deriveResultantQuery produces the executed pipeline from the authored query.
// The scheduled-search runtime extracts the metric value from a field named
// "result", so the terminal aggregate of the last aggregate stage is renamed to
// "result". If query is not a JSON pipeline (e.g. a logQL string) or has no
// aggregate stage, it is returned unchanged — supply resultant_query explicitly
// in that case.
func deriveResultantQuery(query string) string {
	var stages []map[string]interface{}
	if err := json.Unmarshal([]byte(query), &stages); err != nil {
		return query
	}

	lastAgg := -1
	for i, stage := range stages {
		if stage["type"] == "aggregate" {
			lastAgg = i
		}
	}
	if lastAgg == -1 {
		return query
	}

	aggregates, ok := stages[lastAgg]["aggregates"].([]interface{})
	if !ok || len(aggregates) == 0 {
		return query
	}
	terminal, ok := aggregates[len(aggregates)-1].(map[string]interface{})
	if !ok {
		return query
	}
	terminal["as"] = "result"

	out, err := json.Marshal(stages)
	if err != nil {
		return query
	}
	return string(out)
}

func buildTelemetryToMetricsRule(cfg telemetryToMetricsConfig, d *schema.ResourceData) *client.ScheduledSearchAlert {
	query := d.Get("query").(string)

	// resultant_query is the pipeline actually executed. The job reads the
	// value from a field named "result", so the executed pipeline's terminal
	// aggregate must output "result". When the user does not supply
	// resultant_query, derive it from query by renaming that terminal output.
	resultantQuery := d.Get("resultant_query").(string)
	if resultantQuery == "" {
		resultantQuery = deriveResultantQuery(query)
	}

	return &client.ScheduledSearchAlert{
		RuleName:      d.Get("name").(string),
		QueryType:     d.Get("query_type").(string),
		PhysicalIndex: d.Get("physical_index").(string),
		RuleType:      ruleTypeStreamingAggregation,
		Properties: client.ScheduledSearchProperties{
			Telemetry:       cfg.telemetry,
			Query:           query,
			ResultantQuery:  resultantQuery,
			PostProcessor:   []client.PostProcessor{}, // aggregation lives in the query pipeline
			SearchFrequency: d.Get("search_frequency").(int),
			MetricName:      d.Get("metric_name").(string),
		},
	}
}
