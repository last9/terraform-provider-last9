---
page_title: "last9_entity Data Source - Last9"
subcategory: ""
description: |-
  Retrieves information about a Last9 alert group.
---

# last9_entity (Data Source)

Retrieves information about an existing alert group by ID or external reference.

## Example Usage

### By ID

```terraform
data "last9_entity" "example" {
  id = "alert-group-uuid-here"
}

output "alert_group_name" {
  value = data.last9_entity.example.name
}
```

### By External Reference

```terraform
data "last9_entity" "example" {
  external_ref = "api-service-prod"
}

output "alert_group_id" {
  value = data.last9_entity.example.id
}
```

## Schema

### Optional

- `id` (String) The ID of the alert group to retrieve.
- `external_ref` (String) The external reference of the alert group to retrieve.

~> **Note** Either `id` or `external_ref` must be provided.

### Read-Only

- `name` (String) Alert group name.
- `type` (String) Type.
- `description` (String) Description.
- `data_source` (String) Data source name.
- `data_source_id` (String) Data source ID.
- `namespace` (String) Namespace.
- `team` (String) Owning team.
- `tier` (String) Tier.
- `workspace` (String) Workspace.
- `labels` (Map of String) Labels.
- `notification_channels` (List of Number) Default notification channel IDs.
- `ui_readonly` (Boolean) Whether read-only in UI.
