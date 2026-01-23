---
page_title: "last9_drop_rule Resource - Last9"
subcategory: ""
description: |-
  Manages a Last9 log drop rule for filtering and cost optimization.
---

# last9_drop_rule (Resource)

Manages a Last9 drop rule. Drop rules filter out telemetry that matches specified criteria, helping reduce storage costs and noise. Rules are applied at the ingestion layer without requiring code changes or redeployment.

For more information, see the [Drop Rules documentation](https://last9.io/docs/control-plane-drop/).

~> **Warning** Data matching drop rules is permanently discarded and cannot be recovered. Use the "View in Dashboard" preview feature in the Last9 UI to verify filters before saving.

## Example Usage

### Drop Debug Logs

```terraform
resource "last9_drop_rule" "drop_debug" {
  region    = "ap-south-1"
  name      = "drop-debug-logs"
  telemetry = "logs"

  filters {
    key         = "SeverityText"
    value       = "DEBUG"
    operator    = "equals"
    conjunction = "and"
  }

  action {
    drop = true
  }
}
```

### Drop Logs by Multiple Criteria

```terraform
resource "last9_drop_rule" "drop_test_logs" {
  region    = "ap-south-1"
  name      = "drop-test-environment-logs"
  telemetry = "logs"

  filters {
    key         = "attributes[\"environment\"]"
    value       = "test"
    operator    = "equals"
    conjunction = "and"
  }

  filters {
    key         = "SeverityText"
    value       = "INFO"
    operator    = "equals"
    conjunction = "and"
  }

  action {
    drop = true
  }
}
```

## Schema

### Required

- `region` (String) Region for the drop rule (e.g., "ap-south-1", "us-west-2").
- `name` (String) Name of the drop rule.
- `telemetry` (String) Telemetry type. Valid values: `logs`, `traces`, `metrics`.
- `filters` (Block List, Min: 1) Filter conditions. See [Filters](#filters) below.
- `action` (Block) Action to take. See [Action](#action) below.

### Optional

- `cluster_id` (String) Cluster ID. If not provided, uses the default cluster for the region.

### Read-Only

- `id` (String) The ID of the drop rule.

### Filters

The `filters` block supports:

- `key` (String, Required) The field key to filter on (e.g., "SeverityText", "attributes[\"service\"]").
- `value` (String, Required) The value to match.
- `operator` (String, Required) Comparison operator. Valid values: `equals`, `not_equals`.
- `conjunction` (String, Required) Logical conjunction. Valid values: `and`, `or`.

### Action

The `action` block supports:

- `drop` (Boolean) Whether to drop matching logs. Default: `true`.

## Import

Drop rules can be imported using the format `region:cluster_id:name`:

```shell
terraform import last9_drop_rule.example ap-south-1:cluster-123:drop-debug-logs
```
