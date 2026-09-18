# Streaming Aggregation Example

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

resource "last9_streaming_aggregation" "http_by_service" {
  region        = var.region
  name          = "http-requests-by-service"
  telemetry     = "metrics"
  metric        = "http_requests_total"
  resolution    = "1m"
  aggregation   = "sum"
  clause        = "without"
  labels        = ["instance"]
  output_metric = "http_requests_by_service"
}
