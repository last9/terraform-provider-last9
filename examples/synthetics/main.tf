# Synthetic Check Example

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

resource "last9_synthetic_check" "api_health" {
  name      = "api-health"
  type      = "http"
  schedule  = "every 5m"
  timeout   = 30
  frequency = 60
  locations = ["us-east-1"]

  config = jsonencode({
    url    = "https://api.example.com/health"
    method = "GET"
  })
}
```
