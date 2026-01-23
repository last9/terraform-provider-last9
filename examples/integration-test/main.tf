# Integration Test for Last9 Terraform Provider
# This example tests core provider functionality

terraform {
  required_providers {
    last9 = {
      source = "last9/last9"
    }
  }
  required_version = ">= 1.0"
}

# Provider Configuration
provider "last9" {
  api_token    = var.last9_api_token
  org          = var.last9_org
  api_base_url = var.last9_api_base_url
}

# ====================================================================
# ENTITY (ALERT GROUP): Container for metric-based alerts
# ====================================================================
resource "last9_entity" "integration_test" {
  name         = "${var.environment}-integration-test"
  type         = "service"
  external_ref = "${var.environment}-integration-test-v1"
  description  = "Integration test alert group for ${var.environment} environment"

  ui_readonly = true

  tags = [var.environment, "integration-test", "automated"]

  labels = {
    environment = var.environment
    managed_by  = "terraform"
  }
}

# ====================================================================
# ALERTS: Metric-based alerts
# ====================================================================

# High Error Rate Alert
resource "last9_alert" "high_error_rate" {
  entity_id   = last9_entity.integration_test.id
  name        = "${var.environment}-high-error-rate"
  description = "Alert when error rate exceeds ${var.error_rate_threshold}%"
  query       = "100 * sum(rate(http_requests_total{status=~\"5..\",env=\"${var.environment}\"}[5m])) / sum(rate(http_requests_total{env=\"${var.environment}\"}[5m]))"

  greater_than  = var.error_rate_threshold
  bad_minutes   = var.alert_bad_minutes
  total_minutes = var.alert_total_minutes

  severity    = "breach"
  is_disabled = false

  properties {
    runbook_url = var.runbook_base_url != "" ? "${var.runbook_base_url}/high-error-rate" : ""
    annotations = {
      priority    = "critical"
      environment = var.environment
      alert_type  = "error_rate"
    }
  }

  group_timeseries_notifications = true
}

# Low Availability Alert
resource "last9_alert" "low_availability" {
  entity_id   = last9_entity.integration_test.id
  name        = "${var.environment}-low-availability"
  description = "Alert when availability drops below ${var.availability_threshold}%"
  query       = "100 * (1 - sum(rate(http_requests_total{status=~\"5..\",env=\"${var.environment}\"}[5m])) / sum(rate(http_requests_total{env=\"${var.environment}\"}[5m])))"

  less_than     = var.availability_threshold
  bad_minutes   = var.alert_bad_minutes
  total_minutes = var.alert_total_minutes

  severity    = "threat"
  is_disabled = false

  properties {
    runbook_url = var.runbook_base_url != "" ? "${var.runbook_base_url}/low-availability" : ""
    annotations = {
      priority    = "high"
      environment = var.environment
      alert_type  = "availability"
    }
  }

  group_timeseries_notifications = true
}

# ====================================================================
# MACRO: Reusable PromQL queries
# ====================================================================
resource "last9_macro" "integration_test" {
  cluster_id = var.cluster_id

  body = jsonencode({
    macros = {
      "${var.environment}_error_rate"   = "100 * sum(rate(http_requests_total{status=~\"5..\",env=\"${var.environment}\"}[$$window])) / sum(rate(http_requests_total{env=\"${var.environment}\"}[$$window]))"
      "${var.environment}_request_rate" = "sum(rate(http_requests_total{env=\"${var.environment}\"}[$$window])) * 60"
      "${var.environment}_latency_p95"  = "histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{env=\"${var.environment}\"}[$$window])) by (le)) * 1000"
    }
  })
}

# ====================================================================
# DROP RULE: Filter out debug logs
# ====================================================================
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

  action {
    name        = "drop"
    destination = "null"
  }
}

# ====================================================================
# FORWARD RULE: Forward critical errors (optional)
# ====================================================================
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
}
