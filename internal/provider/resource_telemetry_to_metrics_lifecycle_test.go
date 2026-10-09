package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func TestTelemetryToMetricsQueryUpdateRecomputesResultantQuery(t *testing.T) {
	queryV1 := telemetryToMetricsTestQuery("service-v1")
	queryV2 := telemetryToMetricsTestQuery("service-v2")
	for _, tc := range []struct {
		name     string
		resource *schema.Resource
		cfg      telemetryToMetricsConfig
		query    string
		override string
	}{
		{"logs", resourceLogsToMetrics(), telemetryToMetricsConfig{telemetry: "logs"}, `logjson-aggregate`, ""},
		{"traces", resourceTracesToMetrics(), telemetryToMetricsConfig{telemetry: "traces"}, `tracejson-aggregate`, ""},
		{"logs explicit override", resourceLogsToMetrics(), telemetryToMetricsConfig{telemetry: "logs"}, `logjson-aggregate`, `sum by (service_name) (rate({app="api"}[5m]))`},
		{"traces explicit override", resourceTracesToMetrics(), telemetryToMetricsConfig{telemetry: "traces"}, `tracejson-aggregate`, `sum by (service_name) (rate({app="api"}[5m]))`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.resource
			initialConfig := telemetryToMetricsLifecycleConfig(tc.query, queryV1)
			if tc.override != "" {
				initialConfig["resultant_query"] = tc.override
			}
			prior := schema.TestResourceDataRaw(t, r.Schema, initialConfig)
			prior.SetId("ap-south-1:rule-1:query-change")
			if tc.override == "" {
				prior.Set("resultant_query", deriveResultantQuery(queryV1))
			}
			current := *buildTelemetryToMetricsRule(tc.cfg, prior)

			var captured client.ScheduledSearchAlert
			calls := 0
			apiClient := reviewClient(t, func(w http.ResponseWriter, req *http.Request) {
				switch req.Method {
				case http.MethodPut:
					if !strings.HasSuffix(req.URL.Path, "/logs_settings/scheduled_search/rule-1") || req.URL.Query().Get("region") != "ap-south-1" {
						t.Errorf("unexpected update endpoint: %s?%s", req.URL.Path, req.URL.RawQuery)
					}
					calls++
					if err := json.NewDecoder(req.Body).Decode(&captured); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					current = captured
					writeTelemetryToMetricsResponse(t, w, current)
				case http.MethodGet:
					if !strings.HasSuffix(req.URL.Path, "/logs_settings/scheduled_search") || req.URL.Query().Get("rule_type") != ruleTypeStreamingAggregation {
						t.Errorf("unexpected refresh endpoint: %s?%s", req.URL.Path, req.URL.RawQuery)
					}
					writeTelemetryToMetricsListResponse(t, w, current)
				default:
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			})

			updatedConfig := telemetryToMetricsLifecycleConfig(tc.query, queryV2)
			if tc.override != "" {
				updatedConfig["resultant_query"] = tc.override
			}
			diff, err := r.Diff(context.Background(), prior.State(), terraform.NewResourceConfigRaw(updatedConfig), apiClient)
			if err != nil {
				t.Fatal(err)
			}
			diff.RawConfig, err = schema.JSONMapToStateValue(updatedConfig, r.CoreConfigSchema())
			if err != nil {
				t.Fatal(err)
			}
			state, diags := r.Apply(context.Background(), prior.State(), diff, apiClient)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if calls != 1 {
				t.Fatalf("expected one update request, got %d", calls)
			}
			if captured.Properties.Query != queryV2 {
				t.Fatalf("request query = %q, want %q", captured.Properties.Query, queryV2)
			}
			wantResultant := deriveResultantQuery(queryV2)
			if tc.override != "" {
				wantResultant = tc.override
			}
			if captured.Properties.ResultantQuery != wantResultant {
				t.Fatalf("request resultant_query = %q, want regenerated v2 pipeline %q", captured.Properties.ResultantQuery, wantResultant)
			}

			importData := schema.TestResourceDataRaw(t, r.Schema, nil)
			importData.SetId(state.ID)
			imported, err := r.Importer.StateContext(context.Background(), importData, apiClient)
			if err != nil {
				t.Fatal(err)
			}
			state, diags = r.RefreshWithoutUpgrade(context.Background(), imported[0].State(), apiClient)
			if diags.HasError() {
				t.Fatal(diags)
			}
			diff, err = r.Diff(context.Background(), state, terraform.NewResourceConfigRaw(updatedConfig), apiClient)
			if err != nil {
				t.Fatal(err)
			}
			if diff != nil && !diff.Empty() {
				t.Fatalf("plan after refresh/import should be empty, got %#v", diff)
			}
		})
	}
}

func telemetryToMetricsLifecycleConfig(queryType, query string) map[string]interface{} {
	physicalIndex := "logs"
	if queryType == "tracejson-aggregate" {
		physicalIndex = "traces"
	}
	return map[string]interface{}{
		"region":           "ap-south-1",
		"name":             "query-change-v1",
		"query_type":       queryType,
		"physical_index":   physicalIndex,
		"query":            query,
		"metric_name":      "requests_total",
		"search_frequency": 60,
	}
}

func telemetryToMetricsTestQuery(service string) string {
	return fmt.Sprintf(`[{"type":"filter","query":{"$and":[{"$eq":["ServiceName",%q]}]}},{"type":"aggregate","aggregates":[{"function":{"$count":[]},"as":"requests"}],"groupby":{"ServiceName":"service_name"},"window":["1","minutes"]}]`, service)
}

func writeTelemetryToMetricsResponse(t *testing.T, w http.ResponseWriter, rule client.ScheduledSearchAlert) {
	t.Helper()
	response := client.ScheduledSearchAlertFull{
		ID: "rule-1", RuleName: rule.RuleName, RuleType: rule.RuleType,
		QueryType: rule.QueryType, PhysicalIndex: rule.PhysicalIndex, Region: "ap-south-1",
		Properties: rule.Properties,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		t.Error(err)
	}
}

func writeTelemetryToMetricsListResponse(t *testing.T, w http.ResponseWriter, rule client.ScheduledSearchAlert) {
	t.Helper()
	response := client.ScheduledSearchAlertFull{
		ID: "rule-1", RuleName: rule.RuleName, RuleType: rule.RuleType,
		QueryType: rule.QueryType, PhysicalIndex: rule.PhysicalIndex, Region: "ap-south-1",
		Properties: rule.Properties,
	}
	if err := json.NewEncoder(w).Encode([]client.ScheduledSearchAlertFull{response}); err != nil {
		t.Error(err)
	}
}
