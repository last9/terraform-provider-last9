---
page_title: "last9_macro Resource - Last9"
subcategory: ""
description: |-
  Manages Last9 cluster macros for reusable PromQL query templates.
---

# last9_macro (Resource)

Manages Last9 cluster macros. Macros are reusable PromQL query templates for defining SLOs and indicators across services.

For more information about macro syntax and usage, see the [PromQL Macros documentation](https://last9.io/docs/promql-macros/).

~> **Note** Changes to macros take up to 5 minutes to propagate across the system.

## Example Usage

### Basic Macro Definitions

```terraform
resource "last9_macro" "cluster_macros" {
  cluster_id = var.cluster_id

  body = <<-EOT
    function availability(metric, window, service) {
      let exclude = "/metrics"
      let good_codes = "2.*"
      let all_codes = "2.*|5.*"

      return sum(rate(metric{handler!~exclude, method="GET", service=service, code=~good_codes}[window])) / sum(rate(metric{handler!~exclude, method="GET", service=service, code=~all_codes}[window]))
    }

    function error_rate(service, window) {
      return sum(rate(http_requests_total{service=service, status=~"5.."}[window])) / sum(rate(http_requests_total{service=service}[window])) * 100
    }

    function latency_p99(service, window) {
      return histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket{service=service}[window])) by (le))
    }
  EOT
}
```

### Using Macros in Alerts

Once defined, macros can be used in alert queries:

```terraform
resource "last9_alert" "availability_alert" {
  entity_id    = last9_entity.api_alerts.id
  name         = "Low Availability"
  query        = "availability(http_requests_total, 5m, api-service)"
  less_than    = 0.995
  bad_minutes  = 5
  total_minutes = 10
  severity     = "breach"
}
```

## Macro Syntax

Macros use a function-based syntax:

```
function macro_name(param1, param2) {
  let variable = "value"
  return promql_expression
}
```

**Components:**
- `function` - Declares the macro with parameters
- `let` - Defines local variables
- `return` - The PromQL expression to execute

**Invocation:** `macro_name(arg1, arg2)`

### Variable Naming Caveat

~> **Warning** Avoid naming macro parameters the same as labels used in `by` clauses. The macro engine performs text substitution, which can cause unintended replacements.

**Problem:**
```
function rate_by_method(metric, method) {
  return sum(rate(metric[5m])) by (method)  # 'method' param replaces 'by (method)'!
}
```

**Solution:** Rename the parameter:
```
function rate_by_method(metric, filter_method) {
  return sum(rate(metric{method=filter_method}[5m])) by (method)
}
```

## Schema

### Required

- `cluster_id` (String) ID of the cluster to create macros for.
- `body` (String) Macro definitions using function syntax.

### Read-Only

- `id` (String) The ID of the macro resource.
- `created_at` (String) Creation timestamp.
- `updated_at` (String) Last update timestamp.

## Import

Import macros using the cluster ID:

```shell
terraform import last9_macro.example <cluster_id>
```
