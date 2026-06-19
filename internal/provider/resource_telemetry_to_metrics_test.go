package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const tracePipeline = `[{"type":"filter","query":{"$and":[{"$regex":["ServiceName","probe-.*"]}]}},{"type":"aggregate","aggregates":[{"function":{"$count":[]},"as":"probe_up"}],"groupby":{"attributes['check_name']":"check_name"},"window":["1","minutes"]}]`

func TestBuildTelemetryToMetricsRule(t *testing.T) {
	r := resourceTracesToMetrics()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		"region":           "ap-southeast-1",
		"name":             "probe-up",
		"query_type":       "tracejson-aggregate",
		"physical_index":   "traces",
		"query":            tracePipeline,
		"metric_name":      "probe_up",
		"search_frequency": 60,
	})

	cfg := telemetryToMetricsConfig{telemetry: "traces"}
	rule := buildTelemetryToMetricsRule(cfg, d)

	if rule.RuleType != ruleTypeStreamingAggregation {
		t.Errorf("RuleType = %q, want %q", rule.RuleType, ruleTypeStreamingAggregation)
	}
	if rule.Properties.Telemetry != "traces" {
		t.Errorf("Telemetry = %q, want traces", rule.Properties.Telemetry)
	}
	if rule.Properties.MetricName != "probe_up" {
		t.Errorf("MetricName = %q, want probe_up", rule.Properties.MetricName)
	}
	// Aggregation lives in the query pipeline; post_processor must be empty.
	if len(rule.Properties.PostProcessor) != 0 {
		t.Errorf("PostProcessor should be empty, got %d", len(rule.Properties.PostProcessor))
	}
	// resultant_query is derived from query: the terminal aggregate is renamed
	// to "result" (the field the runtime reads the metric value from).
	if !strings.Contains(rule.Properties.ResultantQuery, `"as":"result"`) {
		t.Errorf("ResultantQuery terminal aggregate should be renamed to result, got %q", rule.Properties.ResultantQuery)
	}
	// Metric-only rules carry no alert configuration.
	if len(rule.Properties.AlertDestinations) != 0 {
		t.Errorf("AlertDestinations should be empty, got %d", len(rule.Properties.AlertDestinations))
	}
}

func TestResultantQueryOverride(t *testing.T) {
	r := resourceTracesToMetrics()
	override := `[{"type":"filter","query":{"$and":[]}}]`
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		"region":           "ap-southeast-1",
		"name":             "probe-up",
		"query":            tracePipeline,
		"resultant_query":  override,
		"metric_name":      "probe_up",
		"search_frequency": 60,
	})

	rule := buildTelemetryToMetricsRule(telemetryToMetricsConfig{telemetry: "traces"}, d)
	if rule.Properties.ResultantQuery != override {
		t.Errorf("ResultantQuery = %q, want override %q", rule.Properties.ResultantQuery, override)
	}
}

func TestDeriveResultantQuery(t *testing.T) {
	// Two aggregate stages: only the terminal aggregate of the LAST stage is
	// renamed to "result"; the earlier "_count" is preserved.
	query := `[{"type":"filter","query":{"$and":[{"$neq":["ServiceName",""]}]}},{"type":"aggregate","aggregates":[{"as":"_count","function":{"$count":[]}}],"groupby":{"attributes['_caller']":"_caller"}},{"type":"aggregate","aggregates":[{"as":"avg__count","function":{"$avg":["_count"]}}],"groupby":{"_caller":"_caller"}}]`

	got := deriveResultantQuery(query)
	if !strings.Contains(got, `"as":"result"`) {
		t.Errorf("terminal aggregate not renamed to result: %s", got)
	}
	if strings.Contains(got, "avg__count") {
		t.Errorf("terminal as avg__count should have been replaced: %s", got)
	}
	if !strings.Contains(got, `"as":"_count"`) {
		t.Errorf("earlier aggregate _count should be preserved: %s", got)
	}
}

func TestDeriveResultantQueryNonJSON(t *testing.T) {
	// logQL string (not a JSON pipeline) is returned unchanged.
	logql := `sum by (service) (rate({app="x"}[5m]))`
	if got := deriveResultantQuery(logql); got != logql {
		t.Errorf("non-JSON query should pass through unchanged, got %q", got)
	}
}

func TestTracesToMetricsDefaults(t *testing.T) {
	r := resourceTracesToMetrics()

	if qt := r.Schema["query_type"]; qt.Default != "tracejson-aggregate" {
		t.Errorf("traces query_type default = %v, want tracejson-aggregate", qt.Default)
	}
	if pi := r.Schema["physical_index"]; pi.Default != "traces" {
		t.Errorf("traces physical_index default = %v, want traces", pi.Default)
	}
}

func TestLogsToMetricsDefaults(t *testing.T) {
	r := resourceLogsToMetrics()

	if qt := r.Schema["query_type"]; qt.Default != "logjson-aggregate" {
		t.Errorf("logs query_type default = %v, want logjson-aggregate", qt.Default)
	}
	if pi := r.Schema["physical_index"]; pi.Default != "logs" {
		t.Errorf("logs physical_index default = %v, want logs", pi.Default)
	}
}

func TestMetricNameRegexp(t *testing.T) {
	valid := []string{"errors_by_service", "_foo", ":svc:total", "Metric123"}
	for _, v := range valid {
		if !metricNameRegexp.MatchString(v) {
			t.Errorf("metric_name %q should be valid", v)
		}
	}

	invalid := []string{"1metric", "has-dash", "has space", "has.dot", ""}
	for _, v := range invalid {
		if metricNameRegexp.MatchString(v) {
			t.Errorf("metric_name %q should be invalid", v)
		}
	}
}
