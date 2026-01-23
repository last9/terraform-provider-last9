---
page_title: "last9_macro Resource - Last9"
subcategory: ""
description: |-
  Manages Last9 macros for reusable PromQL query templates.
---

# last9_macro (Resource)

Macros are reusable PromQL query templates for defining SLOs and indicators across services.

For more information about macro usage, see the [PromQL Macros documentation](https://last9.io/docs/promql-macros/).

~> **Note** Changes to macros take up to 5 minutes to propagate across the system.

## Example Usage

### Basic Macro Definitions

```terraform
resource "last9_macro" "cluster_macros" {
  cluster_id = var.cluster_id

  # The body must be a JSON object containing macro definitions
  body = jsonencode({
    macros = {
      # Availability: successful requests / total requests
      availability = "sum(rate(http_requests_total{status=~\"2..\"}[$window])) / sum(rate(http_requests_total[$window]))"

      # Error rate as percentage
      error_rate = "sum(rate(http_requests_total{status=~\"5..\"}[$window])) / sum(rate(http_requests_total[$window])) * 100"

      # Latency percentiles
      latency_p95 = "histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[$window])) by (le))"
      latency_p99 = "histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[$window])) by (le))"
    }
  })
}
```

### Using Variables in Macros

Macros support variable substitution using `$variable` syntax:

```terraform
resource "last9_macro" "service_macros" {
  cluster_id = var.cluster_id

  body = jsonencode({
    macros = {
      # $service and $window are replaced at query time
      service_errors = "sum(rate(http_requests_total{service=\"$service\", status=~\"5..\"}[$window]))"
      service_requests = "sum(rate(http_requests_total{service=\"$service\"}[$window]))"
    }
  })
}
```

### Using Macros in Alerts

Once defined, macros can be referenced in alert queries:

```terraform
resource "last9_alert" "availability_alert" {
  entity_id     = last9_entity.api_alerts.id
  name          = "Low Availability"
  query         = "availability"  # References the macro defined above
  less_than     = 0.995
  bad_minutes   = 5
  total_minutes = 10
  severity      = "breach"
}
```

## Schema

### Required

- `cluster_id` (String) ID of the cluster to create macros for.
- `body` (String) Macro definitions as a JSON string. Use `jsonencode()` for proper formatting.

### Read-Only

- `id` (String) The ID of the macro resource.
- `created_at` (String) Creation timestamp.
- `updated_at` (String) Last update timestamp.

## Import

Import macros using the cluster ID:

```shell
terraform import last9_macro.example <cluster_id>
```
