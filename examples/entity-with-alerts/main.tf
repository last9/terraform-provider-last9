# Example: Creating an Entity (Alert Group) with Loss of Signal Alert
# This demonstrates all the new fields added to the Terraform provider

terraform {
  required_providers {
    last9 = {
      source = "hashicorp.com/edu/last9"
    }
  }
}

provider "last9" {
  api_token    = var.last9_api_token
  delete_token = var.last9_delete_token
  org          = var.last9_org
  api_base_url = var.last9_api_base_url
}

# Create an entity (alert group) with comprehensive configuration
resource "last9_entity" "production_api" {
  # Required fields
  name         = "production-api-service"
  type         = "service"
  external_ref = "production-api-service-v1" # Unique slug identifier

  # Description
  description = "Production API service monitoring"

  # Organization and metadata
  namespace = "production"
  team      = "platform-engineering"

  # Tags for categorization
  tags = [
    "production",
    "api",
    "critical",
    "terraform-managed"
  ]

  # Labels - inherited across all indicators during alert evaluation
  labels = {
    environment = "production"
    service     = "api"
    team        = "platform"
    region      = "ap-south-1"
    managed_by  = "terraform"
  }

  # Entity classification
  entity_class = "alert-manager"

  # Prevent UI edits to avoid IaC drift
  ui_readonly = true

  # Define indicators (metrics) for this entity
  indicators {
    name  = "up"
    query = "up{job=\"api-server\"}"
    unit  = "bool"
  }

  indicators {
    name  = "error_rate"
    query = "100 * (sum(rate(http_requests_total{status=~\"5..\"}[5m])) / sum(rate(http_requests_total[5m])))"
    unit  = "percent"
  }

  indicators {
    name  = "latency_p95"
    query = "histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))"
    unit  = "seconds"
  }

  indicators {
    name  = "availability"
    query = "100 * (1 - sum(rate(http_requests_total{status=~\"5..\"}[5m])) / sum(rate(http_requests_total[5m])))"
    unit  = "percent"
  }

  # Related links
  links {
    name = "Runbook - API Service"
    url  = "https://wiki.example.com/runbooks/api-service"
  }

  links {
    name = "Grafana Dashboard"
    url  = "https://grafana.example.com/d/api-service"
  }

  links {
    name = "Service Repository"
    url  = "https://github.com/example/api-service"
  }

  # Notification channels for this entity
  notification_channels = [
    "testing-integrations",
    "PD Test"
  ]
}

# Query existing notification destinations
data "last9_notification_destination" "slack_platform" {
  name = "testing-integrations"
}

data "last9_notification_destination" "pagerduty_oncall" {
  name = "PD Test"
}

# Loss of Signal Alert - triggers when 'up' metric stops reporting
resource "last9_alert" "loss_of_signal" {
  entity_id   = last9_entity.production_api.id
  name        = "Loss of Signal - API Service"
  description = "Alert when the API service stops reporting metrics (loss of signal)"
  query       = "up{job=\"api-server\"}"

  # Triggers when up metric < 1 for 3 out of 5 minutes
  less_than     = 1
  bad_minutes   = 3
  total_minutes = 5

  severity    = "breach"
  is_disabled = false

  properties {
    runbook_url = "https://wiki.example.com/runbooks/loss-of-signal"
    annotations = {
      priority    = "critical"
      team        = "platform-engineering"
      alert_type  = "loss_of_signal"
      metric      = "up"
      description = "Service heartbeat lost - no metrics being reported"
      escalation  = "immediate"
      impact      = "Service monitoring unavailable"
      remediation = "Check service health and metric collection pipeline"
    }
  }

  group_timeseries_notifications = true

  # Notification channels specific to this alert
  notification_channels = [
    data.last9_notification_destination.slack_platform.id,
    data.last9_notification_destination.pagerduty_oncall.id
  ]
}

# High Error Rate Alert
resource "last9_alert" "high_error_rate" {
  entity_id   = last9_entity.production_api.id
  name        = "High Error Rate - API Service"
  description = "Alert when API error rate exceeds 5%"
  query       = "100 * (sum(rate(http_requests_total{status=~\"5..\"}[5m])) / sum(rate(http_requests_total[5m])))"

  greater_than  = 5
  bad_minutes   = 5
  total_minutes = 10

  severity = "breach"

  properties {
    runbook_url = "https://wiki.example.com/runbooks/high-error-rate"
    annotations = {
      priority    = "high"
      team        = "platform-engineering"
      alert_type  = "error_rate"
      threshold   = "5%"
      description = "API error rate exceeded threshold"
      escalation  = "page_on_call"
    }
  }

  notification_channels = [
    data.last9_notification_destination.slack_platform.id
  ]
}

# Low Availability Alert
resource "last9_alert" "low_availability" {
  entity_id   = last9_entity.production_api.id
  name        = "Low Availability - API Service"
  description = "Alert when availability drops below 99.5%"
  query       = "100 * (1 - sum(rate(http_requests_total{status=~\"5..\"}[5m])) / sum(rate(http_requests_total[5m])))"

  less_than     = 99.5
  bad_minutes   = 5
  total_minutes = 10
  severity      = "threat"

  properties {
    runbook_url = "https://wiki.example.com/runbooks/low-availability"
    annotations = {
      priority   = "high"
      team       = "platform-engineering"
      alert_type = "availability"
      slo        = "99.5%"
      impact     = "Service reliability degraded"
    }
  }

  notification_channels = [
    data.last9_notification_destination.pagerduty_oncall.id
  ]
}

# Outputs
output "entity_id" {
  value       = last9_entity.production_api.id
  description = "The ID of the created entity"
}

output "entity_external_ref" {
  value       = last9_entity.production_api.external_ref
  description = "The unique external reference for the entity"
}

output "loss_of_signal_alert_id" {
  value       = last9_alert.loss_of_signal.id
  description = "The ID of the loss of signal alert"
}

output "notification_channels" {
  value = {
    slack     = data.last9_notification_destination.slack_platform.id
    pagerduty = data.last9_notification_destination.pagerduty_oncall.id
  }
  description = "Notification channel IDs"
}
