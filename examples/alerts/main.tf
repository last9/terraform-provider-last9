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
  indicator   = "throughput"

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

# Expression-based alert
resource "last9_alert" "availability_breach" {
  entity_id   = var.entity_id
  name        = "Availability Breach"
  description = "Availability drops below 99.5%"
  indicator   = "availability"

  expression = "low_spike(0.5, availability)"
  severity   = "breach"

  properties {
    runbook_url = "https://wiki.example.com/runbooks/availability"
    annotations = {
      priority = "critical"
      team     = "reliability"
    }
  }

  mute = false
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

