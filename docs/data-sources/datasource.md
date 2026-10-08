---
page_title: "last9_datasource Data Source - Last9"
subcategory: ""
description: |-
  Looks up a Last9 datasource by id or name.
---

# last9_datasource (Data Source)

Looks up a datasource by `id` or `name`. Set at most one selector. With neither selector, the datasource marked as default is returned; the data source returns an error when no default datasource exists.

```terraform
data "last9_datasource" "default" {}
```

## Schema

### Optional

- `id` (String) Datasource ID to look up. Cannot be set with `name`.
- `name` (String) Datasource name to look up. Cannot be set with `id`.

### Read-Only

- `default` (Boolean) Whether the datasource is the default. Maps to the API `is_default` field.
- `region` (String) Maps to the API `region` field.
- `type` (String) Maps to the API `type` field.
