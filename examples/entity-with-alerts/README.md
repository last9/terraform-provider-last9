# Entity with Alerts Example

This example demonstrates the new entity resource and enhanced alert features added to the Last9 Terraform provider, bringing it to feature parity with the Last9 IaC tool.

## New Features Demonstrated

### Entity Resource (`last9_entity`)

Complete entity/alert group management with all fields from Last9 IaC:

#### Required Fields
- `name` - Entity name
- `type` - Entity type (service, service_alert_manager, etc.)
- `external_ref` - Unique slug identifier (immutable)

#### Metadata Fields
- `description` - Entity description
- `data_source` / `data_source_id` - Data source configuration
- `namespace` - Entity namespace
- `team` - Owning team
- `tier` - Entity tier (critical, high, medium, low)
- `workspace` - Workspace assignment
- `tags` - Array of tags for categorization
- `entity_class` - Entity classification (e.g., alert-manager)

#### IaC-Specific Features

**Labels** - Group-level labels inherited across all indicators:
```hcl
labels = {
  environment = "production"
  service     = "api"
  team        = "platform"
}
```

**UI Readonly** - Prevent UI edits to avoid configuration drift:
```hcl
ui_readonly = true
```

**Adhoc Filters** - Common PromQL label filters applied across all indicators:
```hcl
adhoc_filter {
  data_source = "cluster-name"
  labels = {
    cluster = "production"
    job     = "api-server"
  }
}
```

**Indicators** - Define metrics for the entity:
```hcl
indicators {
  name  = "up"
  query = "up{job=\"api-server\"}"
  unit  = "bool"
}
```

**Links** - Related resource links:
```hcl
links {
  name = "Runbook"
  url  = "https://wiki.example.com/runbooks/service"
}
```

**Notification Channels** - Entity-level alert routing:
```hcl
notification_channels = [
  "slack-platform-alerts",
  "pagerduty-oncall"
]
```

### Enhanced Alert Resource (`last9_alert`)

**Notification Channels** - Alert-specific notification routing:
```hcl
notification_channels = [
  data.last9_notification_destination.slack.id,
  data.last9_notification_destination.pagerduty.id
]
```

### Entity Data Source

Query entities by ID or external reference:
```hcl
data "last9_entity" "my_service" {
  external_ref = "my-service-v1"
}
```

## Usage

1. Set your Last9 credentials:
```bash
export TF_VAR_last9_refresh_token="your-refresh-token"
export TF_VAR_last9_org="your-org-slug"
```

2. Initialize Terraform:
```bash
terraform init
```

3. Review the plan:
```bash
terraform plan
```

4. Apply the configuration:
```bash
terraform apply
```

## What Gets Created

1. **Entity** (`production-api-service`):
   - Complete metadata (namespace, team, tier, tags, labels)
   - 4 indicators (up, error_rate, latency_p95, availability)
   - 3 related links (runbook, dashboard, repository)
   - 2 notification channels
   - UI readonly mode enabled
   - Adhoc filters for common label filtering

2. **Loss of Signal Alert**:
   - Monitors the `up` metric
   - Triggers when service stops reporting (< 1 for 3/5 minutes)
   - Comprehensive annotations for runbook integration
   - Routes to both Slack and PagerDuty

3. **High Error Rate Alert**:
   - Static threshold alert (> 5% error rate)
   - Routes to Slack

4. **Low Availability Alert**:
   - Expression-based alert using `low_spike()` function
   - Routes to PagerDuty for critical issues

## Field Mapping from Last9 IaC

All fields from the Last9 IaC tool comparison are now supported:

| Field | Status | Notes |
|-------|--------|-------|
| `name` | ✅ | Entity/alert name |
| `type` | ✅ | Entity type |
| `external_ref` | ✅ | Unique slug identifier |
| `description` | ✅ | Entity/alert description |
| `data_source` | ✅ | Data source name |
| `data_source_id` | ✅ | Data source ID |
| `namespace` | ✅ | Entity namespace |
| `team` | ✅ | Owning team |
| `tier` | ✅ | Entity tier |
| `workspace` | ✅ | Workspace |
| `tags` | ✅ | Array of tags |
| `labels` | ✅ | Group labels for indicators |
| `entity_class` | ✅ | Entity classification |
| `ui_readonly` | ✅ | Disable UI edits |
| `adhoc_filter` | ✅ | Common rule filters |
| `indicators` | ✅ | Metric definitions |
| `links` | ✅ | Related links |
| `notification_channels` | ✅ | Alert routing |

## Notes

- The `external_ref` field is immutable (ForceNew=true) - changing it will recreate the entity
- Labels defined at the entity level are inherited by all indicators during alert evaluation
- The `ui_readonly` flag prevents accidental UI modifications, maintaining IaC as the source of truth
- Notification channels can be defined at both entity and alert levels for flexible routing
