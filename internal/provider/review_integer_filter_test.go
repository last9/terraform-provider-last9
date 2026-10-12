package provider

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestReviewResultantQueryPreservesIntegerFilter(t *testing.T) {
	for _, value := range []string{"42", "9007199254740993"} {
		for _, explicit := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/explicit=%v", value, explicit), func(t *testing.T) {
				query := fmt.Sprintf(`[{"type":"filter","query":{"$eq":["attributes['order_id']",%s]}},{"type":"aggregate","aggregates":[{"function":{"$count":[]},"as":"result"}],"window":["1","minutes"]}]`, value)
				config := map[string]interface{}{"region": "ap-south-1", "name": "orders", "metric_name": "orders_total", "query": query, "search_frequency": 60}
				if explicit {
					config["resultant_query"] = query
				}
				d := schema.TestResourceDataRaw(t, resourceLogsToMetrics().Schema, config)
				rule := buildTelemetryToMetricsRule(telemetryToMetricsConfig{telemetry: "logs"}, d)
				dec := json.NewDecoder(strings.NewReader(rule.Properties.ResultantQuery))
				dec.UseNumber()
				var stages []map[string]interface{}
				if err := dec.Decode(&stages); err != nil {
					t.Fatal(err)
				}
				got := stages[0]["query"].(map[string]interface{})["$eq"].([]interface{})[1].(json.Number).String()
				if got != value {
					t.Fatalf("executed order_id=%s; authored order_id=%s", got, value)
				}
			})
		}
	}
}
