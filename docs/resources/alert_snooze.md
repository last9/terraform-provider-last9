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

## Import

```shell
terraform import last9_alert_snooze.maintenance <entity_id>
```
