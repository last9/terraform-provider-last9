# Minimal Last9 Terraform Provider Test
# This example creates minimal resources to verify provider functionality

terraform {
  required_providers {
    last9 = {
      source = "hashicorp.com/edu/last9"
    }
  }
  required_version = ">= 1.0"
}

# Provider Configuration
provider "last9" {
  refresh_token = var.last9_refresh_token
  org           = var.last9_org
  api_base_url  = var.last9_api_base_url
}

# Data source to verify connectivity
data "last9_entity" "test" {
  name = var.entity_name
}

# Simple Dashboard - Core functionality test
resource "last9_dashboard" "minimal_test" {
  name        = "minimal-test-dashboard-${random_string.suffix.result}"
  description = "Minimal test dashboard for provider verification"
  readonly    = false

  panels {
    title         = "Test Panel - Request Rate"
    query         = "sum(rate(http_requests_total[5m]))"
    visualization = "line"
  }

  tags = ["minimal-test", "automated"]
}

# Simple Alert - Provider functionality test
resource "last9_alert" "minimal_test" {
  entity_id   = data.last9_entity.test.id
  name        = "minimal-test-alert-${random_string.suffix.result}"
  description = "Minimal test alert for provider verification"
  indicator   = "error_rate"

  greater_than  = 50.0
  bad_minutes   = 2
  total_minutes = 5

  severity   = "info"
  is_disabled = false
  mute        = false

  properties {
    annotations = {
      test_type = "minimal"
      automated = "true"
    }
  }

  group_timeseries_notifications = false
}

# Simple Drop Rule - Log management functionality test
resource "last9_drop_rule" "minimal_test" {
  region    = var.region
  name      = "minimal-test-drop-rule-${random_string.suffix.result}"
  telemetry = "logs"

  filters {
    key         = "attributes[\"test\"]"
    value       = "minimal"
    operator    = "equals"
    conjunction = "and"
  }
}

# Random suffix to avoid naming conflicts
resource "random_string" "suffix" {
  length  = 8
  lower   = true
  upper   = false
  numeric = true
  special = false
}

# Variables
variable "last9_refresh_token" {
  description = "Last9 refresh token"
  type        = string
  sensitive   = true
}

variable "last9_org" {
  description = "Last9 organization slug"
  type        = string
}

variable "last9_api_base_url" {
  description = "Last9 API base URL"
  type        = string
  default     = "https://api.last9.io"
}

variable "entity_name" {
  description = "Entity name for testing"
  type        = string
}

variable "region" {
  description = "Region for drop rule"
  type        = string
  default     = "us-west-2"
}

# Outputs
output "test_results" {
  description = "Minimal test results"
  value = {
    entity_found = data.last9_entity.test.id != ""
    dashboard_created = last9_dashboard.minimal_test.id
    alert_created = last9_alert.minimal_test.id
    drop_rule_created = last9_drop_rule.minimal_test.id
    test_suffix = random_string.suffix.result
  }
}

output "dashboard_id" {
  description = "Created dashboard ID"
  value       = last9_dashboard.minimal_test.id
}

output "alert_id" {
  description = "Created alert ID"
  value       = last9_alert.minimal_test.id
}

output "drop_rule_id" {
  description = "Created drop rule ID"
  value       = last9_drop_rule.minimal_test.id
}

output "validation_summary" {
  description = "Summary for validation"
  value = {
    provider_working = true
    resources_created = 3
    data_sources_working = true
    test_id = random_string.suffix.result
  }
}