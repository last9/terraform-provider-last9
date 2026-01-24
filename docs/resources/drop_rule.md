---
page_title: "last9_drop_rule Resource - Last9"
subcategory: ""
description: |-
  Manages a Last9 drop rule for filtering telemetry and cost optimization.
---

# last9_drop_rule (Resource)

Manages a Last9 drop rule. Drop rules filter out telemetry (logs, traces, or metrics) that matches specified criteria, helping reduce storage costs and noise. Rules are applied at the ingestion layer without requiring code changes or redeployment.

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
    key      = "attributes[\"level\"]"
    value    = "debug"
    operator = "equals"
  }

  action {
    name = "drop-matching"
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
    key         = "resource.attributes[\"environment\"]"
    value       = "test"
    operator    = "equals"
    conjunction = "AND"
  }

  filters {
    key      = "attributes[\"level\"]"
    value    = "info"
    operator = "equals"
  }

  action {
    name = "drop-matching"
  }
}
```

### Drop Traces by Service Name (Regex)

```terraform
resource "last9_drop_rule" "drop_test_traces" {
  region    = "ap-south-1"
  name      = "drop-test-service-traces"
  telemetry = "traces"

  filters {
    key      = "resource.attributes[\"service.name\"]"
    value    = "test-.*"
    operator = "like"
  }

  action {
    name = "drop-matching"
  }
}
```

### Drop Metrics by Name

```terraform
resource "last9_drop_rule" "drop_high_cardinality_metric" {
  region    = "ap-south-1"
  name      = "drop-high-cardinality-metric"
  telemetry = "metrics"

  filters {
    key      = "name"
    value    = "prometheus_http_requests_total"
    operator = "equals"
  }

  action {
    name = "drop-matching"
  }
}
```

## Schema

### Required

- `region` (String) Region for the drop rule (e.g., "ap-south-1", "us-west-2").
- `name` (String) Name of the drop rule.
- `telemetry` (String) Telemetry type. Valid values: `logs`, `traces`, `metrics`.
- `filters` (Block List, Min: 1) Filter conditions. See [Filters](#filters) below.
- `action` (Block List, Max: 1) Action to take. See [Action](#action) below.

### Optional

- `cluster_id` (String) Cluster ID. If not provided, uses the default cluster for the region.

### Read-Only

- `id` (String) The ID of the drop rule in format `region:cluster_id:name`.

### Filters

The `filters` block supports:

- `key` (String, Required) The field key to filter on.
  - For **logs/traces**: Must use `attributes["key"]` or `resource.attributes["key"]` format.
  - For **metrics**: Use `"name"` to filter by metric name.
- `value` (String, Required) The value to match.
  - For **logs/traces**: The attribute value to match.
  - For **metrics**: The metric name.
- `operator` (String, Required) Comparison operator. Valid values: `equals`, `not_equals`, `like` (regex).
- `conjunction` (String, Optional) Logical conjunction for combining multiple filters. Valid value: `AND`.

### Action

The `action` block supports:

- `name` (String, Required) Action name. Must be `drop-matching` for drop rules.

## Import

Drop rules can be imported using the format `region:cluster_id:name`:

```shell
terraform import last9_drop_rule.example ap-south-1:cluster-123:drop-debug-logs
```
