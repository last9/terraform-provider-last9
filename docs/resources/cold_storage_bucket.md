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

## Import

```shell
terraform import last9_cold_storage_bucket.archive <region>:<id>
```
