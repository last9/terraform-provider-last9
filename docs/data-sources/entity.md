---
page_title: "last9_entity Data Source - Last9"
subcategory: ""
description: |-
  Retrieves information about a Last9 entity.
---

# last9_entity (Data Source)

Retrieves information about an existing Last9 entity by ID or external reference.

## Example Usage

### By ID

```terraform
data "last9_entity" "example" {
  id = "entity-uuid-here"
}

output "entity_name" {
  value = data.last9_entity.example.name
}
```

### By External Reference

```terraform
data "last9_entity" "example" {
  external_ref = "api-service-prod"
}

output "entity_id" {
  value = data.last9_entity.example.id
}
```

## Schema

### Optional

- `id` (String) The ID of the entity to retrieve.
- `external_ref` (String) The external reference of the entity to retrieve.

~> **Note** Either `id` or `external_ref` must be provided.

### Read-Only

- `name` (String) Name of the entity.
- `type` (String) Type of the entity.
- `description` (String) Description of the entity.
- `data_source` (String) Data source for the entity.
- `data_source_id` (String) Data source ID.
- `namespace` (String) Namespace for the entity.
- `team` (String) Team responsible for the entity.
- `tier` (String) Service tier.
- `workspace` (String) Workspace for the entity.
- `entity_class` (String) Entity class.
- `tags` (Map of String) Tags for the entity.
- `labels` (Map of String) Labels for the entity.
- `notification_channels` (List of Number) List of notification channel IDs.
- `ui_readonly` (Boolean) Whether the entity is read-only in the UI.
