---
page_title: "last9_datasource Data Source - Last9"
subcategory: ""
description: |-
  Looks up a Last9 datasource by id or name.
---

# last9_datasource (Data Source)

```terraform
data "last9_datasource" "default" {}
```

## Schema

### Optional

- `id` (String) Datasource ID to look up. Takes precedence over `name`.
- `name` (String) Datasource name to look up. If neither `id` nor `name` is set, the datasource marked as default is returned, or the first datasource if none is marked as default.

### Read-Only

- `default` (Boolean) Whether the datasource is the default. Maps to the API `is_default` field.
- `region` (String) Maps to the API `region` field.
- `type` (String) Maps to the API `type` field.
