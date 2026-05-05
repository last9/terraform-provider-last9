---
page_title: "Last9 Provider"
description: |-
  The Last9 provider is used to manage Last9 resources including alerts, notification channels, log management rules, and scheduled search alerts.
---

# Last9 Provider

The Last9 provider enables Terraform to manage [Last9](https://last9.io) resources for observability and monitoring.

## Documentation Links

- [Last9 API Getting Started](https://last9.io/docs/getting-started-with-api/) - Authentication and token management
- [Control Plane](https://last9.io/docs/control-plane/) - Ingestion, storage, query, and analytics
- [Alerting Overview](https://last9.io/docs/alerting-overview/) - Alert groups, rules, and indicators
- [Notification Channels](https://last9.io/docs/notification-channels/) - Slack, PagerDuty, email, webhooks such as Jira, Flock, Telegram and more.

## Features

- **Alerts**: Configure alerting rules (metric-based with thresholds/expressions, or log-based with scheduled searches)
- **Notification Channels**: Manage alert destinations (Slack, PagerDuty, email, webhooks)
- **Drop Rules**: Filter and drop logs for cost optimization at Last9 Control Plane
- **Forward Rules**: Forward logs to external destinations such as S3 bucket
- **Remapping Rules**: Extract fields from logs and map attributes to standard fields (logs and traces)
- **Dashboards**: Define custom dashboards with panels, queries, variables, and layout grid
- **Macros**: PromQL query templates

## Alerting Lifecycle

The provider supports two types of alerting:

### Metric-Based Alerts

For monitoring metrics and KPIs with thresholds:

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────────────┐
│  last9_entity   │────▶│   last9_alert   │────▶│ last9_notification_     │
│  (alert group)  │     │ (threshold/expr)│     │ channel (Slack/PD/etc)  │
└─────────────────┘     └─────────────────┘     └─────────────────────────┘
```

### Log-Based Alerts

For alerting on log patterns and aggregations:

```
┌───────────────────────────┐     ┌─────────────────────────┐
│ last9_scheduled_search_   │────▶│  notification_          │
│ alert (query + threshold) │     │  destinations           │
└───────────────────────────┘     └─────────────────────────┘
```

## Example Usage

### Metric Alert with Notification

```terraform
terraform {
  required_providers {
    last9 = {
      source  = "last9/last9"
      version = "~> 0.2"
    }
  }
}

provider "last9" {
  refresh_token        = var.last9_refresh_token
  delete_refresh_token = var.last9_delete_refresh_token
  org                  = var.last9_org
  api_base_url         = var.last9_api_base_url
}

# Create an alert group
resource "last9_entity" "api_alerts" {
  name         = "api-service"
  type         = "service"
  external_ref = "api-service-prod"
  description  = "Alerts for Production API Service"
  ui_readonly  = true  # Prevent UI edits, manage via Terraform only
}

# Create a notification channel
resource "last9_notification_channel" "slack_alerts" {
  name         = "platform-alerts"
  type         = "slack"
  destination  = "https://hooks.slack.com/services/xxx/yyy/zzz"
  send_resolved = true
}

# Create an alert in the group
resource "last9_alert" "high_error_rate" {
  entity_id     = last9_entity.api_alerts.id
  name          = "High Error Rate"
  description   = "Alert when error rate exceeds 100/min for 5 minutes"
  query         = "sum(rate(http_errors_total[5m]))"
  greater_than  = 100
  bad_minutes   = 5
  total_minutes = 10
  severity      = "breach"

  notification_channels = [last9_notification_channel.slack_alerts.id]

  properties {
    runbook_url = "https://wiki.example.com/runbooks/high-error-rate"
    annotations = {
      team     = "platform"
      priority = "high"
    }
  }
}
```

### Log-Based Scheduled Search Alert

```terraform
resource "last9_scheduled_search_alert" "error_spike" {
  region         = "us-west-2"
  rule_name      = "Error Log Spike"
  query_type     = "pipeline"
  physical_index = "default"

  telemetry        = "logs"
  search_frequency = 5

  query = jsonencode([
    { "match": { "level": "error" } }
  ])

  threshold_value    = 100
  threshold_operator = ">"

  notification_destination_ids = [12345]
}
```

## Authentication

The provider supports two authentication methods. For detailed information about API authentication and token management, see the [Last9 API Getting Started Guide](https://last9.io/docs/getting-started-with-api/).

### Token Types

Last9 uses three token scopes with distinct permissions:

- **Read/Write Tokens**: For creating, reading, and modifying resources
- **Delete Tokens**: Required for delete operations (e.g., `terraform destroy`)

Access tokens expire after **24 hours**. When using refresh tokens, the provider automatically handles token renewal.

### Refresh Tokens (Recommended)

```terraform
provider "last9" {
  refresh_token        = var.last9_refresh_token         # or LAST9_REFRESH_TOKEN env var
  delete_refresh_token = var.last9_delete_refresh_token  # or LAST9_DELETE_REFRESH_TOKEN env var
  org                  = var.last9_org                   # or LAST9_ORG env var
  api_base_url         = var.last9_api_base_url          # or LAST9_API_BASE_URL env var
}
```

### Direct Access Tokens (Legacy)

```terraform
provider "last9" {
  api_token    = var.last9_api_token       # or LAST9_API_TOKEN env var
  delete_token = var.last9_delete_token    # or LAST9_DELETE_TOKEN env var
  org          = var.last9_org             # or LAST9_ORG env var
  api_base_url = var.last9_api_base_url    # or LAST9_API_BASE_URL env var
}
```

~> **Note** Direct access tokens expire after 24 hours and must be manually refreshed. Refresh tokens are recommended for automated workflows.

## Schema

### Required

- `api_base_url` (String) Last9 API base URL. Can be set via `LAST9_API_BASE_URL` environment variable.
- `org` (String) Last9 organization slug. Can be set via `LAST9_ORG` environment variable.

### Optional

- `refresh_token` (String, Sensitive) Last9 refresh token for read/write operations. Can be set via `LAST9_REFRESH_TOKEN` environment variable. Recommended over `api_token`.
- `api_token` (String, Sensitive) Last9 API access token for read/write operations. Can be set via `LAST9_API_TOKEN` environment variable. Legacy method - tokens expire after 24 hours.
- `delete_refresh_token` (String, Sensitive) Last9 refresh token for delete operations. Can be set via `LAST9_DELETE_REFRESH_TOKEN` environment variable. Required for `terraform destroy`.
- `delete_token` (String, Sensitive) Last9 API token with delete scope. Can be set via `LAST9_DELETE_TOKEN` environment variable. Legacy method.

~> **Note** Either `refresh_token` or `api_token` must be provided for read/write operations. For delete operations (e.g., `terraform destroy`), either `delete_refresh_token` or `delete_token` is required. See the [Last9 API documentation](https://last9.io/docs/getting-started-with-api/) for token generation.
