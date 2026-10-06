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

## Schema

### Required

- `name` (String) Changeboard name.
- `owner_id` (String) Owner ID (user, organization, or team UUID).
- `owner_type` (String) Owner type. Valid values: `user`, `organization`, `team`.

### Optional

- `description` (String) Changeboard description.
- `filter` (Block List) Entity filters defining which entities belong to this changeboard. Sent as the API `filters` list. (see [below for nested schema](#nestedblock--filter))
- `granularity` (String) Changeboard time granularity, such as `1h`. Sent as `properties.granularity`. If not set, the value from the API is kept in state.
- `group` (Block List) Entity grouping dimensions. Sent as the API `groups` list. (see [below for nested schema](#nestedblock--group))
- `relationship` (Block List) Hierarchy of entity types (`id` = entity type, one nesting level). Sent as the API `relationships` list. (see [below for nested schema](#nestedblock--relationship))

### Read-Only

- `id` (String) The ID of this resource. The changeboard ID returned by the API.
- `created_at` (Number) Maps to the API `created_at` field.
- `updated_at` (Number) Maps to the API `updated_at` field.

<a id="nestedblock--filter"></a>
### Nested Schema for `filter`

Required:

- `filter_type` (String) Maps to the API `filter_type` field (for example, `team`).
- `key` (String) Maps to the API `key` field.
- `operator` (String) Maps to the API `operator` field (for example, `equals`).

Optional:

- `conjunction` (String) Maps to the API `conjunction` field. Omitted from the request when empty.
- `value` (String) Maps to the API `value` field. Omitted from the request when empty.

<a id="nestedblock--group"></a>
### Nested Schema for `group`

Required:

- `name` (String) Grouping dimension name (for example, `service`). Maps to the API `name` field.
- `order` (String) Maps to the API `order` field (for example, `asc`).

<a id="nestedblock--relationship"></a>
### Nested Schema for `relationship`

Required:

- `id` (String) Entity type at this hierarchy node.

Optional:

- `child_id` (String) Nested child entity type (one level). Sent as the single entry in the node's `children` list.

## Import

```shell
terraform import last9_changeboard.payments <changeboard_id>
```
