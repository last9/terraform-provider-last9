# Integration Test for Last9 Terraform Provider
# This example tests core provider functionality

terraform {
  required_providers {
    last9 = {
      source  = "last9/last9"
      version = "~> 0.2"
    }
  }
  required_version = ">= 1.0"
}

# Provider Configuration
# Supports both authentication methods:
# - refresh_token (recommended): Auto-generates access tokens
# - api_token (legacy): Static access token
provider "last9" {
  # Authentication - use refresh_token if provided, otherwise api_token
  refresh_token        = var.last9_refresh_token
  api_token            = var.last9_api_token
  delete_refresh_token = var.last9_delete_refresh_token
  delete_token         = var.last9_delete_token

  org          = var.last9_org
  api_base_url = var.last9_api_base_url
}

# ====================================================================
# LOCALS: Generate unique destinations for integration tests
# These ensure no conflicts with existing notification channels
# ====================================================================
locals {
  # Generate unique destinations using environment name (contains timestamp)
  webhook_url           = var.webhook_url != "" ? var.webhook_url : "https://webhook.site/${var.environment}"
  slack_webhook_url     = var.slack_webhook_url != "" ? var.slack_webhook_url : "https://hooks.slack.com/services/T00000000/B00000000/${var.environment}"
  pagerduty_key         = var.pagerduty_integration_key != "" ? var.pagerduty_integration_key : "pd-key-${var.environment}"
  alert_email           = var.alert_email != "" ? var.alert_email : "${var.environment}@last9.io"
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
# DROP RULES: Filter out telemetry for cost optimization
# ====================================================================

# Drop debug logs
resource "last9_drop_rule" "drop_debug_logs" {
  region    = var.region
  name      = "${var.environment}-drop-debug-logs"
  telemetry = "logs"

  filters {
    key      = "attributes[\"level\"]"
    value    = "debug"
    operator = "equals"
  }

  action {
    name = "drop-matching"
  }
}

# Drop test service traces
resource "last9_drop_rule" "drop_test_traces" {
  region    = var.region
  name      = "${var.environment}-drop-test-traces"
  telemetry = "traces"

  filters {
    key      = "resource.attributes[\"service.name\"]"
    value    = "test-.*"
    operator = "like"
  }

  action {
    name = "drop-matching"
  }
}

# Drop test metrics (using non-existent metric names for safe testing)
resource "last9_drop_rule" "drop_test_metrics" {
  region    = var.region
  name      = "${var.environment}-drop-test-metrics"
  telemetry = "metrics"

  filters {
    key      = "name"
    value    = "integration_test_fake_metric"
    operator = "equals"
  }

  action {
    name = "drop-matching"
  }
}

# ====================================================================
# NOTIFICATION CHANNELS: Various types for alerting
# ====================================================================

# Generic Webhook
resource "last9_notification_channel" "webhook" {
  name          = "${var.environment}-webhook"
  type          = "generic_webhook"
  destination   = local.webhook_url
  send_resolved = true
}

# Slack
resource "last9_notification_channel" "slack" {
  name          = "${var.environment}-slack"
  type          = "slack"
  destination   = local.slack_webhook_url
  send_resolved = true
}

# PagerDuty
resource "last9_notification_channel" "pagerduty" {
  name          = "${var.environment}-pagerduty"
  type          = "pagerduty"
  destination   = local.pagerduty_key
  send_resolved = true
}

# Email
resource "last9_notification_channel" "email" {
  name          = "${var.environment}-email"
  type          = "email"
  destination   = local.alert_email
  send_resolved = false
}

# ====================================================================
# FORWARD RULE: Forward critical errors
# ====================================================================
resource "last9_forward_rule" "forward_critical_errors" {
  region      = var.region
  name        = "${var.environment}-forward-critical-errors"
  telemetry   = "logs"
  destination = local.webhook_url

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

# ====================================================================
# SCHEDULED SEARCH ALERT: Log-based alerting
# ====================================================================
resource "last9_scheduled_search_alert" "high_error_count" {
  region           = var.region
  name             = "${var.environment}-high-error-count"
  query_type       = "logjson-aggregate"
  physical_index   = "logs"
  telemetry        = "logs"
  search_frequency = 300 # 5 minutes

  query = jsonencode([
    {
      type = "filter"
      query = {
        "$and" = [
          { "$eq" = ["SeverityText", "ERROR"] }
        ]
      }
    }
  ])

  post_processor {
    type = "aggregate"

    aggregates {
      function = jsonencode({ "$count" = [] })
      as       = "error_count"
    }

    groupby = "{}"
  }

  threshold {
    operator = ">"
    value    = 100
  }

  alert_destinations = [last9_notification_channel.webhook.id]
}
