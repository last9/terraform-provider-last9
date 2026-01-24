# ====================================================================
# PROVIDER CONFIGURATION
# ====================================================================

variable "last9_api_token" {
  description = "Last9 API access token"
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
  default     = "https://app.last9.io"
}

variable "last9_delete_token" {
  description = "Last9 API token with delete scope"
  type        = string
  sensitive   = true
  default     = ""
}

# ====================================================================
# ENVIRONMENT CONFIGURATION
# ====================================================================

variable "environment" {
  description = "Environment name (e.g., production, staging)"
  type        = string
  default     = "integration-test"
}

variable "region" {
  description = "Region for control plane rules"
  type        = string
  default     = "ap-south-1"
}

# ====================================================================
# ALERT THRESHOLDS
# ====================================================================

variable "error_rate_threshold" {
  description = "Error rate threshold percentage for alerts"
  type        = number
  default     = 5.0
}

variable "availability_threshold" {
  description = "Minimum availability percentage threshold"
  type        = number
  default     = 99.5
}

variable "alert_bad_minutes" {
  description = "Minutes the condition must be true before firing"
  type        = number
  default     = 5
}

variable "alert_total_minutes" {
  description = "Evaluation window in minutes"
  type        = number
  default     = 10
}

# ====================================================================
# OPTIONAL CONFIGURATION
# ====================================================================

variable "runbook_base_url" {
  description = "Base URL for runbooks (optional)"
  type        = string
  default     = ""
}


variable "webhook_url" {
  description = "Webhook URL for generic webhook notification channel and forward rule tests"
  type        = string
  default     = ""  # Will be generated with unique timestamp
}

variable "slack_webhook_url" {
  description = "Slack webhook URL for notification channel test"
  type        = string
  default     = ""  # Will be generated with unique timestamp
}

variable "pagerduty_integration_key" {
  description = "PagerDuty integration key for notification channel test"
  type        = string
  default     = ""  # Will be generated with unique timestamp
}

variable "alert_email" {
  description = "Email address for notification channel test (must be company domain, not personal)"
  type        = string
  default     = ""  # Will be generated with unique timestamp
}
