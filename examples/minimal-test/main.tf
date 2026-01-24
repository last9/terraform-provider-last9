# Minimal Last9 Terraform Provider Test
# This example creates minimal resources to verify provider functionality

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
provider "last9" {
  api_token    = var.last9_api_token
  delete_token = var.last9_delete_token
  org          = var.last9_org
  api_base_url = var.last9_api_base_url
}

# Create a simple entity (alert group) for testing
resource "last9_entity" "minimal_test" {
  name         = "minimal-test-entity"
  type         = "service"
  external_ref = "minimal-test-entity-v1"
  description  = "Minimal test entity for provider verification"
  ui_readonly  = true

  tags = ["minimal-test", "automated"]
}

# Simple Alert - Provider functionality test
resource "last9_alert" "minimal_test" {
  entity_id   = last9_entity.minimal_test.id
  name        = "minimal-test-alert"
  description = "Minimal test alert for provider verification"
  query       = "sum(rate(http_requests_total{status=~\"5..\"}[5m])) / sum(rate(http_requests_total[5m])) * 100"

  greater_than  = 50.0
  bad_minutes   = 2
  total_minutes = 5

  severity    = "breach"
  is_disabled = false

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
  name      = "minimal-test-drop-rule"
  telemetry = "logs"

  filters {
    key      = "attributes[\"test\"]"
    value    = "minimal"
    operator = "equals"
  }

  action {
    name = "drop-matching"
  }
}

# Variables
variable "last9_api_token" {
  description = "Last9 API access token"
  type        = string
  sensitive   = true
}

variable "last9_delete_token" {
  description = "Last9 API token with delete scope"
  type        = string
  sensitive   = true
  default     = ""
}

variable "last9_org" {
  description = "Last9 organization slug"
  type        = string
}

variable "last9_api_base_url" {
  description = "Last9 API base URL"
  type        = string
  default     = "https://app.last9.io"
}

variable "region" {
  description = "Region for drop rule"
  type        = string
  default     = "ap-south-1"
}

# Outputs
output "test_results" {
  description = "Minimal test results"
  value = {
    entity_created    = last9_entity.minimal_test.id != ""
    alert_created     = last9_alert.minimal_test.id != ""
    drop_rule_created = last9_drop_rule.minimal_test.id != ""
  }
}

output "entity_id" {
  description = "Created entity ID"
  value       = last9_entity.minimal_test.id
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
    provider_working  = true
    resources_created = 3
  }
}
