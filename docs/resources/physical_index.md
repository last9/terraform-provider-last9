---
page_title: "last9_physical_index Resource - Last9"
subcategory: ""
description: |-
  Manages an OTel physical index for logs.
---

# last9_physical_index (Resource)

Configures a physical index partition for logs via `/otel_settings/physical_index`.

## Example Usage

```terraform
resource "last9_physical_index" "payments" {
  region      = "ap-south-1"
  name        = "payments_logs"
  telemetry   = "logs"
  description = "Payments service logs"
  retain      = true

  filters {
    key      = "attributes[\"service\"]"
    value    = "payments"
    operator = "equals"
  }
}
```

## Import

```shell
terraform import last9_physical_index.payments <region>:<cluster_id>:<id>
```
