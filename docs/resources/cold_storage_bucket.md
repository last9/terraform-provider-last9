---
page_title: "last9_cold_storage_bucket Resource - Last9"
subcategory: ""
description: |-
  Manages an OTel cold storage S3 bucket configuration.
---

# last9_cold_storage_bucket (Resource)

Configures an S3 bucket used for log cold storage via `/otel_settings/cold_storage/bucket`.

Auth is either IAM `role` (recommended) or static `credentials`.

## Example Usage

```terraform
resource "last9_cold_storage_bucket" "archive" {
  region     = "ap-south-1"
  name       = "last9-log-archive"
  aws_region = "ap-south-1"
  aws_bucket = "my-company-last9-archives"
  auth_type  = "role"
  aws_role   = "arn:aws:iam::123456789012:role/Last9ColdStorage"
  default    = true
}
```

## Schema

### Required

- `auth_type` (String) Authentication type for the S3 bucket. Valid values: `credentials`, `role`.
- `aws_bucket` (String) Name of the S3 bucket. Sent as `properties.aws_bucket`.
- `aws_region` (String) AWS region of the S3 bucket. Sent as `properties.aws_region`.
- `name` (String) Name of the bucket configuration. Referenced by `bucket_name` in `last9_cold_storage_backup`.
- `region` (String) Last9 region. Sent as the `region` query parameter and used in the resource ID. Changing this forces a new resource.

### Optional

- `aws_access_key` (String, Sensitive) AWS access key for `credentials` auth. Sent as `properties.aws_access_key`. The value in state is kept when the API does not return it.
- `aws_role` (String) IAM role ARN for `role` auth. Sent as `properties.aws_role`.
- `aws_secret_key` (String, Sensitive) AWS secret key for `credentials` auth. Sent as `properties.aws_secret_key`. The value in state is kept when the API does not return it.
- `default` (Boolean) When `true`, the provider marks this bucket as the default bucket after create, or after update when the value changes. Default: `false`.
- `retention_period` (Number) Maps to the API `properties.retention_period` field. Omitted from the request when not set.

### Read-Only

- `id` (String) The ID of this resource, in the format `region:id`.
- `status` (String) Status returned by the API.

## Import

```shell
terraform import last9_cold_storage_bucket.archive <region>:<id>
```
