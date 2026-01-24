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

variable "last9_api_token" {
  type        = string
  description = "Last9 API access token"
  sensitive   = true
}

variable "last9_org" {
  type        = string
  description = "Last9 organization slug"
}

variable "last9_api_base_url" {
  type        = string
  description = "Last9 API base URL"
  default     = "https://app.last9.io"
}

variable "last9_delete_token" {
  type        = string
  description = "Last9 API token with delete scope"
  sensitive   = true
  default     = ""
}

# Data source to lookup notification destinations
data "last9_notification_destination" "slack_alerts" {
  name = "testing-integrations"
}

data "last9_notification_destination" "pagerduty" {
  name = "PD Test"
}

# Example 1: Error count alert
resource "last9_scheduled_search_alert" "high_error_count" {
  region         = "ap-south-1"
  name           = "High Error Count Alert"
  query_type     = "logjson-aggregate"
  physical_index = "logs"
  telemetry      = "logs"

  # Query: Filter for ERROR severity logs
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

  # Post-processor: Count errors
  post_processor {
    type = "aggregate"

    aggregates {
      function = jsonencode({ "$count" = [] })
      as       = "error_count"
    }

    groupby = "{}"
  }

  # Run every 5 minutes
  search_frequency = 300

  # Alert when error count > 100
  threshold {
    operator = ">"
    value    = 100
  }

  # Send to Slack and PagerDuty
  alert_destinations = [
    data.last9_notification_destination.slack_alerts.id,
    data.last9_notification_destination.pagerduty.id
  ]
}

# Example 2: Service-specific alert with grouping
resource "last9_scheduled_search_alert" "api_error_spike" {
  region         = "ap-south-1"
  name           = "API Service Error Spike"
  query_type     = "logjson-aggregate"
  physical_index = "logs"
  telemetry      = "logs"

  # Query: Filter for API service errors
  query = jsonencode([
    {
      type = "filter"
      query = {
        "$and" = [
          { "$eq" = ["SeverityText", "ERROR"] },
          { "$eq" = ["attributes.service", "api-service"] }
        ]
      }
    }
  ])

  # Post-processor: Count errors grouped by endpoint
  post_processor {
    type = "aggregate"

    aggregates {
      function = jsonencode({ "$count" = [] })
      as       = "error_count"
    }

    # Group by endpoint
    groupby = jsonencode({
      "endpoint" = ["attributes.endpoint"]
    })
  }

  # Check every 2 minutes
  search_frequency = 120

  # Alert when error count >= 50
  threshold {
    operator = ">="
    value    = 50
  }

  alert_destinations = [
    data.last9_notification_destination.pagerduty.id
  ]
}

# Example 3: Critical log detection
resource "last9_scheduled_search_alert" "critical_logs" {
  region         = "ap-south-1"
  name           = "Critical Log Detection"
  query_type     = "logjson-aggregate"
  physical_index = "logs"
  telemetry      = "logs"

  # Query: Filter for CRITICAL severity
  query = jsonencode([
    {
      type = "filter"
      query = {
        "$and" = [
          { "$eq" = ["SeverityText", "CRITICAL"] }
        ]
      }
    }
  ])

  # Post-processor: Count critical logs
  post_processor {
    type = "aggregate"

    aggregates {
      function = jsonencode({ "$count" = [] })
      as       = "critical_count"
    }

    groupby = "{}"
  }

  # Check every minute for critical issues
  search_frequency = 60

  # Alert on ANY critical log (> 0)
  threshold {
    operator = ">"
    value    = 0
  }

  alert_destinations = [
    data.last9_notification_destination.slack_alerts.id,
    data.last9_notification_destination.pagerduty.id
  ]
}

# Example 4: Low activity alert (using < operator)
resource "last9_scheduled_search_alert" "low_activity" {
  region         = "ap-south-1"
  name           = "Low Activity Alert"
  query_type     = "logjson-aggregate"
  physical_index = "logs"
  telemetry      = "logs"

  # Query: Count all logs from production service
  query = jsonencode([
    {
      type = "filter"
      query = {
        "$and" = [
          { "$eq" = ["attributes.environment", "production"] },
          { "$eq" = ["attributes.service", "payment-service"] }
        ]
      }
    }
  ])

  # Post-processor: Count logs
  post_processor {
    type = "aggregate"

    aggregates {
      function = jsonencode({ "$count" = [] })
      as       = "log_count"
    }

    groupby = "{}"
  }

  # Check every 10 minutes
  search_frequency = 600

  # Alert when activity is too low (< 10 logs in 10 minutes)
  threshold {
    operator = "<"
    value    = 10
  }

  alert_destinations = [
    data.last9_notification_destination.slack_alerts.id
  ]
}

# Outputs
output "high_error_alert_id" {
  description = "ID of the high error count alert"
  value       = last9_scheduled_search_alert.high_error_count.id
}

output "api_error_alert_id" {
  description = "ID of the API error spike alert"
  value       = last9_scheduled_search_alert.api_error_spike.id
}

output "slack_destination_id" {
  description = "ID of the testing-integrations notification destination"
  value       = data.last9_notification_destination.slack_alerts.id
}

output "pagerduty_destination_id" {
  description = "ID of the PD test notification destination"
  value       = data.last9_notification_destination.pagerduty.id
}
