# Example: Macro configuration

terraform {
  required_providers {
    last9 = {
      source  = "last9/last9"
      version = "~> 1.0"
    }
  }
}

provider "last9" {
  api_token = var.last9_api_token
  org       = var.last9_org
}

resource "last9_macro" "production_macros" {
  cluster_id = var.cluster_id
  
  body = jsonencode({
    macros = {
      "service_name" = "$service"
      "environment"  = "$env"
      "region"      = "$region"
      
      # Custom query macros
      "request_rate" = "sum(rate(http_requests_total{service=\"$service\", env=\"$env\"}[5m]))"
      "error_rate"  = "sum(rate(http_requests_total{service=\"$service\", env=\"$env\", status=~\"5..\"}[5m]))"
      "latency_p95" = "histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{service=\"$service\", env=\"$env\"}[5m])) by (le))"
      
      # Aggregation macros
      "total_requests" = "sum(http_requests_total{service=\"$service\", env=\"$env\"})"
      "error_percentage" = "sum(rate(http_requests_total{service=\"$service\", env=\"$env\", status=~\"5..\"}[5m])) / sum(rate(http_requests_total{service=\"$service\", env=\"$env\"}[5m])) * 100"
    }
  })
}

variable "last9_api_token" {
  type      = string
  sensitive = true
}

variable "last9_org" {
  type = string
}

variable "cluster_id" {
  type        = string
  description = "Cluster ID for macros"
}

