# ====================================================================
# PROVIDER CONFIGURATION VARIABLES
# ====================================================================

variable "last9_refresh_token" {
  description = "Last9 refresh token (recommended authentication method)"
  type        = string
  default     = ""
  sensitive   = true
}

variable "last9_api_token" {
  description = "Last9 API token (legacy authentication method)"
  type        = string
  default     = ""
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
  validation {
    condition     = can(regex("^https?://", var.last9_api_base_url))
    error_message = "The api_base_url must be a valid HTTP or HTTPS URL."
  }
}

# ====================================================================
# ENVIRONMENT CONFIGURATION VARIABLES
# ====================================================================

variable "environment" {
  description = "Environment name (e.g., production, staging, development)"
  type        = string
  default     = "integration-test"

  validation {
    condition     = length(var.environment) > 0 && length(var.environment) <= 50
    error_message = "Environment name must be between 1 and 50 characters."
  }
}

variable "region" {
  description = "AWS region or deployment region"
  type        = string
  default     = "us-west-2"
}

variable "service_name" {
  description = "Name of the primary service being monitored"
  type        = string
  default     = "api-service"
}

variable "team_name" {
  description = "Team responsible for this service"
  type        = string
  default     = "platform"
}

variable "cluster_id" {
  description = "Last9 cluster ID for macros"
  type        = string
}

variable "entity_name" {
  description = "Entity name to monitor (will be looked up via data source)"
  type        = string
  default     = "production-api"
}

# ====================================================================
# ALERT THRESHOLD VARIABLES
# ====================================================================

variable "error_rate_threshold" {
  description = "Error rate threshold percentage to trigger alerts"
  type        = number
  default     = 5.0

  validation {
    condition     = var.error_rate_threshold >= 0 && var.error_rate_threshold <= 100
    error_message = "Error rate threshold must be between 0 and 100."
  }
}

variable "availability_threshold" {
  description = "Availability threshold percentage (alerts when below this value)"
  type        = number
  default     = 99.5

  validation {
    condition     = var.availability_threshold >= 0 && var.availability_threshold <= 100
    error_message = "Availability threshold must be between 0 and 100."
  }
}

variable "response_time_threshold" {
  description = "P95 response time threshold in milliseconds"
  type        = number
  default     = 500

  validation {
    condition     = var.response_time_threshold > 0
    error_message = "Response time threshold must be greater than 0."
  }
}

variable "alert_bad_minutes" {
  description = "Number of minutes the condition must be bad before alerting"
  type        = number
  default     = 5

  validation {
    condition     = var.alert_bad_minutes >= 1 && var.alert_bad_minutes <= 60
    error_message = "Alert bad minutes must be between 1 and 60."
  }
}

variable "alert_total_minutes" {
  description = "Total evaluation window in minutes for alerts"
  type        = number
  default     = 10

  validation {
    condition     = var.alert_total_minutes >= 1 && var.alert_total_minutes <= 1440
    error_message = "Alert total minutes must be between 1 and 1440 (24 hours)."
  }
}

# ====================================================================
# EXTERNAL INTEGRATION VARIABLES
# ====================================================================

variable "runbook_base_url" {
  description = "Base URL for runbook documentation"
  type        = string
  default     = ""
}

variable "external_log_destination" {
  description = "External destination URL for forwarding critical logs"
  type        = string
  default     = ""

  validation {
    condition     = var.external_log_destination == "" || can(regex("^https?://", var.external_log_destination))
    error_message = "External log destination must be empty or a valid HTTP/HTTPS URL."
  }
}

variable "security_log_destination" {
  description = "External destination URL for forwarding security logs"
  type        = string
  default     = ""

  validation {
    condition     = var.security_log_destination == "" || can(regex("^https?://", var.security_log_destination))
    error_message = "Security log destination must be empty or a valid HTTP/HTTPS URL."
  }
}

variable "trace_destination" {
  description = "External destination URL for forwarding distributed traces"
  type        = string
  default     = ""

  validation {
    condition     = var.trace_destination == "" || can(regex("^https?://", var.trace_destination))
    error_message = "Trace destination must be empty or a valid HTTP/HTTPS URL."
  }
}

# ====================================================================
# FEATURE TOGGLE VARIABLES
# ====================================================================

variable "enable_log_forwarding" {
  description = "Enable log forwarding rules"
  type        = bool
  default     = true
}

variable "enable_debug_log_dropping" {
  description = "Enable dropping of debug logs to reduce costs"
  type        = bool
  default     = true
}

variable "enable_advanced_alerts" {
  description = "Enable advanced alert rules (expression-based)"
  type        = bool
  default     = true
}

# ====================================================================
# TAGGING VARIABLES
# ====================================================================

variable "additional_tags" {
  description = "Additional tags to apply to resources"
  type        = list(string)
  default     = []
}

variable "cost_center" {
  description = "Cost center for billing/accounting purposes"
  type        = string
  default     = ""
}