---
page_title: "last9_policy Resource - Last9"
subcategory: ""
description: |-
  Manages a Last9 control plane policy for governance and compliance.
---

# last9_policy (Resource)

Manages a Last9 control plane policy. Policies define rules and requirements that entities must comply with, such as SLO targets, alert coverage, and naming conventions.

## Example Usage

```terraform
resource "last9_policy" "slo_compliance" {
  name        = "SLO Compliance Policy"
  description = "Ensures all production services meet SLO requirements"

  filters = {
    entity_type = "service"
    tags        = "production"
  }

  rules {
    type = "slo_compliance"
    config = {
      slo_name          = "availability"
      threshold         = "99.9"
      evaluation_window = "30d"
    }
  }

  rules {
    type = "alert_coverage"
    config = {
      min_alerts = "3"
    }
  }

  rules {
    type = "slo_compliance"
    config = {
      slo_name          = "latency"
      threshold         = "200ms"
      evaluation_window = "7d"
    }
  }
}
```

## Schema

### Required

- `name` (String) Name of the policy.
- `rules` (Block List, Min: 1) Policy rules. See [Rules](#rules) below.

### Optional

- `description` (String) Description of the policy.
- `filters` (Map of String) Filters to determine which entities the policy applies to.

### Read-Only

- `id` (String) The ID of the policy.

### Rules

The `rules` block supports:

- `type` (String, Required) Rule type (e.g., "slo_compliance", "alert_coverage").
- `config` (Map of String, Required) Rule configuration as key-value pairs.

## Import

Policies can be imported using the policy ID:

```shell
terraform import last9_policy.example <policy_id>
```
