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

output "drop_rule_ids" {
  description = "IDs of created drop rules"
  value = {
    logs    = last9_drop_rule.drop_debug_logs.id
    traces  = last9_drop_rule.drop_test_traces.id
    metrics = last9_drop_rule.drop_test_metrics.id
  }
}

# ====================================================================
# FORWARD RULE OUTPUTS
# ====================================================================

# ====================================================================
# NOTIFICATION CHANNEL OUTPUTS
# ====================================================================

output "notification_channel_ids" {
  description = "IDs of created notification channels"
  value = {
    webhook   = last9_notification_channel.webhook.id
    slack     = last9_notification_channel.slack.id
    pagerduty = last9_notification_channel.pagerduty.id
    email     = last9_notification_channel.email.id
  }
}

# ====================================================================
# FORWARD RULE OUTPUTS
# ====================================================================

output "forward_rule_id" {
  description = "ID of the forward rule"
  value       = last9_forward_rule.forward_critical_errors.id
}

# ====================================================================
# REMAPPING RULE OUTPUTS
# ====================================================================

output "remapping_rule_ids" {
  description = "IDs of created remapping rules"
  value = {
    logs_extract_pattern = last9_remapping_rule.extract_request_id.id
    logs_extract_json    = last9_remapping_rule.extract_json_metadata.id
    logs_map_service     = last9_remapping_rule.map_service_name.id
    logs_map_severity    = last9_remapping_rule.map_severity.id
    traces_map_service   = last9_remapping_rule.map_trace_service.id
  }
}

# ====================================================================
# SCHEDULED SEARCH ALERT OUTPUTS
# ====================================================================

output "scheduled_search_alert_id" {
  description = "ID of the scheduled search alert"
  value       = last9_scheduled_search_alert.high_error_count.id
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
      entities                = 1
      alerts                  = 2
      drop_rules              = 3  # logs, traces, metrics
      notification_channels   = 4  # webhook, slack, pagerduty, email
      forward_rules           = 1
      remapping_rules         = 5  # 2 logs_extract, 2 logs_map, 1 traces_map
      scheduled_search_alerts = 1
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
