---
page_title: "last9_notification_channel Resource - Last9"
subcategory: ""
description: |-
  Manages a Last9 notification channel for alert routing.
---

# last9_notification_channel (Resource)

Manages a Last9 notification channel. Notification channels define where alerts are sent.

For more information about setting up notification channels, see the [Notification Channels documentation](https://last9.io/docs/notification-channels/).

## Supported Channel Types

| Type | Description | Destination Format |
|------|-------------|-------------------|
| `slack` | Slack incoming webhook | Webhook URL |
| `pagerduty` | PagerDuty Events API v2 | Integration key |
| `email` | Email notifications | Email address |
| `generic_webhook` | Custom webhook | Endpoint URL |

~> **Note** Email channels are not supported for SLO alerts. Use Slack, PagerDuty, or webhooks for SLO alerting.

## Example Usage

### Slack Channel

```terraform
resource "last9_notification_channel" "slack_alerts" {
  name          = "Platform Alerts Slack"
  type          = "slack"
  destination   = "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXX"
  send_resolved = true
}
```

### PagerDuty Channel

```terraform
resource "last9_notification_channel" "pagerduty" {
  name          = "Platform PagerDuty"
  type          = "pagerduty"
  destination   = "<integration-key>"
  send_resolved = true
}
```

### Generic Webhook

```terraform
resource "last9_notification_channel" "webhook" {
  name          = "Custom Webhook"
  type          = "generic_webhook"
  destination   = "https://api.example.com/alerts"
  send_resolved = false
}
```

### Generic Webhook with Custom Headers

```terraform
resource "last9_notification_channel" "webhook_with_auth" {
  name          = "Authenticated Webhook"
  type          = "generic_webhook"
  destination   = "https://api.example.com/alerts"
  send_resolved = true

  headers = {
    "Authorization" = "Bearer ${var.webhook_token}"
    "X-Custom-Header" = "my-value"
  }
}
```

## Schema

### Required

- `name` (String) Name of the notification channel.
- `type` (String) Channel type: `slack`, `pagerduty`, `email`, or `generic_webhook`.
- `destination` (String) Destination URL, key, or email address depending on type.

### Optional

- `send_resolved` (Boolean) Whether to send notifications when alerts are resolved. Default: `true`.
- `headers` (Map of String) Custom HTTP headers to send with webhook requests. Only applicable for `generic_webhook` type. Useful for authentication tokens or custom metadata.

### Read-Only

- `id` (Number) The ID of the notification channel.
- `global` (Boolean) Whether this is a global (master) channel.
- `in_use` (Boolean) Whether the channel has any attachments.
- `organization_id` (String) Organization ID.
- `created_at` (String) Creation timestamp.
- `updated_at` (String) Last update timestamp.

## Webhook Payload Structure

When using `generic_webhook`, Last9 sends the following JSON payload:

```json
{
  "routing_key": "<webhook-url>",
  "event_action": "trigger",
  "dedup_key": "<unique-key-max-255-chars>",
  "client": "Last9 Dashboard",
  "client_url": "https://app.last9.io/...",
  "payload": {
    "summary": "Alert: High Error Rate on api-service",
    "source": "api-service.prod",
    "severity": "critical",
    "timestamp": "2024-01-15T10:30:00Z",
    "component": "api-service",
    "group": "production",
    "class": "Static Threshold",
    "custom_details": {
      "rule_name": "High Error Rate",
      "telemetry_type": "metrics",
      "query": "sum(rate(http_errors_total[5m]))",
      "evaluation_frequency": "1m",
      "alert_condition": "> 100",
      "metric_name": "http_errors_total"
    }
  },
  "images": [],
  "links": []
}
```

| Field | Description |
|-------|-------------|
| `event_action` | `trigger`, `acknowledge`, or `resolve` |
| `dedup_key` | Correlation key for grouping related events (max 255 chars) |
| `payload.severity` | `critical`, `error`, `warning`, or `info` |
| `payload.timestamp` | ISO 8601 format |

-> **Note** The `send_resolved` option controls whether `resolve` events are sent when alerts clear.

-> **Note** When `headers` are configured, they are included in every HTTP request to the webhook endpoint.

## Import

Notification channels can be imported using the channel ID:

```shell
terraform import last9_notification_channel.example <channel_id>
```
