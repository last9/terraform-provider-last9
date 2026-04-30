terraform {
  required_providers {
    last9 = {
      source  = "last9/last9"
      version = "~> 0.2"
    }
  }
}

provider "last9" {
  refresh_token        = var.last9_refresh_token
  api_token            = var.last9_api_token
  delete_refresh_token = var.last9_delete_refresh_token
  delete_token         = var.last9_delete_token
  org                  = var.last9_org
  api_base_url         = var.last9_api_base_url
}

variable "last9_refresh_token" {
  type        = string
  description = "Last9 refresh token (recommended)"
  sensitive   = true
  default     = ""
}

variable "last9_api_token" {
  type        = string
  description = "Last9 API access token (legacy)"
  sensitive   = true
  default     = ""
}

variable "last9_delete_refresh_token" {
  type        = string
  description = "Last9 delete-scope refresh token"
  sensitive   = true
  default     = ""
}

variable "last9_delete_token" {
  type        = string
  description = "Last9 delete-scope API token (legacy)"
  sensitive   = true
  default     = ""
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

variable "region" {
  type        = string
  description = "Region for remapping rules"
  default     = "ap-south-1"
}

# logs_extract with regex pattern - extract named capture groups from log lines
resource "last9_remapping_rule" "logs_extract_pattern" {
  region            = var.region
  type              = "logs_extract"
  name              = "TF Example - Pattern Extract"
  extract_type      = "pattern"
  action            = "upsert"
  remap_keys        = ["(?P<level>\\w+)\\s+(?P<message>.+)"]
  target_attributes = "log_attributes"
}

# logs_extract with JSON parser - extract fields from a JSON-formatted body, gated on a precondition
resource "last9_remapping_rule" "logs_extract_json" {
  region            = var.region
  type              = "logs_extract"
  name              = "TF Example - JSON Extract"
  extract_type      = "json"
  action            = "upsert"
  remap_keys        = ["attributes[\"json_body\"]"]
  target_attributes = "log_attributes"

  preconditions {
    key      = "severity"
    value    = "error"
    operator = "equals"
  }
}

# logs_map - promote an attribute to a standard field
resource "last9_remapping_rule" "logs_map_service" {
  region            = var.region
  type              = "logs_map"
  name              = "TF Example - Map svc to service"
  remap_keys        = ["svc", "app_name"]
  target_attributes = "service"
}

# traces_map - same idea for traces
resource "last9_remapping_rule" "traces_map_service" {
  region            = var.region
  type              = "traces_map"
  name              = "TF Example - Map service.name"
  remap_keys        = ["service.name", "k8s.deployment.name"]
  target_attributes = "service"
}

output "logs_extract_pattern_id" {
  value = last9_remapping_rule.logs_extract_pattern.id
}

output "logs_extract_json_id" {
  value = last9_remapping_rule.logs_extract_json.id
}

output "logs_map_service_id" {
  value = last9_remapping_rule.logs_map_service.id
}

output "traces_map_service_id" {
  value = last9_remapping_rule.traces_map_service.id
}
