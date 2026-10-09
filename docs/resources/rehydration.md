---
page_title: "last9_rehydration Resource - Last9"
subcategory: ""
description: |-
  Creates an OTel log rehydration job.
---

# last9_rehydration (Resource)

Creates a rehydration job to restore archived logs into a physical index for a time range.
ForceNew on all arguments — treat as create-once jobs.

## Example Usage

```terraform
resource "last9_rehydration" "incident" {
  region         = "ap-south-1"
  name           = "incident-2026-09-17"
  physical_index = "payments_logs"
  telemetry      = "logs"
  from           = 1726502400
  to             = 1726588800
  message        = "Rehydrate for incident review"
}
```

## Schema

### Required

- `from` (Number) Start unix timestamp in seconds. Must be a cold/archived range (recent hot windows are rejected by the API). Changing this forces a new resource.
- `name` (String) Name of the rehydration job. Changing this forces a new resource.
- `physical_index` (String) Name of the physical index to restore logs into. Sent as `properties.physical_index`. Changing this forces a new resource.
- `region` (String) Last9 region. Sent as the `region` query parameter and used in the resource ID. Changing this forces a new resource.
- `telemetry` (String) Telemetry type. Valid values: `logs`. Changing this forces a new resource.
- `to` (Number) End unix timestamp in seconds. Must be a cold/archived range (recent hot windows are rejected by the API). Changing this forces a new resource.

### Optional

- `bucket_name` (String) Maps to the API `properties.bucket_name` field. Omitted from the request when empty. Changing this forces a new resource.
- `filters` (Block List) Filter conditions. Sent as `properties.filters`. Changing this forces a new resource. (see [below for nested schema](#nestedblock--filters))
- `granularity` (String) Maps to the API `properties.granularity` field. If not set, the value from the API is kept in state. Changing this forces a new resource.
- `message` (String) Message for the rehydration job. Sent as `properties.message`. Changing this forces a new resource.
- `notification_channel_id` (Number) Maps to the API `properties.notification_channel_id` field. Changing this forces a new resource.
- `targets` (List of String) Maps to the API `properties.targets` field. Changing this forces a new resource.

### Read-Only

- `id` (String) The ID of this resource, in the format `region:id`.
- `status` (String) Status returned by the API. If the status contains `delete`, the provider removes the resource from state.

<a id="nestedblock--filters"></a>
### Nested Schema for `filters`

Required:

- `operator` (String) Comparison operator. Valid values: `equals`, `not_equals`, `like`.
- `value` (String) The value to match.

Optional:

- `conjunction` (String) Logical conjunction used to combine filters. Omitted from the request when empty.
- `key` (String) The field key to filter on.

## Import

```shell
terraform import last9_rehydration.incident <region>:<id>
```
