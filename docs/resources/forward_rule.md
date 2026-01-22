---
page_title: "last9_forward_rule Resource - Last9"
subcategory: ""
description: |-
  Manages a Last9 log forward rule for routing logs to external destinations.
---

# last9_forward_rule (Resource)

Manages a Last9 log forward rule. Forward rules route logs matching specified criteria to external destinations like SIEM systems, data lakes, or other logging platforms.

## Example Usage

### Forward Critical Logs

```terraform
resource "last9_forward_rule" "forward_critical" {
  region      = "ap-south-1"
  name        = "forward-critical-logs"
  telemetry   = "logs"
  destination = "https://logs.external-system.com/webhook"

  filters {
    key         = "SeverityText"
    value       = "CRITICAL"
    operator    = "equals"
    conjunction = "and"
  }
}
```

### Forward Security Logs to SIEM

```terraform
resource "last9_forward_rule" "security_logs" {
  region      = "ap-south-1"
  name        = "forward-security-logs"
  telemetry   = "logs"
  destination = "https://siem.example.com/api/logs"

  filters {
    key         = "attributes[\"category\"]"
    value       = "security"
    operator    = "equals"
    conjunction = "and"
  }

  filters {
    key         = "SeverityText"
    value       = "WARNING"
    operator    = "not_equals"
    conjunction = "and"
  }
}
```

## Schema

### Required

- `region` (String) Region for the forward rule (e.g., "ap-south-1", "us-west-2").
- `name` (String) Name of the forward rule.
- `telemetry` (String) Telemetry type. Valid values: `logs`, `traces`, `metrics`.
- `destination` (String) Destination URL to forward logs to.
- `filters` (Block List, Min: 1) Filter conditions. See [Filters](#filters) below.

### Optional

- `cluster_id` (String) Cluster ID. If not provided, uses the default cluster for the region.

### Read-Only

- `id` (String) The ID of the forward rule.

### Filters

The `filters` block supports:

- `key` (String, Required) The field key to filter on (e.g., "SeverityText", "attributes[\"service\"]").
- `value` (String, Required) The value to match.
- `operator` (String, Required) Comparison operator. Valid values: `equals`, `not_equals`, `contains`, `not_contains`, `regex`.
- `conjunction` (String, Required) Logical conjunction. Valid values: `and`, `or`.

## Import

Forward rules can be imported using the format `region:cluster_id:name`:

```shell
terraform import last9_forward_rule.example ap-south-1:cluster-123:forward-critical-logs
```
