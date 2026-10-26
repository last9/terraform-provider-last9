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
  description = "Region for dashboards"
  default     = "ap-south-1"
}

# Example 1: AWS Cost Explorer-style dashboard
# Demonstrates: sections, stat panel with thresholds, bar with bar_config, label
# variables, relative_time, and metadata.
resource "last9_dashboard" "aws_cost_explorer" {
  region        = var.region
  name          = "TF Example - AWS Cost Explorer"
  relative_time = 10080 # last 7 days, in minutes

  metadata {
    category = "custom"
    type     = "metrics"
    tags     = ["terraform", "aws", "cost"]
  }

  # Variables propagate as $var into queries
  variable {
    display_name   = "Account"
    target         = "account"
    type           = "label"
    source         = "aws_account_id"
    matches        = ["aws_cost_unblended_USD{cost_date!=\"\"}"]
    multiple       = true
    current_values = [".*"]
  }

  variable {
    display_name   = "Region"
    target         = "region"
    type           = "label"
    source         = "aws_region"
    matches        = ["aws_cost_unblended_USD{cost_date!=\"\", aws_account_id=~\"$account\"}"]
    multiple       = true
    current_values = [".*"]
  }

  # Section divider
  panel {
    key      = "spend-section"
    position = 0
    name     = "Spend at a Glance"
    visualization {
      type       = "section"
      full_width = true
    }
  }

  # Markdown panel for dashboard notes
  panel {
    key      = "cost-notes"
    position = 1
    name     = "Cost Dashboard Notes"

    layout {
      x = 3
      y = 0
      w = 9
      h = 6
    }

    visualization {
      type = "markdown"

      markdown_config {
        content = <<-EOT
        ### Cost Dashboard Notes

        Review account and region filters before comparing week-over-week cost.
        EOT
      }
    }
  }

  # Stat with thresholds
  panel {
    key      = "total-spend"
    position = 2
    name     = "Total Spend"
    unit     = "USD"

    layout {
      x = 0
      y = 0
      w = 3
      h = 6
    }

    visualization {
      type = "stat"

      stat_config {
        threshold {
          position = 0
          value    = 0
          color    = "#22c55e"
        }
        threshold {
          position = 1
          value    = 1000
          color    = "#eab308"
        }
        threshold {
          position = 2
          value    = 5000
          color    = "#ef4444"
        }
      }
    }

    query {
      position         = 0
      name             = "A"
      expr             = "sum(sum by (cost_date) (last_over_time(aws_cost_unblended_USD{cost_date!=\"\", aws_account_id=~\"$account\", aws_region=~\"$region\"}[7d]))) or vector(0)"
      unit             = "USD"
      telemetry        = "metrics"
      query_type       = "promql"
      legend_type      = "custom"
      legend_value     = "$ total"
      legend_placement = "bottom"
    }
  }

  # Section + bar with stacked vertical bars
  panel {
    key      = "trends-section"
    position = 3
    name     = "Trends"
    visualization {
      type       = "section"
      full_width = true
    }
  }

  panel {
    key      = "cost-by-date-service"
    position = 4
    name     = "Cost by Date and Service"
    unit     = "USD"

    layout {
      x = 0
      y = 1
      w = 12
      h = 8
    }

    visualization {
      type       = "bar"
      full_width = true

      bar_config {
        orientation = "vertical"
        stacked     = true
      }
    }

    query {
      position         = 0
      name             = "A"
      expr             = "sum by (cost_date, aws_service) (last_over_time(aws_cost_unblended_USD{cost_date!=\"\", aws_account_id=~\"$account\", aws_region=~\"$region\"}[7d]))"
      unit             = "USD"
      telemetry        = "metrics"
      query_type       = "promql"
      legend_type      = "custom"
      legend_value     = "{{aws_service}}"
      legend_placement = "bottom"
    }
  }
}

# Example 2: Mixed-telemetry dashboard
# Demonstrates: timeseries, table, multiple queries on a single panel,
# and traces/logs/metrics combined.
resource "last9_dashboard" "mixed_telemetry" {
  region        = var.region
  name          = "TF Example - Mixed Telemetry"
  relative_time = 60 # last 1 hour

  metadata {
    category = "custom"
    type     = "metrics"
    tags     = ["terraform", "demo"]
  }

  # PromQL timeseries with timeseries_config
  panel {
    key       = "jvm-major-gc-rate"
    position  = 0
    name      = "JVM Major GC Rate"
    telemetry = "metrics"

    layout {
      x = 0
      y = 0
      w = 6
      h = 6
    }

    visualization {
      type = "timeseries"

      timeseries_config {
        display_type = "line"
      }
    }

    query {
      position         = 0
      name             = "A"
      expr             = "sum(rate(process_runtime_jvm_gc_duration_seconds_sum{action=\"end of major GC\"}[5m]))"
      telemetry        = "metrics"
      query_type       = "promql"
      legend_type      = "auto"
      legend_placement = "bottom"
    }
  }

  # Multi-query timeseries (LogQL, comparing services)
  panel {
    key       = "logs-per-service"
    position  = 1
    name      = "Logs Per Service"
    telemetry = "logs"

    layout {
      x = 6
      y = 0
      w = 6
      h = 6
    }

    visualization {
      type = "timeseries"
    }

    query {
      position         = 0
      name             = "A"
      expr             = "sum by (service) (count_over_time({service=\"nginx\"} [1m]))"
      telemetry        = "logs"
      query_type       = "log_ql"
      legend_type      = "custom"
      legend_value     = "{{service}}"
      legend_placement = "right"
    }

    query {
      position         = 1
      name             = "B"
      expr             = "sum by (service) (count_over_time({service=\"frontend-proxy\"} [1m]))"
      telemetry        = "logs"
      query_type       = "log_ql"
      legend_type      = "custom"
      legend_value     = "{{service}}"
      legend_placement = "right"
    }
  }

  # Doughnut panel for categorical PromQL results
  panel {
    key       = "http-status-mix"
    position  = 2
    name      = "HTTP Status Mix"
    telemetry = "metrics"

    layout {
      x = 0
      y = 11
      w = 6
      h = 6
    }

    visualization {
      type = "doughnut"
    }

    query {
      position         = 0
      name             = "A"
      expr             = "sum by (status) (http_requests_total)"
      telemetry        = "metrics"
      query_type       = "promql"
      legend_type      = "custom"
      legend_value     = "{{status}}"
      legend_placement = "bottom"
    }
  }

  # Table panel from traces (JSON pipeline as expr)
  panel {
    key       = "failing-spans-by-service"
    position  = 3
    name      = "Failing Spans by Service"
    telemetry = "traces"

    layout {
      x = 0
      y = 6
      w = 12
      h = 5
    }

    visualization {
      type = "table"

      # table_config is opaque JSON. Backend stores as untyped blob.
      # Use jsonencode so the HCL stays readable. Any field set in the UI
      # (column widths, thresholds, etc.) round-trips verbatim.
      table_config_json = jsonencode({
        density          = "compact"
        showColumnFilter = true
        showSummary      = false
        transpose        = false
      })
    }

    query {
      position   = 0
      name       = "A"
      expr       = "[{\"query\":{\"$and\":[{\"$eq\":[\"SpanKind\",\"SPAN_KIND_CLIENT\"]},{\"$neq\":[\"StatusCode\",\"STATUS_CODE_OK\"]}]},\"type\":\"filter\"},{\"type\":\"aggregate\",\"aggregates\":[{\"function\":{\"$count\":[]},\"as\":\"_count\"}],\"groupby\":{\"ServiceName\":\"service\",\"StatusCode\":\"status\"}}]"
      telemetry  = "traces"
      query_type = "trace_json"
    }
  }
}

output "aws_cost_explorer_id" {
  value       = last9_dashboard.aws_cost_explorer.id
  description = "Created AWS Cost Explorer dashboard ID"
}

output "mixed_telemetry_id" {
  value       = last9_dashboard.mixed_telemetry.id
  description = "Created Mixed Telemetry dashboard ID"
}
