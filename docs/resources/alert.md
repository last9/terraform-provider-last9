---
page_title: "last9_alert Resource - Last9"
subcategory: ""
description: |-
  Creates a metric-based alert rule with static thresholds or expressions.
---

# last9_alert (Resource)

Creates a metric-based alert rule. Alerts evaluate PromQL queries against thresholds and notify when conditions are met.

For more information about alerting concepts, see the [Alerting Overview](https://last9.io/docs/alerting-overview/) and [Configuring an Alert](https://last9.io/docs/configuring-an-alert/) documentation.

-> **Note** For log-based alerting, use `last9_scheduled_search_alert` instead.

## Example Usage

### Threshold Alert (Greater Than)

```terraform
# First, create an alert group
resource "last9_entity" "api_alerts" {
  name         = "api-service"
  type         = "service"
  entity_class = "alert-manager"
  external_ref = "api-service-prod"
  ui_readonly  = true
}

# Create an alert in the group
resource "last9_alert" "high_error_rate" {
  entity_id     = last9_entity.api_alerts.id
  name          = "High Error Rate"
  description   = "Alert when error rate exceeds 100 per minute"
  query         = "sum(rate(http_errors_total[5m]))"

  greater_than  = 100
  bad_minutes   = 5
  total_minutes = 10

  severity = "breach"

  notification_channels = [last9_notification_channel.slack.id]

  properties {
    runbook_url = "https://wiki.example.com/runbooks/high-error-rate"
    annotations = {
      priority = "high"
      team     = "platform"
    }
  }
}
```

### Threshold Alert (Less Than)

```terraform
resource "last9_alert" "low_availability" {
  entity_id     = last9_entity.api_alerts.id
  name          = "Low Availability"
  description   = "Alert when availability drops below 99.5%"
  query         = "avg(up{service=\"api\"})"

  less_than     = 99.5
  bad_minutes   = 3
  total_minutes = 10

  severity = "threat"
}
```

### Loss of Signal Alert

```terraform
resource "last9_alert" "service_down" {
  entity_id     = last9_entity.api_alerts.id
  name          = "Service Down"
  description   = "Alert when service stops reporting metrics"
  query         = "up{service=\"api\"}"

  less_than     = 1
  bad_minutes   = 3
  total_minutes = 5

  severity = "breach"
}
```

## Schema

### Required

- `entity_id` (String) ID of the alert group (`last9_entity`) this alert belongs to.
- `name` (String) Alert name.
- `query` (String) PromQL query that defines the metric for this alert. A KPI is automatically created.

### Optional

- `description` (String) Alert description.
- `severity` (String) Alert severity: `breach` or `threat`. Default: `breach`.
- `greater_than` (Number) Fire when query result exceeds this value.
- `less_than` (Number) Fire when query result drops below this value.
- `bad_minutes` (Number) Minutes the condition must be true before firing.
- `total_minutes` (Number) Evaluation window in minutes.
- `is_disabled` (Boolean) Disable the alert. Default: `false`.
- `group_timeseries_notifications` (Boolean) Group notifications for multiple time series. Default: `true`.
- `notification_channels` (List of String) Notification channel IDs to notify when alert fires.
- `properties` (Block) Additional alert properties. See [Properties](#properties).

### Read-Only

- `id` (String) Alert ID.
- `indicator` (String) Indicator name (derived from auto-created KPI).
- `expression` (String) Alert expression (computed from KPI).
- `kpi_id` (String) ID of the automatically created KPI.
- `kpi_name` (String) Name of the automatically created KPI.

### Properties

The `properties` block supports:

- `runbook_url` (String) Link to runbook for responders.
- `annotations` (Map of String) Custom key-value annotations. Supports dynamic template variables.

### Dynamic Annotations

Annotations support template variables for dynamic content in notifications:

| Variable | Description | Example |
|----------|-------------|---------|
| `{{ $labels.label_name }}` | Value of a metric label | `{{ $labels.service }}` |
| `{{ .Labels.label_name }}` | Alternative label syntax | `{{ .Labels.instance }}` |
| `{{ $value }}` | Current metric value (worst timeseries) | `{{ $value }}` |
| `{{ .Value }}` | Alternative value syntax | `{{ .Value }}` |

**Example:**

```terraform
properties {
  runbook_url = "https://wiki.example.com/runbooks/{{ $labels.service }}"
  annotations = {
    summary   = "High error rate on {{ $labels.service }}: {{ $value }}%"
    dashboard = "https://grafana.example.com/d/svc?var-service={{ $labels.service }}"
  }
}
```

-> **Note** When `group_timeseries_notifications` is enabled, `{{ $labels }}` shows label counts and `{{ $value }}` shows P99 of worst values across grouped timeseries.

## Import

Import using format `entity_id:alert_id`:

```shell
terraform import last9_alert.example <entity_id>:<alert_id>
```
