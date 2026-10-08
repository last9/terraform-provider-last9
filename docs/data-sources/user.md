---
page_title: "last9_user Data Source - Last9"
subcategory: ""
description: |-
  Looks up a Last9 organization user by id or email.
---

# last9_user (Data Source)

```terraform
data "last9_user" "alice" {
  email = "alice@example.com"
}
```

## Schema

### Optional

- `email` (String) User email to look up. The match is case-insensitive. Exactly one of `id` or `email` must be set.
- `id` (String) User ID to look up. Exactly one of `id` or `email` must be set.

### Read-Only

- `active` (Boolean) `true` when the API `deleted_at` field is not set.
- `name` (String) User name. Maps to the API `name` field.
- `organization_id` (String) Maps to the API `organization_id` field.
- `role` (String) Maps to the API `role` field.
- `status` (String) Maps to the API `status` field.
