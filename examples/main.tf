terraform {
  required_providers {
    last9 = {
      source  = "last9/last9"
      version = "~> 1.0"
    }
  }
}

provider "last9" {
  api_token  = var.last9_api_token
  org        = var.last9_org
  api_base_url = "https://api.last9.io"
}

# Example: Dashboard Resource
resource "last9_dashboard" "production_metrics" {
  name        = "Production Metrics Dashboard"
  description = "Comprehensive dashboard for production environment monitoring"
  readonly    = false

  panels {
    title         = "Request Rate"
    query         = "sum(rate(http_requests_total[5m])) by (service)"
    visualization = "line"
    config = {
      legend = "show"
      yaxis  = "left"
    }
  }

  panels {
    title         = "Error Rate"
    query         = "sum(rate(http_requests_total{status=~\"5..\"}[5m])) by (service)"
    visualization = "line"
    config = {
      legend = "show"
      yaxis  = "left"
    }
  }

  panels {
    title         = "Response Time (p95)"
    query         = "histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, service))"
    visualization = "line"
  }

  tags = ["production", "monitoring", "metrics"]
}

# Example: Alert Resource
resource "last9_alert" "high_error_rate" {
  entity_id   = "entity-123"  # Replace with actual entity ID
  name        = "High Error Rate Alert"
  description = "Alert when error rate exceeds threshold for 5 minutes"
  indicator   = "error_rate"
  
  greater_than  = 100
  bad_minutes   = 5
  total_minutes = 10
  
  severity = "breach"
  
  properties {
    runbook_url = "https://wiki.example.com/runbooks/high-error-rate"
    annotations = {
      priority    = "high"
      team        = "platform"
      escalation  = "oncall"
    }
  }
  
  group_timeseries_notifications = true
}

resource "last9_alert" "availability_threat" {
  entity_id   = "entity-123"
  name        = "Availability Threat"
  description = "Availability drops below 99.5%"
  indicator   = "availability"
  
  expression = "low_spike(0.5, availability)"
  severity   = "threat"
  
  properties {
    runbook_url = "https://wiki.example.com/runbooks/availability"
    annotations = {
      priority = "medium"
      team     = "reliability"
    }
  }
  
  mute = false
}

# Example: Macro Resource
resource "last9_macro" "cluster_macros" {
  cluster_id = "cluster-123"  # Replace with actual cluster ID
  
  body = jsonencode({
    macros = {
      "service_name" = "$service"
      "environment"  = "$env"
      "custom_query" = "sum(rate(http_requests_total{service=\"$service\", env=\"$env\"}[5m]))"
      "error_query"  = "sum(rate(http_requests_total{service=\"$service\", status=~\"5..\"}[5m]))"
    }
  })
}

# Example: Policy Resource
resource "last9_policy" "slo_compliance" {
  name        = "SLO Compliance Policy"
  description = "Ensures all production services meet SLO requirements"
  
  filters = {
    entity_type = "service"
    tags        = "production"
  }
  
  rules {
    type = "slo_compliance"
    config = {
      slo_name          = "availability"
      threshold         = "99.9"
      evaluation_window = "30d"
    }
  }
  
  rules {
    type = "alert_coverage"
    config = {
      min_alerts = 3
    }
  }
  
  rules {
    type = "slo_compliance"
    config = {
      slo_name          = "latency"
      threshold         = "200ms"
      evaluation_window = "7d"
    }
  }
}

# Variables
variable "last9_api_token" {
  description = "Last9 API token"
  type        = string
  sensitive   = true
}

variable "last9_org" {
  description = "Last9 organization slug"
  type        = string
}

# Outputs
output "dashboard_id" {
  description = "ID of the created dashboard"
  value       = last9_dashboard.production_metrics.id
}

output "alert_ids" {
  description = "IDs of the created alerts"
  value = [
    last9_alert.high_error_rate.id,
    last9_alert.availability_threat.id
  ]
}

output "policy_id" {
  description = "ID of the created policy"
  value       = last9_policy.slo_compliance.id
}

