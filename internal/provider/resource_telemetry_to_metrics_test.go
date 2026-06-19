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

func TestDeriveResultantQueryUnchangedCases(t *testing.T) {
	// Each input must be returned byte-for-byte unchanged.
	cases := map[string]string{
		"no aggregate stage":    `[{"type":"filter","query":{"$and":[]}}]`,
		"empty aggregates":      `[{"type":"aggregate","aggregates":[]}]`,
		"malformed json array":  `[{"type":"aggregate",`,
		"json object not array": `{"type":"aggregate","aggregates":[{"as":"x"}]}`,
		"terminal not object":   `[{"type":"aggregate","aggregates":["notanobject"]}]`,
	}
	for name, in := range cases {
		if got := deriveResultantQuery(in); got != in {
			t.Errorf("%s: expected unchanged, got %q", name, got)
		}
	}
}

func TestDeriveResultantQueryMultipleAggregatesInStage(t *testing.T) {
	// Only the terminal aggregate of the last stage is renamed; siblings stay.
	in := `[{"type":"aggregate","aggregates":[{"as":"keep","function":{"$count":[]}},{"as":"rename","function":{"$avg":["x"]}}]}]`
	got := deriveResultantQuery(in)
	if !strings.Contains(got, `"as":"result"`) {
		t.Errorf("terminal aggregate should become result: %s", got)
	}
	if !strings.Contains(got, `"as":"keep"`) {
		t.Errorf("non-terminal aggregate should be preserved: %s", got)
	}
	if strings.Contains(got, `"as":"rename"`) {
		t.Errorf("terminal aggregate 'rename' should have been replaced: %s", got)
	}
}

func TestDeriveResultantQueryIdempotent(t *testing.T) {
	// A pipeline whose terminal aggregate is already "result" stays "result".
	in := `[{"type":"aggregate","aggregates":[{"as":"result","function":{"$count":[]}}]}]`
	got := deriveResultantQuery(in)
	if !strings.Contains(got, `"as":"result"`) || strings.Count(got, `"result"`) != 1 {
		t.Errorf("already-result pipeline should remain result: %s", got)
	}
}

func TestParseTelemetryToMetricsID(t *testing.T) {
	region, id, name, err := parseTelemetryToMetricsID("ap-south-1:abc-123:my-rule")
	if err != nil || region != "ap-south-1" || id != "abc-123" || name != "my-rule" {
		t.Errorf("normal parse failed: %q %q %q %v", region, id, name, err)
	}

	// A name containing ':' must be preserved (SplitN with limit 3).
	_, _, name, err = parseTelemetryToMetricsID("ap-south-1:abc-123:errors:prod")
	if err != nil || name != "errors:prod" {
		t.Errorf("colon-in-name not preserved: name=%q err=%v", name, err)
	}

	if _, _, _, err := parseTelemetryToMetricsID("too:few"); err == nil {
		t.Error("expected error for malformed ID, got nil")
	}
}

func TestQueryTypeAllowlist(t *testing.T) {
	logs := resourceLogsToMetrics().Schema["query_type"]
	traces := resourceTracesToMetrics().Schema["query_type"]

	// logs accepts log query types, rejects the traces one.
	if _, errs := logs.ValidateFunc("logjson-aggregate", "query_type"); len(errs) != 0 {
		t.Errorf("logs should accept logjson-aggregate: %v", errs)
	}
	if _, errs := logs.ValidateFunc("tracejson-aggregate", "query_type"); len(errs) == 0 {
		t.Error("logs should reject tracejson-aggregate")
	}
	// traces accepts the trace query type, rejects a logs one.
	if _, errs := traces.ValidateFunc("tracejson-aggregate", "query_type"); len(errs) != 0 {
		t.Errorf("traces should accept tracejson-aggregate: %v", errs)
	}
	if _, errs := traces.ValidateFunc("logql-aggregate", "query_type"); len(errs) == 0 {
		t.Error("traces should reject logql-aggregate")
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
