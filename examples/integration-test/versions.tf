# versions.tf - Local Development Configuration
# Use this file for local testing before the provider is published to the Terraform Registry

# For local development and testing
terraform {
  required_providers {
    last9 = {
      source = "hashicorp.com/edu/last9"
    }
  }
  required_version = ">= 1.0"
}

# Uncomment and modify the below for published provider usage:
# terraform {
#   required_providers {
#     last9 = {
#       source  = "last9/last9"
#       version = "~> 1.0"
#     }
#   }
#   required_version = ">= 1.0"
# }