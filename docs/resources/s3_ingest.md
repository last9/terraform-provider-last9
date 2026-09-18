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

## Import

```shell
terraform import last9_s3_ingest.imports <region>:<id>
```
