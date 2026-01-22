---
page_title: "last9_notification_channel Resource - Last9"
subcategory: ""
description: |-
  Manages a Last9 notification channel for alert routing.
---

# last9_notification_channel (Resource)

Manages a Last9 notification channel. Notification channels define where alerts are sent (Slack, PagerDuty, webhooks, etc.).

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
  destination   = "https://events.pagerduty.com/v2/enqueue"
  send_resolved = true
}
```

### Webhook Channel

```terraform
resource "last9_notification_channel" "webhook" {
  name          = "Custom Webhook"
  type          = "webhook"
  destination   = "https://api.example.com/alerts"
  send_resolved = false
}
```

## Schema

### Required

- `name` (String) Name of the notification channel.
- `type` (String) Type of notification channel (e.g., "slack", "pagerduty", "webhook").
- `destination` (String) Destination URL or endpoint for notifications.

### Optional

- `send_resolved` (Boolean) Whether to send notifications when alerts are resolved. Default: `true`.

### Read-Only

- `id` (Number) The ID of the notification channel.

## Import

Notification channels can be imported using the channel ID:

```shell
terraform import last9_notification_channel.example <channel_id>
```
