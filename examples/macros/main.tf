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
  api_token    = var.last9_api_token
  org          = var.last9_org
  api_base_url = var.last9_api_base_url
}

# Define reusable PromQL macros for the cluster
resource "last9_macro" "production_macros" {
  cluster_id = var.cluster_id

  # The body must be a JSON object containing macro definitions
  body = jsonencode({
    macros = {
      # Availability calculation: successful requests / total requests
      availability = "sum(rate(http_requests_total{status=~\"2..\"}[$window])) / sum(rate(http_requests_total[$window]))"

      # Error rate as percentage
      error_rate = "sum(rate(http_requests_total{status=~\"5..\"}[$window])) / sum(rate(http_requests_total[$window])) * 100"

      # Request rate per second
      request_rate = "sum(rate(http_requests_total[$window]))"

      # Latency P95
      latency_p95 = "histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[$window])) by (le))"

      # Latency P99
      latency_p99 = "histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[$window])) by (le))"
    }
  })
}

variable "last9_api_token" {
  type        = string
  sensitive   = true
  description = "Last9 API access token"
}

variable "last9_org" {
  type        = string
  description = "Last9 organization"
}

variable "last9_api_base_url" {
  type        = string
  description = "Last9 API base URL"
  default     = "https://app.last9.io"
}

variable "cluster_id" {
  type        = string
  description = "Cluster ID for macros"
}
