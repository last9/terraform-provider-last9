# ====================================================================
# ENTITY OUTPUTS
# ====================================================================

output "entity_id" {
  description = "ID of the created alert group"
  value       = last9_entity.integration_test.id
}

output "entity_name" {
  description = "Name of the created alert group"
  value       = last9_entity.integration_test.name
}

# ====================================================================
# ALERT OUTPUTS
# ====================================================================

output "alert_ids" {
  description = "IDs of created alerts"
  value = {
    high_error_rate  = last9_alert.high_error_rate.id
    low_availability = last9_alert.low_availability.id
  }
}

# ====================================================================
# DROP RULE OUTPUTS
# ====================================================================

output "drop_rule_id" {
  description = "ID of the drop rule"
  value       = last9_drop_rule.drop_debug_logs.id
}

# ====================================================================
# FORWARD RULE OUTPUTS
# ====================================================================

output "forward_rule_id" {
  description = "ID of the forward rule (if created)"
  value       = var.external_log_destination != "" ? try(last9_forward_rule.forward_critical_errors[0].id, null) : null
}

# ====================================================================
# INTEGRATION TEST SUMMARY
# ====================================================================

output "integration_test_summary" {
  description = "Summary for integration test validation"
  value = {
    environment = var.environment
    region      = var.region

    resources_created = {
      entities      = 1
      alerts        = 2
      drop_rules    = 1
      forward_rules = var.external_log_destination != "" ? 1 : 0
    }

    configuration = {
      api_base_url = var.last9_api_base_url
      thresholds = {
        error_rate_percent   = var.error_rate_threshold
        availability_percent = var.availability_threshold
      }
      alert_timing = {
        bad_minutes   = var.alert_bad_minutes
        total_minutes = var.alert_total_minutes
      }
    }
  }
}
