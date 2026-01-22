---
page_title: "last9_notification_destination Data Source - Last9"
subcategory: ""
description: |-
  Retrieves information about a Last9 notification destination.
---

# last9_notification_destination (Data Source)

Retrieves information about an existing Last9 notification destination by ID or name. Use this to reference existing notification channels when creating alerts.

## Example Usage

### By Name

```terraform
data "last9_notification_destination" "slack" {
  name = "Platform Alerts Slack"
}

resource "last9_alert" "example" {
  # ... other configuration ...
  notification_channels = [data.last9_notification_destination.slack.id]
}
```

### By ID

```terraform
data "last9_notification_destination" "example" {
  id = 123
}

output "destination_type" {
  value = data.last9_notification_destination.example.type
}
```

## Schema

### Optional

- `id` (Number) The ID of the notification destination to retrieve.
- `name` (String) The name of the notification destination to retrieve.

~> **Note** Either `id` or `name` must be provided.

### Read-Only

- `type` (String) Type of notification destination (e.g., "slack", "pagerduty", "webhook").
- `destination` (String) Destination URL or endpoint.
- `global` (Boolean) Whether this is a global notification destination.
- `send_resolved` (Boolean) Whether resolved notifications are sent.
- `in_use` (Boolean) Whether this destination is currently in use.
