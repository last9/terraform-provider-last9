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
  refresh_token = var.last9_refresh_token
  org           = var.last9_org
}

# Define reusable PromQL macros for the cluster
resource "last9_macro" "production_macros" {
  cluster_id = var.cluster_id

  body = <<-EOT
    # Availability macro - calculates success rate
    function availability(metric, window, service) {
      let exclude = "/metrics|/health"
      let good_codes = "2.*"
      let all_codes = "2.*|4.*|5.*"

      return sum(rate(metric{handler!~exclude, service=service, code=~good_codes}[window])) / sum(rate(metric{handler!~exclude, service=service, code=~all_codes}[window]))
    }

    # Error rate as percentage
    function error_rate(service, window) {
      return sum(rate(http_requests_total{service=service, status=~"5.."}[window])) / sum(rate(http_requests_total{service=service}[window])) * 100
    }

    # Request rate per second
    function request_rate(service, window) {
      return sum(rate(http_requests_total{service=service}[window]))
    }

    # Latency percentiles
    function latency_p95(service, window) {
      return histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{service=service}[window])) by (le))
    }

    function latency_p99(service, window) {
      return histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket{service=service}[window])) by (le))
    }
  EOT
}

variable "last9_refresh_token" {
  type        = string
  sensitive   = true
  description = "Last9 refresh token"
}

variable "last9_org" {
  type        = string
  description = "Last9 organization"
}

variable "cluster_id" {
  type        = string
  description = "Cluster ID for macros"
}
