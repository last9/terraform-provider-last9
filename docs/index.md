---
page_title: "Last9 Provider"
description: |-
  The Last9 provider is used to manage Last9 resources including entities, alerts, log management rules, and scheduled search alerts.
---

# Last9 Provider

The Last9 provider enables Terraform to manage [Last9](https://last9.io) resources for observability and monitoring.

## Features

- **Entities**: Manage services and components with KPIs
- **Alerts**: Configure alert rules with static thresholds or expressions
- **Notification Channels**: Manage alert notification destinations
- **Drop Rules**: Configure log drop rules for filtering and cost optimization
- **Forward Rules**: Set up log forwarding to external destinations
- **Scheduled Search Alerts**: Create log-based alerts with custom queries
- **Macros**: Manage cluster-level macros for query templating
- **Policies**: Define and enforce control plane rules

## Example Usage

```terraform
terraform {
  required_providers {
    last9 = {
      source  = "last9/last9"
      version = "~> 1.0"
    }
  }
}

provider "last9" {
  refresh_token = var.last9_refresh_token
  org           = var.last9_org
  api_base_url  = var.last9_api_base_url
}

# Create an entity
resource "last9_entity" "api_service" {
  name         = "api-service"
  type         = "service"
  external_ref = "api-service-prod"
  description  = "Production API Service"
}

# Create an alert
resource "last9_alert" "high_error_rate" {
  entity_id     = last9_entity.api_service.id
  name          = "High Error Rate"
  description   = "Alert when error rate exceeds threshold"
  query         = "sum(rate(http_errors_total[5m]))"
  greater_than  = 100
  bad_minutes   = 5
  total_minutes = 10
  severity      = "breach"
}
```

## Authentication

The provider supports two authentication methods:

### Refresh Tokens (Recommended)

```terraform
provider "last9" {
  refresh_token = var.last9_refresh_token  # or LAST9_REFRESH_TOKEN env var
  org           = var.last9_org            # or LAST9_ORG env var
  api_base_url  = var.last9_api_base_url   # or LAST9_API_BASE_URL env var
}
```

### Direct Access Tokens (Legacy)

```terraform
provider "last9" {
  api_token    = var.last9_api_token      # or LAST9_API_TOKEN env var
  org          = var.last9_org            # or LAST9_ORG env var
  api_base_url = var.last9_api_base_url   # or LAST9_API_BASE_URL env var
}
```

## Schema

### Required

- `api_base_url` (String) Last9 API base URL. Can be set via `LAST9_API_BASE_URL` environment variable.
- `org` (String) Last9 organization slug. Can be set via `LAST9_ORG` environment variable.

### Optional

- `refresh_token` (String, Sensitive) Last9 refresh token for authentication. Can be set via `LAST9_REFRESH_TOKEN` environment variable. Recommended over `api_token`.
- `api_token` (String, Sensitive) Last9 API access token. Can be set via `LAST9_API_TOKEN` environment variable. Legacy method.

~> **Note** Either `refresh_token` or `api_token` must be provided. Refresh tokens are recommended as they automatically handle token refresh.
