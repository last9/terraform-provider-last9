---
page_title: "last9_s3_ingest Resource - Last9"
subcategory: ""
description: |-
  Manages an OTel S3 ingest bucket configuration.
---

# last9_s3_ingest (Resource)

Configures S3 ingest for pulling objects into Last9 (`/otel_settings/s3_ingest`).
`auth_type` must be `role`.

## Example Usage

```terraform
resource "last9_s3_ingest" "imports" {
  region     = "ap-south-1"
  name       = "customer-log-imports"
  aws_region = "ap-south-1"
  aws_bucket = "my-company-last9-ingest"
  aws_role   = "arn:aws:iam::123456789012:role/Last9S3Ingest"
  auth_type  = "role"
}
```

## Schema

### Required

- `aws_bucket` (String) Name of the S3 bucket. Sent as `properties.aws_bucket`.
- `aws_region` (String) AWS region of the S3 bucket. Sent as `properties.aws_region`.
- `aws_role` (String) IAM role ARN (`auth_type` is always `role`). Sent as `properties.aws_role`.
- `name` (String) Name of the S3 ingest configuration.
- `region` (String) Last9 region. Sent as the `region` query parameter and used in the resource ID. Changing this forces a new resource.

### Optional

- `auth_type` (String) Authentication type. Valid values: `role`. Default: `role`.
- `default` (Boolean) Maps to the API `properties.default` field. Default: `false`.

### Read-Only

- `id` (String) The ID of this resource, in the format `region:id`.
- `status` (String) Status returned by the API.

## Import

```shell
terraform import last9_s3_ingest.imports <region>:<id>
```
