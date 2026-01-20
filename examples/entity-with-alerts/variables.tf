variable "last9_refresh_token" {
  type        = string
  description = "Last9 refresh token for authentication"
  sensitive   = true
}

variable "last9_org" {
  type        = string
  description = "Last9 organization slug"
}
