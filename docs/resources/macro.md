---
page_title: "last9_macro Resource - Last9"
subcategory: ""
description: |-
  Manages a Last9 cluster macro for query templating.
---

# last9_macro (Resource)

Manages a Last9 cluster macro. Macros provide reusable query templates that can be referenced across dashboards and alerts.

## Example Usage

```terraform
resource "last9_macro" "cluster_macros" {
  cluster_id = "cluster-123"

  body = jsonencode({
    macros = {
      "service_name" = "$service"
      "environment"  = "$env"
      "error_query"  = "sum(rate(http_requests_total{service=\"$service\", status=~\"5..\"}[5m]))"
      "rate_query"   = "sum(rate(http_requests_total{service=\"$service\"}[5m]))"
    }
  })
}
```

## Schema

### Required

- `cluster_id` (String) ID of the cluster to create macros for.
- `body` (String) JSON-encoded macro definitions.

### Read-Only

- `id` (String) The ID of the macro resource.

## Import

Macros can be imported using the cluster ID:

```shell
terraform import last9_macro.example <cluster_id>
```
