---
page_title: "last9_traces_to_metrics Resource - Last9"
subcategory: ""
description: |-
  Manages a Last9 traces-to-metrics (T2M) rule that aggregates traces into a custom metric.
---

# last9_traces_to_metrics (Resource)

Manages a Last9 traces-to-metrics (T2M) rule. The rule continuously aggregates matching spans into a custom metric at the configured frequency. This is a metric-only rule — it does not create an alert. Use [`last9_alert`](alert.md) to alert on the resulting metric.

The entire aggregation — the filter, the aggregate function, the group-by, and the window — is expressed as a JSON pipeline in `query`. The output metric's labels come from the aggregate stage's `groupby`.

For logs, use [`last9_logs_to_metrics`](logs_to_metrics.md).

## Example Usage

```terraform
resource "last9_traces_to_metrics" "probe_up" {
  region           = "ap-southeast-1"
  name             = "probe_up"
  query_type       = "tracejson-aggregate"
  metric_name      = "probe_up"
  search_frequency = 60

  query = jsonencode([
    {
      type  = "filter"
      query = { "$and" = [{ "$regex" = ["ServiceName", "probe-.*"] }] }
    },
    {
      type = "aggregate"
      aggregates = [
        { function = { "$count" = [] }, as = "probe_up" }
      ]
      groupby = {
        "attributes['check_name']" = "check_name"
        "attributes['location']"   = "location"
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
- `query` (String) The full aggregation pipeline as a JSON array: a `filter` stage followed by an `aggregate` stage with `aggregates`, `groupby`, and `window`.
- `metric_name` (String) Name of the output metric. Must start with a letter, underscore, or colon and contain only letters, numbers, underscores, and colons.
- `search_frequency` (Number) How often to run the aggregation, in seconds (60–86400).

### Optional

- `query_type` (String) `tracejson-aggregate`. Defaults to `tracejson-aggregate`.
- `physical_index` (String) Physical index to query. Defaults to `traces`.
- `resultant_query` (String) The resolved pipeline actually executed. If omitted, it is derived from `query` by renaming the terminal aggregate's output to `result` (the field the runtime reads the metric value from). Set this explicitly only for advanced cases such as a `logql-aggregate` string query.

## Import

```shell
terraform import last9_traces_to_metrics.probe_up ap-southeast-1:rule-id-123:probe_up
```
