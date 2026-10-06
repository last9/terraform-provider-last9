---
page_title: "last9_alert_snooze Resource - Last9"
subcategory: ""
description: |-
  Snoozes all alerts for an entity until a unix timestamp.
---

# last9_alert_snooze (Resource)

Entity-level mute/snooze (Datadog downtime analogue). Destroy clears the snooze (`until = 0`).

## Example Usage

```terraform
resource "last9_alert_snooze" "maintenance" {
  entity_id = last9_entity.api.id
  until     = 1893456000 # unix timestamp
}
```

## Schema

### Required

- `entity_id` (String) Entity (alert group) ID to snooze. Changing this forces a new resource.
- `until` (Number) Unix timestamp until which alerts are snoozed. Set to 0 to clear. On read, this is set from the API `alert_snoozed_until` value.

### Read-Only

- `id` (String) The ID of this resource. Same as `entity_id`.
- `alert_snoozed_until` (Number) Effective snooze end timestamp from the API.

## Import

```shell
terraform import last9_alert_snooze.maintenance <entity_id>
```
