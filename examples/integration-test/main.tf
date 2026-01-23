terraform {
  required_providers {
    last9 = {
      source  = "last9/last9"
      version = "~> 1.0"
    }
  }
  required_version = ">= 1.0"
}

# Provider Configuration
# Uses either refresh_token (recommended) or api_token (legacy)
provider "last9" {
  refresh_token = var.last9_refresh_token != "" ? var.last9_refresh_token : null
  api_token     = var.last9_refresh_token == "" ? var.last9_api_token : null
  org           = var.last9_org
  api_base_url  = var.last9_api_base_url
}

# Data source to get entity information
data "last9_entity" "production_api" {
  name = var.entity_name
}

# ====================================================================
# DASHBOARD: Production Monitoring Dashboard
# ====================================================================
resource "last9_dashboard" "production_monitoring" {
  name        = "${var.environment}-monitoring-dashboard"
  description = "Comprehensive monitoring dashboard for ${var.environment} environment"
  readonly    = false

  panels {
    title         = "Request Rate (req/min)"
    query         = "sum(rate(http_requests_total{env=\"${var.environment}\"}[5m])) * 60"
    visualization = "line"
    config = {
      legend = "show"
      yaxis  = "left"
      unit   = "req/min"
    }
  }

  panels {
    title         = "Error Rate (%)"
    query         = "100 * sum(rate(http_requests_total{status=~\"5..\",env=\"${var.environment}\"}[5m])) / sum(rate(http_requests_total{env=\"${var.environment}\"}[5m]))"
    visualization = "line"
    config = {
      legend = "show"
      yaxis  = "left"
      unit   = "percent"
    }
  }

  panels {
    title         = "Response Time P95 (ms)"
    query         = "histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{env=\"${var.environment}\"}[5m])) by (le)) * 1000"
    visualization = "line"
    config = {
      legend = "show"
      yaxis  = "left"
      unit   = "ms"
    }
  }

  panels {
    title         = "Service Availability (%)"
    query         = "100 * (1 - sum(rate(http_requests_total{status=~\"5..\",env=\"${var.environment}\"}[5m])) / sum(rate(http_requests_total{env=\"${var.environment}\"}[5m])))"
    visualization = "stat"
    config = {
      thresholds = jsonencode([
        { value = 95, color = "red" },
        { value = 99, color = "yellow" },
        { value = 99.9, color = "green" }
      ])
    }
  }

  tags = [var.environment, "monitoring", "integration-test"]
}

# ====================================================================
# ALERTS: Critical System Alerts
# ====================================================================

# High Error Rate Alert
resource "last9_alert" "high_error_rate" {
  entity_id   = data.last9_entity.production_api.id
  name        = "${var.environment}-high-error-rate"
  description = "Alert when error rate exceeds ${var.error_rate_threshold}% for ${var.alert_bad_minutes} minutes"
  indicator   = "error_rate"

  greater_than  = var.error_rate_threshold
  bad_minutes   = var.alert_bad_minutes
  total_minutes = var.alert_total_minutes

  severity    = "breach"
  is_disabled = false
  mute        = false

  properties {
    runbook_url = var.runbook_base_url != "" ? "${var.runbook_base_url}/high-error-rate" : ""
    annotations = {
      priority    = "critical"
      team        = var.team_name
      environment = var.environment
      alert_type  = "error_rate"
      escalation  = "immediate"
    }
  }

  group_timeseries_notifications = true
}

# Low Availability Alert
resource "last9_alert" "low_availability" {
  entity_id   = data.last9_entity.production_api.id
  name        = "${var.environment}-low-availability"
  description = "Alert when availability drops below ${var.availability_threshold}%"
  indicator   = "availability"

  less_than     = var.availability_threshold
  bad_minutes   = var.alert_bad_minutes
  total_minutes = var.alert_total_minutes

  severity    = "threat"
  is_disabled = false
  mute        = false

  properties {
    runbook_url = var.runbook_base_url != "" ? "${var.runbook_base_url}/low-availability" : ""
    annotations = {
      priority    = "high"
      team        = var.team_name
      environment = var.environment
      alert_type  = "availability"
    }
  }

  group_timeseries_notifications = true
}

# Response Time Alert (using expression)
resource "last9_alert" "high_response_time" {
  entity_id   = data.last9_entity.production_api.id
  name        = "${var.environment}-high-response-time"
  description = "Alert when P95 response time exceeds ${var.response_time_threshold}ms"
  indicator   = "latency_p95"

  expression  = "spike(${var.response_time_threshold / 1000}, latency_p95)"
  severity    = "info"
  is_disabled = false
  mute        = false

  properties {
    runbook_url = var.runbook_base_url != "" ? "${var.runbook_base_url}/high-response-time" : ""
    annotations = {
      priority    = "medium"
      team        = var.team_name
      environment = var.environment
      alert_type  = "latency"
    }
  }

  group_timeseries_notifications = false
}

# ====================================================================
# MACRO: Query Templates
# ====================================================================
resource "last9_macro" "environment_macros" {
  cluster_id = var.cluster_id

  body = jsonencode({
    macros = {
      "service_name"       = var.service_name
      "environment"        = var.environment
      "error_rate_query"   = "100 * sum(rate(http_requests_total{service=\"${var.service_name}\",status=~\"5..\",env=\"${var.environment}\"}[5m])) / sum(rate(http_requests_total{service=\"${var.service_name}\",env=\"${var.environment}\"}[5m]))"
      "request_rate_query" = "sum(rate(http_requests_total{service=\"${var.service_name}\",env=\"${var.environment}\"}[5m])) * 60"
      "latency_p95_query"  = "histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{service=\"${var.service_name}\",env=\"${var.environment}\"}[5m])) by (le)) * 1000"
      "availability_query" = "100 * (1 - sum(rate(http_requests_total{service=\"${var.service_name}\",status=~\"5..\",env=\"${var.environment}\"}[5m])) / sum(rate(http_requests_total{service=\"${var.service_name}\",env=\"${var.environment}\"}[5m])))"
    }
  })
}

# ====================================================================
# POLICY: SLO and Compliance Rules
# ====================================================================
resource "last9_policy" "slo_compliance" {
  name        = "${var.environment}-slo-compliance-policy"
  description = "SLO compliance and alert coverage policy for ${var.environment} environment"

  filters = {
    entity_type = "service"
    environment = var.environment
    tags        = "${var.environment},monitored"
  }

  rules {
    type = "slo_compliance"
    config = {
      slo_name          = "availability"
      threshold         = tostring(var.availability_threshold)
      evaluation_window = "30d"
    }
  }

  rules {
    type = "slo_compliance"
    config = {
      slo_name          = "latency"
      threshold         = "${var.response_time_threshold}ms"
      evaluation_window = "7d"
    }
  }

  rules {
    type = "alert_coverage"
    config = {
      min_alerts          = 2
      required_severities = jsonencode(["breach", "threat"])
    }
  }
}

# ====================================================================
# LOG MANAGEMENT: Drop and Forward Rules
# ====================================================================

# Drop debug logs to reduce costs
resource "last9_drop_rule" "drop_debug_logs" {
  region    = var.region
  name      = "${var.environment}-drop-debug-logs"
  telemetry = "logs"

  filters {
    key         = "attributes[\"level\"]"
    value       = "debug"
    operator    = "equals"
    conjunction = "and"
  }

  filters {
    key         = "attributes[\"environment\"]"
    value       = var.environment
    operator    = "equals"
    conjunction = "and"
  }
}

# Drop test environment logs
resource "last9_drop_rule" "drop_test_logs" {
  count = var.environment != "test" ? 1 : 0

  region    = var.region
  name      = "${var.environment}-drop-test-logs"
  telemetry = "logs"

  filters {
    key         = "attributes[\"environment\"]"
    value       = "test"
    operator    = "equals"
    conjunction = "and"
  }
}

# Forward critical errors to external system
resource "last9_forward_rule" "forward_critical_errors" {
  count = var.external_log_destination != "" ? 1 : 0

  region      = var.region
  name        = "${var.environment}-forward-critical-errors"
  telemetry   = "logs"
  destination = var.external_log_destination

  filters {
    key         = "attributes[\"level\"]"
    value       = "error"
    operator    = "equals"
    conjunction = "and"
  }

  filters {
    key         = "attributes[\"environment\"]"
    value       = var.environment
    operator    = "equals"
    conjunction = "and"
  }

  filters {
    key         = "attributes[\"service\"]"
    value       = var.service_name
    operator    = "equals"
    conjunction = "and"
  }
}

# Forward security-related logs
resource "last9_forward_rule" "forward_security_logs" {
  count = var.security_log_destination != "" ? 1 : 0

  region      = var.region
  name        = "${var.environment}-forward-security-logs"
  telemetry   = "logs"
  destination = var.security_log_destination

  filters {
    key         = "attributes[\"category\"]"
    value       = "security"
    operator    = "equals"
    conjunction = "and"
  }
}

# Forward traces for distributed tracing analysis
resource "last9_forward_rule" "forward_distributed_traces" {
  count = var.trace_destination != "" ? 1 : 0

  region      = var.region
  name        = "${var.environment}-forward-traces"
  telemetry   = "traces"
  destination = var.trace_destination

  filters {
    key         = "resource.attributes[\"service.name\"]"
    value       = var.service_name
    operator    = "equals"
    conjunction = "and"
  }

  filters {
    key         = "attributes[\"environment\"]"
    value       = var.environment
    operator    = "equals"
    conjunction = "and"
  }
}