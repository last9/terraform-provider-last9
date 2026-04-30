# Terraform Provider for Last9

Manage Last9 alerts, notification channels, log pipelines, and remapping rules as code. Stop clicking through dashboards to configure observability infrastructure.

## Quick Start

```hcl
terraform {
  required_providers {
    last9 = {
      source  = "last9/last9"
      version = "~> 0.2"
    }
  }
}

provider "last9" {
  refresh_token = var.last9_refresh_token
  org           = "your-org-slug"
  api_base_url  = "https://app.last9.io"
}
```

Get your refresh token from the Last9 dashboard under API tokens. Use environment variables so you don't accidentally commit credentials:

```bash
export LAST9_REFRESH_TOKEN=your-token
export LAST9_ORG=your-org-slug
export LAST9_API_BASE_URL=https://app.last9.io
```

## What You Can Manage

**Alerting**
- `last9_entity` — Alert groups that organize your metrics
- `last9_alert` — Metric-based alert rules with thresholds
- `last9_scheduled_search_alert` — Log-based alerts on search queries

**Notifications**
- `last9_notification_channel` — Slack, PagerDuty, webhooks, email

**Log Pipeline**
- `last9_drop_rule` — Drop logs before they're stored (cuts costs)
- `last9_forward_rule` — Route logs to external destinations
- `last9_remapping_rule` — Extract fields and map attributes from logs and traces

**Data Sources**
- `last9_entity` — Query existing alert groups
- `last9_notification_destination` — Query notification destinations

## Examples

### Alert on high error rate

```hcl
resource "last9_alert" "high_error_rate" {
  entity_id   = last9_entity.api.id
  name        = "High Error Rate"
  description = "Error rate exceeded threshold"
  indicator   = "error_rate"

  greater_than  = 100
  bad_minutes   = 5
  total_minutes = 10

  severity = "breach"

  properties {
    runbook_url = "https://wiki.example.com/runbooks/high-error-rate"
  }
}
```

### Drop debug logs to save money

```hcl
resource "last9_drop_rule" "debug_logs" {
  region    = "ap-south-1"
  name      = "drop-debug-logs"
  telemetry = "logs"

  filters {
    key      = "attributes[\"severity\"]"
    value    = "debug"
    operator = "equals"
  }
}
```

### Extract fields from log lines

```hcl
resource "last9_remapping_rule" "nginx_access" {
  region            = "ap-south-1"
  type              = "logs_extract"
  name              = "nginx-access-log-parser"
  extract_type      = "pattern"
  action            = "upsert"
  remap_keys        = ["(?P<method>\\w+) (?P<path>/[^\\s]*) HTTP/(?P<version>[\\d.]+)"]
  target_attributes = "log_attributes"
}
```

Three remapping types: `logs_extract` pulls fields out of log lines, `logs_map` promotes an existing attribute to a standard field, `traces_map` does the same for spans.

### Map a log attribute to service name

```hcl
resource "last9_remapping_rule" "service_name" {
  region            = "ap-south-1"
  type              = "logs_map"
  name              = "map-svc-to-service"
  remap_keys        = ["svc", "app_name"]
  target_attributes = "service"
}
```

Valid targets for `logs_map`: `service`, `severity`, `resource_deployment.environment`.  
Valid target for `traces_map`: `service`.

### Forward critical logs to an external system

```hcl
resource "last9_forward_rule" "critical_to_siem" {
  region      = "ap-south-1"
  name        = "forward-critical-to-siem"
  telemetry   = "logs"
  destination = "https://siem.example.com/ingest"

  filters {
    key      = "attributes[\"severity\"]"
    value    = "critical"
    operator = "equals"
  }
}
```

## Auth

Use `refresh_token` — it automatically manages token rotation and is what the Last9 dashboard uses. The `api_token` option exists for legacy integrations but you probably don't need it.

For operations that delete resources, supply a separate delete-scoped token:

```hcl
provider "last9" {
  refresh_token        = var.last9_refresh_token
  delete_refresh_token = var.last9_delete_refresh_token
  org                  = "your-org-slug"
  api_base_url         = "https://app.last9.io"
}
```

## Importing Existing Resources

All resources support import. The import ID format is documented per resource:

```bash
# Alert
terraform import last9_alert.example entity_slug/alert_id

# Remapping rule
terraform import last9_remapping_rule.example region:type:id

# Drop rule
terraform import last9_drop_rule.example region:id
```

## Building from Source

```bash
git clone https://github.com/last9/terraform-provider-last9
cd terraform-provider-last9
go build -o terraform-provider-last9
```

Requires Go 1.21+.

## Running Tests

Unit tests require no credentials:

```bash
go test ./internal/provider -run '^Test[^A]'
```

Acceptance tests hit the real API:

```bash
export LAST9_API_TOKEN=your-access-token
export LAST9_DELETE_TOKEN=your-delete-token
export LAST9_ORG=your-org-slug
export LAST9_API_BASE_URL=https://app.last9.io
export LAST9_TEST_REGION=ap-south-1
TF_ACC=1 go test ./internal/provider -v -run '^TestAcc' -timeout 300s
```

## License

Mozilla Public License 2.0 — see [LICENSE](LICENSE).
