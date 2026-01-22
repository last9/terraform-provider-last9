---
page_title: "last9_alert Resource - Last9"
subcategory: ""
description: |-
  Manages a Last9 alert rule with static thresholds or expressions.
---

# last9_alert (Resource)

Manages a Last9 alert rule. Alerts can be configured with static thresholds or expressions to monitor entity KPIs.

## Example Usage

### Static Threshold Alert

```terraform
resource "last9_alert" "high_error_rate" {
  entity_id     = last9_entity.api_service.id
  name          = "High Error Rate"
  description   = "Alert when error rate exceeds 100 per minute"
  query         = "sum(rate(http_errors_total[5m]))"

  greater_than  = 100
  bad_minutes   = 5
  total_minutes = 10

  severity = "breach"

  properties {
    runbook_url = "https://wiki.example.com/runbooks/high-error-rate"
    annotations = {
      priority = "high"
      team     = "platform"
    }
  }
}
```

### Less Than Threshold Alert

```terraform
resource "last9_alert" "low_availability" {
  entity_id     = last9_entity.api_service.id
  name          = "Low Availability"
  description   = "Alert when availability drops below 99.5%"
  query         = "avg(up{service=\"api\"})"

  less_than     = 99.5
  bad_minutes   = 3
  total_minutes = 10

  severity = "threat"
}
```

### Expression-Based Alert

```terraform
resource "last9_alert" "custom_expression" {
  entity_id   = last9_entity.api_service.id
  name        = "Custom Alert"
  description = "Custom expression-based alert"
  expression  = "low_spike(0.5, availability)"

  severity = "warning"
}
```

## Schema

### Required

- `entity_id` (String) ID of the entity this alert belongs to.
- `name` (String) Name of the alert rule.
- `severity` (String) Alert severity. Valid values: `breach`, `threat`, `warning`.

### Optional

- `description` (String) Description of the alert.
- `query` (String) PromQL query for the alert. Required for static threshold alerts.
- `expression` (String) Expression for the alert. Used for expression-based alerts.
- `greater_than` (Number) Threshold value for greater-than comparison.
- `less_than` (Number) Threshold value for less-than comparison.
- `bad_minutes` (Number) Number of minutes the condition must be true to trigger.
- `total_minutes` (Number) Evaluation window in minutes.
- `is_disabled` (Boolean) Whether the alert is disabled. Default: `false`.
- `group_timeseries_notifications` (Boolean) Group notifications for multiple time series. Default: `false`.
- `notification_channels` (List of Number) List of notification channel IDs.
- `properties` (Block) Alert properties block. See [Properties](#properties) below.

### Read-Only

- `id` (String) The ID of the alert.
- `indicator` (String) The KPI indicator name.
- `kpi_id` (String) The associated KPI ID.
- `kpi_name` (String) The associated KPI name.

### Properties

The `properties` block supports:

- `runbook_url` (String) URL to the runbook for this alert.
- `annotations` (Map of String) Additional annotations for the alert.

## Import

Alerts can be imported using the format `entity_id:alert_id`:

```shell
terraform import last9_alert.example <entity_id>:<alert_id>
```
