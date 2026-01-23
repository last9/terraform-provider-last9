# ====================================================================
# DASHBOARD OUTPUTS
# ====================================================================

output "dashboard_id" {
  description = "ID of the created monitoring dashboard"
  value       = last9_dashboard.production_monitoring.id
}

output "dashboard_name" {
  description = "Name of the created monitoring dashboard"
  value       = last9_dashboard.production_monitoring.name
}

output "dashboard_url" {
  description = "URL to access the dashboard (if available)"
  value       = "https://console.last9.io/dashboards/${last9_dashboard.production_monitoring.id}"
}

# ====================================================================
# ALERT OUTPUTS
# ====================================================================

output "alert_ids" {
  description = "IDs of all created alerts"
  value = {
    high_error_rate    = last9_alert.high_error_rate.id
    low_availability   = last9_alert.low_availability.id
    high_response_time = last9_alert.high_response_time.id
  }
}

output "alert_names" {
  description = "Names of all created alerts"
  value = {
    high_error_rate    = last9_alert.high_error_rate.name
    low_availability   = last9_alert.low_availability.name
    high_response_time = last9_alert.high_response_time.name
  }
}

output "alert_summary" {
  description = "Summary of alert configuration"
  value = {
    total_alerts = 3
    severities = {
      breach = 1 # high_error_rate
      threat = 1 # low_availability
      info   = 1 # high_response_time
    }
    thresholds = {
      error_rate_percent   = var.error_rate_threshold
      availability_percent = var.availability_threshold
      response_time_ms     = var.response_time_threshold
    }
  }
}

# ====================================================================
# MACRO OUTPUTS
# ====================================================================

output "macro_cluster_id" {
  description = "Cluster ID where macros were created"
  value       = last9_macro.environment_macros.cluster_id
}

output "macro_details" {
  description = "Details of the created macros"
  value = {
    cluster_id   = var.cluster_id
    environment  = var.environment
    service_name = var.service_name
    total_macros = 5 # service_name, environment, error_rate_query, request_rate_query, latency_p95_query, availability_query
  }
}

# ====================================================================
# POLICY OUTPUTS
# ====================================================================

output "policy_id" {
  description = "ID of the created SLO compliance policy"
  value       = last9_policy.slo_compliance.id
}

output "policy_name" {
  description = "Name of the created SLO compliance policy"
  value       = last9_policy.slo_compliance.name
}

output "policy_summary" {
  description = "Summary of policy configuration"
  value = {
    name        = last9_policy.slo_compliance.name
    total_rules = 3 # 2 SLO compliance rules + 1 alert coverage rule
    environment = var.environment
    filters = {
      entity_type = "service"
      environment = var.environment
    }
  }
}

# ====================================================================
# LOG MANAGEMENT OUTPUTS
# ====================================================================

output "drop_rule_ids" {
  description = "IDs of created drop rules"
  value = {
    debug_logs = last9_drop_rule.drop_debug_logs.id
    test_logs  = var.environment != "test" ? try(last9_drop_rule.drop_test_logs[0].id, null) : null
  }
}

output "forward_rule_ids" {
  description = "IDs of created forward rules"
  value = {
    critical_errors    = var.external_log_destination != "" ? try(last9_forward_rule.forward_critical_errors[0].id, null) : null
    security_logs      = var.security_log_destination != "" ? try(last9_forward_rule.forward_security_logs[0].id, null) : null
    distributed_traces = var.trace_destination != "" ? try(last9_forward_rule.forward_distributed_traces[0].id, null) : null
  }
}

output "log_management_summary" {
  description = "Summary of log management configuration"
  value = {
    region = var.region
    drop_rules = {
      debug_logs = {
        name      = last9_drop_rule.drop_debug_logs.name
        telemetry = last9_drop_rule.drop_debug_logs.telemetry
      }
      test_logs = var.environment != "test" ? {
        name      = try(last9_drop_rule.drop_test_logs[0].name, null)
        telemetry = try(last9_drop_rule.drop_test_logs[0].telemetry, null)
      } : null
    }
    forward_rules = {
      critical_errors_enabled = var.external_log_destination != ""
      security_logs_enabled   = var.security_log_destination != ""
      traces_enabled          = var.trace_destination != ""
    }
  }
}

# ====================================================================
# INTEGRATION TEST OUTPUTS
# ====================================================================

output "integration_test_summary" {
  description = "Complete summary for integration test validation"
  value = {
    environment  = var.environment
    service_name = var.service_name
    region       = var.region

    # Resource counts
    resources_created = {
      dashboards    = 1
      alerts        = 3
      macros        = 1
      policies      = 1
      drop_rules    = var.environment != "test" ? 2 : 1
      forward_rules = (var.external_log_destination != "" ? 1 : 0) + (var.security_log_destination != "" ? 1 : 0) + (var.trace_destination != "" ? 1 : 0)
    }

    # Configuration validation
    configuration = {
      authentication_method = var.last9_refresh_token != "" ? "refresh_token" : "api_token"
      api_base_url          = var.last9_api_base_url
      thresholds = {
        error_rate_percent   = var.error_rate_threshold
        availability_percent = var.availability_threshold
        response_time_ms     = var.response_time_threshold
      }
      alert_timing = {
        bad_minutes   = var.alert_bad_minutes
        total_minutes = var.alert_total_minutes
      }
    }

    # External integrations
    integrations = {
      runbook_configured = var.runbook_base_url != ""
      log_forwarding = {
        critical_errors = var.external_log_destination != ""
        security_logs   = var.security_log_destination != ""
        traces          = var.trace_destination != ""
      }
    }
  }
}

# ====================================================================
# VALIDATION OUTPUTS
# ====================================================================

output "validation_urls" {
  description = "URLs for validating the created resources"
  value = {
    dashboard_url = "https://console.last9.io/dashboards/${last9_dashboard.production_monitoring.id}"
    alerts_url    = "https://console.last9.io/alerts"
    policies_url  = "https://console.last9.io/policies"
    logs_url      = "https://console.last9.io/logs"
  }
}

output "terraform_validation" {
  description = "Information for Terraform validation"
  value = {
    provider_version  = "~> 1.0"
    terraform_version = ">= 1.0"
    resources_managed = [
      "last9_dashboard.production_monitoring",
      "last9_alert.high_error_rate",
      "last9_alert.low_availability",
      "last9_alert.high_response_time",
      "last9_macro.environment_macros",
      "last9_policy.slo_compliance",
      "last9_drop_rule.drop_debug_logs",
      # Conditional resources based on variables
    ]
  }
}

# ====================================================================
# DATA SOURCE OUTPUTS
# ====================================================================

output "entity_info" {
  description = "Information about the monitored entity"
  value = {
    entity_id   = data.last9_entity.production_api.id
    entity_name = data.last9_entity.production_api.name
  }
}