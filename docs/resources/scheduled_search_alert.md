---
page_title: "last9_scheduled_search_alert Resource - Last9"
subcategory: ""
description: |-
  Manages a Last9 scheduled search alert for log-based alerting.
---

# last9_scheduled_search_alert (Resource)

Manages a Last9 scheduled search alert. These alerts run log queries on a schedule and trigger when results match threshold conditions.

## Example Usage

### Basic Error Count Alert

```terraform
resource "last9_scheduled_search_alert" "error_count" {
  region           = "ap-south-1"
  name             = "High Error Count Alert"
  query_type       = "logjson-aggregate"
  physical_index   = "logs"
  telemetry        = "logs"
  search_frequency = 300  # 5 minutes

  query = jsonencode([
    {
      type  = "filter"
      query = {
        "$and" = [
          { "$eq" = ["SeverityText", "ERROR"] }
        ]
      }
    }
  ])

  post_processor {
    type = "aggregate"

    aggregates {
      function = jsonencode({ "$count" = [] })
      as       = "error_count"
    }

    groupby = "{}"
  }

  threshold {
    operator = ">"
    value    = 100
  }

  alert_destinations = [last9_notification_channel.slack.id]
}
```

### Alert with Grouping

```terraform
resource "last9_scheduled_search_alert" "errors_by_service" {
  region           = "ap-south-1"
  name             = "Errors by Service"
  query_type       = "logjson-aggregate"
  physical_index   = "logs"
  telemetry        = "logs"
  search_frequency = 600  # 10 minutes

  query = jsonencode([
    {
      type  = "filter"
      query = {
        "$and" = [
          { "$eq" = ["SeverityText", "ERROR"] }
        ]
      }
    }
  ])

  post_processor {
    type = "aggregate"

    aggregates {
      function = jsonencode({ "$count" = [] })
      as       = "error_count"
    }

    groupby = jsonencode({ "service" = "$attributes.service" })
  }

  threshold {
    operator = ">="
    value    = 50
  }

  alert_destinations = [last9_notification_channel.slack.id]
}
```

## Schema

### Required

- `region` (String) Region for the alert (e.g., "ap-south-1").
- `name` (String) Name of the scheduled search alert.
- `query` (String) JSON-encoded LogJSON query pipeline.
- `post_processor` (Block) Post-processor configuration. See [Post Processor](#post-processor) below.
- `search_frequency` (Number) Search frequency in seconds (60-86400).
- `threshold` (Block) Threshold configuration. See [Threshold](#threshold) below.
- `alert_destinations` (List of Number) List of notification destination IDs.

### Optional

- `query_type` (String) Query type. Default: `logjson-aggregate`.
- `physical_index` (String) Physical index to search. Default: `logs`.
- `telemetry` (String) Telemetry type. Default: `logs`.

### Read-Only

- `id` (String) The ID of the scheduled search alert.

### Post Processor

The `post_processor` block supports:

- `type` (String, Required) Post-processor type (e.g., "aggregate").
- `aggregates` (Block List, Required) Aggregation functions. See [Aggregates](#aggregates) below.
- `groupby` (String) JSON-encoded grouping specification.

### Aggregates

The `aggregates` block supports:

- `function` (String, Required) JSON-encoded aggregation function (e.g., `{"$count": []}`, `{"$sum": ["field"]}`, `{"$avg": ["field"]}`).
- `as` (String, Required) Output field name for the aggregation result.

### Threshold

The `threshold` block supports:

- `operator` (String, Required) Comparison operator. Valid values: `>`, `<`, `>=`, `<=`, `==`, `!=`.
- `value` (Number, Required) Threshold value.

## Import

Scheduled search alerts can be imported using the format `region:id:name`:

```shell
terraform import last9_scheduled_search_alert.example ap-south-1:abc123:my-alert
```
