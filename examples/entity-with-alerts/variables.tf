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
