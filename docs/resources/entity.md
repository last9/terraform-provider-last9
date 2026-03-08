---
page_title: "last9_entity Resource - Last9"
subcategory: ""
description: |-
  Creates an alert group for organizing metric-based alerts.
---

# last9_entity (Resource)

Creates an alert group for organizing related metric-based alerts. Each `last9_alert` must belong to an alert group.

-> **Note** For log-based alerting, use `last9_scheduled_search_alert` instead, which does not require an alert group.

## Example Usage

### Alert Group with Alerts

```terraform
# Create an alert group
resource "last9_entity" "api_alerts" {
  name         = "api-service"
  type         = "service"
  entity_class = "alert-manager"
  external_ref = "api-service-prod"
  description  = "Alerts for Production API Service"
  ui_readonly  = true  # Manage via Terraform only

  labels = {
    tier        = "backend"
    environment = "production"
  }
}

# Create alerts in the group
resource "last9_alert" "high_error_rate" {
  entity_id    = last9_entity.api_alerts.id
  name         = "High Error Rate"
  query        = "sum(rate(http_errors_total[5m]))"
  greater_than = 100
  bad_minutes  = 5
  total_minutes = 10
  severity     = "breach"
}
```

### With Default Notification Channels

```terraform
resource "last9_entity" "api_alerts" {
  name         = "api-service"
  type         = "service"
  entity_class = "alert-manager"
  external_ref = "api-service-prod"
  ui_readonly  = true

  # Default notification channels for all alerts in this group
  notification_channels = ["slack-platform-alerts", "pagerduty-oncall"]
}
```

## Schema

### Required

- `name` (String) Alert group name.
- `type` (String) Type (e.g., `service`, `component`).
- `entity_class` (String) Entity classification. Must be set to `"alert-manager"` for alert groups to ensure visibility in Alert Studio UI.
- `external_ref` (String) Unique identifier slug for this alert group.

### Optional
- `description` (String) Description of the alert group.
- `data_source` (String) Metrics data source name.
- `data_source_id` (String) Metrics data source ID.
- `namespace` (String) Namespace.
- `team` (String) Owning team.
- `tier` (String) Tier (e.g., `critical`, `high`, `medium`, `low`).
- `workspace` (String) Workspace.
- `labels` (Map of String) Key-value labels for grouping and filtering.
- `notification_channels` (List of String) Default notification channel IDs/names for alerts in this group.
- `ui_readonly` (Boolean) When `true`, prevents edits via UI. Recommended for IaC-managed resources. Default: `false`.

### Read-Only

- `id` (String) Alert group ID. Use this as `entity_id` when creating alerts.

## Import

Import using the alert group ID:

```shell
terraform import last9_entity.example <id>
```
