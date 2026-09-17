---
page_title: "last9_user Resource - Last9"
subcategory: ""
description: |-
  Invites and manages a Last9 organization user.
---

# last9_user (Resource)

Invites a user by email and manages their organization role and active status.
Analogous to Datadog's `datadog_user` + role assignment.

Create calls `POST /users/invite`, then resolves the user ID by listing users.
Role changes use `PUT /users/{id}/roles`. Deactivate uses `PATCH` with `active=false`.
Destroy deletes/revokes the user (or invite).

## Example Usage

```terraform
resource "last9_user" "oncall" {
  email  = "oncall@example.com"
  role   = "editor"
  active = true
}
```

## Argument Reference

* `email` - (Required, ForceNew) Invite email.
* `role` - (Optional) `admin`, `editor`, or `viewer`. Omit to use the org default on first login.
* `active` - (Optional) Defaults to `true`. Set `false` to deactivate.

## Attribute Reference

* `id` - User UUID.
* `name` - Display name (after acceptance).
* `status` - e.g. `invited`.
* `organization_id` - Organization UUID.

## Import

Import by user ID or email:

```shell
terraform import last9_user.oncall <user_id>
terraform import last9_user.oncall oncall@example.com
```
