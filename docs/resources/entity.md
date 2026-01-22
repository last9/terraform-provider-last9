---
page_title: "last9_entity Resource - Last9"
subcategory: ""
description: |-
  Manages a Last9 entity (service, component, or other monitored resource).
---

# last9_entity (Resource)

Manages a Last9 entity. Entities represent services, components, or other monitored resources in your infrastructure.

## Example Usage

```terraform
resource "last9_entity" "api_service" {
  name         = "api-service"
  type         = "service"
  external_ref = "api-service-prod"
  description  = "Production API Service"

  tags = {
    environment = "production"
    team        = "platform"
  }

  labels = {
    tier = "backend"
  }
}
```

### With Notification Channels

```terraform
resource "last9_entity" "api_service" {
  name         = "api-service"
  type         = "service"
  external_ref = "api-service-prod"
  description  = "Production API Service"

  notification_channels = [123, 456]
}
```

## Schema

### Required

- `name` (String) Name of the entity.
- `type` (String) Type of the entity (e.g., "service", "component").

### Optional

- `external_ref` (String) External reference identifier for the entity.
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

### Read-Only

- `id` (String) The ID of the entity.

## Import

Entities can be imported using the entity ID:

```shell
terraform import last9_entity.example <entity_id>
```
