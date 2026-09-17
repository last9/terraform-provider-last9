---
page_title: "last9_changeboard Resource - Last9"
subcategory: ""
description: |-
  Manages a Last9 changeboard for correlating entity changes.
---

# last9_changeboard (Resource)

Manages a changeboard — a filtered, grouped view of entities used for change correlation.

## Example Usage

```terraform
resource "last9_changeboard" "payments" {
  name        = "payments-services"
  description = "Payment path services"
  owner_id    = var.org_id
  owner_type  = "organization"
  granularity = "1h"

  filter {
    filter_type = "team"
    key         = "team"
    value       = "payments"
    operator    = "equals"
  }

  group {
    name  = "service"
    order = "asc"
  }

  relationship {
    id       = "service"
    child_id = "api"
  }
}
```

## Import

```shell
terraform import last9_changeboard.payments <changeboard_id>
```
