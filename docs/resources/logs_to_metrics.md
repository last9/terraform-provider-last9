---
page_title: "last9_logs_to_metrics Resource - Last9"
subcategory: ""
description: |-
  Manages a Last9 logs-to-metrics (L2M) rule that aggregates logs into a custom metric.
---

# last9_logs_to_metrics (Resource)

Manages a Last9 logs-to-metrics (L2M) rule. The rule continuously aggregates matching logs into a custom metric at the configured frequency. This is a metric-only rule — it does not create an alert. Use [`last9_alert`](alert.md) to alert on the resulting metric.

The entire aggregation — the filter, the aggregate function, the group-by, and the window — is expressed in `query`. For `logjson-aggregate` this is a JSON pipeline; for `logql-aggregate` it is a logQL string with an aggregation. The output metric's labels come from the aggregate stage's `groupby`.

For traces, use [`last9_traces_to_metrics`](traces_to_metrics.md). For logs aggregations that also raise an alert, use [`last9_scheduled_search_alert`](scheduled_search_alert.md).

## Example Usage

```terraform
resource "last9_logs_to_metrics" "errors_by_service" {
  region           = "ap-south-1"
  name             = "errors_by_service"
  query_type       = "logjson-aggregate"
  metric_name      = "log_errors_by_service"
  search_frequency = 60

  query = jsonencode([
    {
      type  = "filter"
      query = { "$and" = [{ "$eq" = ["level", "error"] }] }
    },
    {
      type = "aggregate"
      aggregates = [
        { function = { "$count" = [] }, as = "log_errors_by_service" }
      ]
      groupby = {
        "attributes['service']" = "service"
      }
      window = ["1", "minutes"]
    }
  ])
}
```

## Schema

### Required

- `region` (String) Region for the rule. Changing this forces a new resource.
- `name` (String) Name of the rule.
- `query` (String) The full aggregation. For `logjson-aggregate`, a JSON array (filter stage + aggregate stage with `aggregates`, `groupby`, `window`). For `logql-aggregate`, a logQL string with aggregation.
- `metric_name` (String) Name of the output metric. Must start with a letter, underscore, or colon and contain only letters, numbers, underscores, and colons.
- `search_frequency` (Number) How often to run the aggregation, in seconds (60–86400).

### Optional

- `query_type` (String) One of `logql-aggregate`, `logjson-aggregate`. Defaults to `logjson-aggregate`.
- `physical_index` (String) Physical index to query. Defaults to `logs`.
- `resultant_query` (String) The resolved pipeline actually executed. If omitted, it is derived from `query` by renaming the terminal aggregate's output to `result` (the field the runtime reads the metric value from). Set this explicitly only for advanced cases such as a `logql-aggregate` string query.

## Import

```shell
terraform import last9_logs_to_metrics.errors_by_service ap-south-1:rule-id-123:errors_by_service
```
