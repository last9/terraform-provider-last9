---
page_title: "last9_physical_index Resource - Last9"
subcategory: ""
description: |-
  Manages an OTel physical index for logs.
---

# last9_physical_index (Resource)

Configures a physical index partition for logs via `/otel_settings/physical_index`.

## Example Usage

```terraform
resource "last9_physical_index" "payments" {
  region      = "ap-south-1"
  name        = "payments_logs"
  telemetry   = "logs"
  description = "Payments service logs"
  retain      = true

  filters {
    key      = "attributes[\"service\"]"
    value    = "payments"
    operator = "equals"
  }
}
```

## Schema

### Required

- `filters` (Block List) Filter conditions that select the logs for this index. At least one block is required. (see [below for nested schema](#nestedblock--filters))
- `name` (String) Physical index name. Must contain only letters, digits, and underscores. Changing this forces a new resource.
- `region` (String) Last9 region. Sent as the `region` query parameter and used in the resource ID. Changing this forces a new resource.
- `telemetry` (String) Telemetry type. Valid values: `logs`.

### Optional

- `bucket_name` (String) Maps to the API `properties.bucket_name` field. Omitted from the request when empty.
- `cluster_id` (String) Cluster ID. If not set, the provider uses the default cluster for the region. Changing this forces a new resource.
- `description` (String) Description of the physical index.
- `retain` (Boolean) Maps to the API `properties.retain` field. Default: `false`.
- `retention_period` (Number) Maps to the API `properties.retention_period` field. Sent as `null` when not set, including when removed from an existing configuration.

### Read-Only

- `destination` (String) Destination returned by the API. The provider preserves it in update requests.
- `id` (String) The ID of this resource, in the format `region:cluster_id:id`.
- `status` (String) Status returned by the API.

<a id="nestedblock--filters"></a>
### Nested Schema for `filters`

Required:

- `operator` (String) Comparison operator. Valid values: `equals`, `not_equals`, `like`.
- `value` (String) The value to match.

Optional:

- `conjunction` (String) Logical conjunction used to combine filters. Omitted from the request when empty.
- `key` (String) The field key to filter on, for example `attributes["service"]`.

## Import

```shell
terraform import last9_physical_index.payments <region>:<cluster_id>:<id>
```
