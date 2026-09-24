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

### Notification Channels

```terraform
resource "last9_entity" "api_alerts" {
  name         = "api-service"
  type         = "service"
  entity_class = "alert-manager"
  external_ref = "api-service-prod"
  ui_readonly  = true

  # Notification channels are bound per severity. Every last9_alert in this
  # group at a given severity shares the exact same bindings — the Last9 API
  # has no per-alert-rule notification setting, only per-(entity, severity).
  notification_channels {
    severity = "breach"
    channels = ["slack-platform-alerts", "pagerduty-oncall"]
  }

  notification_channels {
    severity = "threat"
    channels = ["slack-platform-alerts"]
  }
}
```

~> **Note** Manage notification channels here, on the alert group, not on individual `last9_alert` resources. This matches the Last9 UI, which only lets you edit notification channels at the alert-group level ("Inherited from the alert group"). `last9_alert.notification_channels` is deprecated and a no-op — see its own docs for why two resources can't safely manage the same entity/severity.

~> **Note** A `notification_channels` block only reconciles the severity it names, and declaring one makes this resource **fully authoritative** for that severity's bindings — like any other Terraform-managed list. It will attach every channel you list, and detach anything else bound at that severity (added via the UI, `last9_alert.notification_channels`, or any other means) on the next apply. If a severity's channels are managed elsewhere, simply omit that severity's block here entirely — this resource never touches, reports on, or reconciles a severity it has no block for. To manage a severity but intentionally keep it empty, add a block with `channels = []`. `terraform import` seeds a block for every severity that currently has a live binding, so an imported entity starts out managing everything already bound to it.

### Notify-Once (Suppress Repeat Notifications)

```terraform
resource "last9_entity" "api_alerts" {
  name         = "api-service"
  type         = "service"
  entity_class = "alert-manager"
  external_ref = "api-service-prod"

  # Send only the first firing notification and the resolved notification.
  # No re-notifications while the alert stays firing.
  renotify_enabled = false
}
```

### Custom Repeat Interval with Occurrence Cap

```terraform
resource "last9_entity" "api_alerts" {
  name         = "api-service"
  type         = "service"
  entity_class = "alert-manager"
  external_ref = "api-service-prod"

  renotify_enabled          = true
  renotify_interval_seconds = 1800  # re-notify every 30 minutes
  renotify_occurrences      = 3     # stop after 3 repeats; use -1 for unlimited
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
- `notification_channels` (Block List) Notification channel bindings for this alert group, one block per severity. See [Notification Channels](#notification-channels) above. Each block:
  - `severity` (String, Required) `breach` or `threat`.
  - `channels` (List of String, Required) Notification channel IDs or names to bind at this severity.
- `ui_readonly` (Boolean) When `true`, prevents edits via UI. Recommended for IaC-managed resources. Default: `false`.
- `renotify_enabled` (Boolean) Controls repeat notifications while an alert stays firing. `false` = notify-once (first + resolved only). `true` = re-notify per `renotify_interval_seconds`. Omit to inherit the tenant default (re-notify enabled, 1 hour interval).
- `renotify_interval_seconds` (Number) Seconds between repeat notifications while firing. Must be a positive integer (≥ 1). Ignored when `renotify_enabled` is `false`. Omit to inherit the tenant default.
- `renotify_occurrences` (Number) Maximum number of repeat notifications per firing episode. `-1` = unlimited. Must be `-1` or ≥ 1. Omit to inherit the tenant default.

-> **Note** To reset all renotify overrides back to tenant defaults, remove all three `renotify_*` fields from your config and run `terraform apply`. Removing individual fields without removing all three may not clear the remaining fields from the server.

### Read-Only

- `id` (String) Alert group ID. Use this as `entity_id` when creating alerts.

## Import

Import using the alert group ID:

```shell
terraform import last9_entity.example <id>
```
