# Example: Dashboard-only configuration

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

resource "last9_dashboard" "service_overview" {
  name        = "Service Overview"
  description = "Overview dashboard for all services"
  readonly    = false

  panels {
    title         = "Service Health"
    query         = "up{job=\"service\"}"
    visualization = "stat"
  }

  panels {
    title         = "Request Rate by Service"
    query         = "sum(rate(http_requests_total[5m])) by (service)"
    visualization = "line"
  }

  panels {
    title         = "Error Rate by Service"
    query         = "sum(rate(http_requests_total{status=~\"5..\"}[5m])) by (service)"
    visualization = "line"
  }

  tags = ["services", "overview"]
}

variable "last9_api_token" {
  type      = string
  sensitive = true
}

variable "last9_org" {
  type = string
}

