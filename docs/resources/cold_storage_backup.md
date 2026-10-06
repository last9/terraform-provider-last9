---
page_title: "last9_cold_storage_backup Resource - Last9"
subcategory: ""
description: |-
  Manages an OTel cold storage backup rule.
---

# last9_cold_storage_backup (Resource)

Defines which log data is written to a cold storage bucket (`/otel_settings/cold_storage/backup`).

`granularity` is `index` (whole index; `targets` must be empty) or `service` (list services in `targets`).

## Example Usage

```terraform
resource "last9_cold_storage_backup" "all_logs" {
  region      = "ap-south-1"
  name        = "backup-all"
  bucket_name = last9_cold_storage_bucket.archive.name
  granularity = "index"
  enabled     = true
}
```

## Schema

### Required

- `bucket_name` (String) Cold storage bucket name (the `name` of a `last9_cold_storage_bucket`). Sent as `properties.bucket_name`.
- `granularity` (String) Backup granularity. Valid values: `index`, `service`.
- `name` (String) Name of the backup rule.
- `region` (String) Last9 region. Sent as the `region` query parameter and used in the resource ID. Changing this forces a new resource.

### Optional

- `enabled` (Boolean) Whether the backup rule is enabled. Sent as `properties.enabled`. Default: `true`.
- `targets` (List of String) Services to back up. Required when `granularity` is `service`; must be empty when `granularity` is `index`.

### Read-Only

- `id` (String) The ID of this resource, in the format `region:id`.
- `status` (String) Status returned by the API.

## Import

```shell
terraform import last9_cold_storage_backup.all_logs <region>:<id>
```
