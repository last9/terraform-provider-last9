# Example: Policy configuration

terraform {
  required_providers {
    last9 = {
      source  = "last9/last9"
      version = "~> 1.0"
    }
  }
}

provider "last9" {
  api_token = var.last9_api_token
  org       = var.last9_org
}

# SLO Compliance Policy
resource "last9_policy" "slo_compliance" {
  name        = "Production SLO Compliance"
  description = "Ensures all production services meet SLO requirements"
  
  filters = {
    entity_type = "service"
    tags        = "production"
  }
  
  rules {
    type = "slo_compliance"
    config = {
      slo_name          = "availability"
      threshold         = "99.9"
      evaluation_window = "30d"
    }
  }
  
  rules {
    type = "slo_compliance"
    config = {
      slo_name          = "latency_p95"
      threshold         = "200ms"
      evaluation_window = "7d"
    }
  }
}

# Alert Coverage Policy
resource "last9_policy" "alert_coverage" {
  name        = "Alert Coverage Policy"
  description = "Ensures all services have adequate alert coverage"
  
  filters = {
    entity_type = "service"
  }
  
  rules {
    type = "alert_coverage"
    config = {
      min_alerts = 3
      required_severities = "breach,threat"
    }
  }
}

# Multi-rule Policy
resource "last9_policy" "comprehensive" {
  name        = "Comprehensive Service Policy"
  description = "Multiple compliance rules for services"
  
  filters = {
    entity_type = "service"
    tags        = "production,monitored"
  }
  
  # SLO Compliance
  rules {
    type = "slo_compliance"
    config = {
      slo_name          = "availability"
      threshold         = "99.95"
      evaluation_window = "30d"
    }
  }
  
  # Alert Coverage
  rules {
    type = "alert_coverage"
    config = {
      min_alerts = 5
    }
  }
  
  # Owner Assignment
  rules {
    type = "owner_assignment"
    config = {
      require_owner = "true"
    }
  }
}

variable "last9_api_token" {
  type      = string
  sensitive = true
}

variable "last9_org" {
  type = string
}

