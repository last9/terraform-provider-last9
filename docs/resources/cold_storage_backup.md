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

## Import

```shell
terraform import last9_cold_storage_backup.all_logs <region>:<id>
```
