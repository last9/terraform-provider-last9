# Cold Storage Example

```hcl
terraform {
  required_providers {
    last9 = {
      source  = "last9/last9"
      version = "~> 0.5"
    }
  }
}

provider "last9" {
  refresh_token        = var.last9_refresh_token
  delete_refresh_token = var.last9_delete_refresh_token
  org                  = var.last9_org
  api_base_url         = var.last9_api_base_url
}

resource "last9_cold_storage_bucket" "archive" {
  region     = var.region
  name       = "last9-log-archive"
  aws_region = "ap-south-1"
  aws_bucket = var.aws_bucket
  auth_type  = "role"
  aws_role   = var.aws_role
  default    = true
}

resource "last9_cold_storage_backup" "all_logs" {
  region      = var.region
  name        = "backup-all"
  bucket_name = last9_cold_storage_bucket.archive.name
  granularity = "index"
  enabled     = true
}

resource "last9_s3_ingest" "imports" {
  region     = var.region
  name       = "customer-log-imports"
  aws_region = "ap-south-1"
  aws_bucket = var.aws_ingest_bucket
  aws_role   = var.aws_role
  auth_type  = "role"
}
```
