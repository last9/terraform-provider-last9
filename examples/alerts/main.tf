# Example: Alert configuration

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

# Static threshold alert
resource "last9_alert" "high_throughput" {
  entity_id   = var.entity_id
  name        = "High Throughput Alert"
  description = "Alert when throughput exceeds 2000 req/min"
  query       = "sum(rate(http_requests_total[5m])) * 60"

  greater_than  = 2000
  bad_minutes   = 4
  total_minutes = 10

  severity = "breach"

  properties {
    runbook_url = "https://wiki.example.com/runbooks/high-throughput"
    annotations = {
      priority = "high"
      team     = "platform"
    }
  }
}

# Availability alert with threshold
resource "last9_alert" "availability_breach" {
  entity_id   = var.entity_id
  name        = "Availability Breach"
  description = "Availability drops below 99.5%"
  query       = "100 * (1 - sum(rate(http_requests_total{status=~\"5..\"}[5m])) / sum(rate(http_requests_total[5m])))"

  less_than     = 99.5
  bad_minutes   = 5
  total_minutes = 10
  severity      = "breach"

  properties {
    runbook_url = "https://wiki.example.com/runbooks/availability"
    annotations = {
      priority = "critical"
      team     = "reliability"
    }
  }
}

variable "last9_api_token" {
  type      = string
  sensitive = true
}

variable "last9_org" {
  type = string
}

variable "entity_id" {
  type        = string
  description = "Entity ID to attach alerts to"
}

