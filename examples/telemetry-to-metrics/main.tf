terraform {
  required_providers {
    last9 = {
      source = "last9/last9"
    }
  }
}

provider "last9" {
  # org, refresh_token, and api_base_url can also be set via
  # LAST9_ORG, LAST9_REFRESH_TOKEN, and LAST9_API_BASE_URL env vars.
}

variable "region" {
  type    = string
  default = "ap-south-1"
}

# Logs to metrics (L2M): count error logs per service.
# The aggregation (filter + aggregate + groupby + window) lives in `query`.
resource "last9_logs_to_metrics" "errors_by_service" {
  region           = var.region
  name             = "errors_by_service"
  query_type       = "logjson-aggregate"
  metric_name      = "log_errors_by_service"
  search_frequency = 60

  query = jsonencode([
    {
      type  = "filter"
      query = { "$and" = [{ "$eq" = ["level", "error"] }] }
    },
    {
      type       = "aggregate"
      aggregates = [{ function = { "$count" = [] }, as = "log_errors_by_service" }]
      groupby    = { "attributes['service']" = "service" }
      window     = ["1", "minutes"]
    }
  ])
}

# Traces to metrics (T2M): count probe spans per check, grouped by location.
resource "last9_traces_to_metrics" "probe_up" {
  region           = "ap-southeast-1"
  name             = "probe_up"
  query_type       = "tracejson-aggregate"
  metric_name      = "probe_up"
  search_frequency = 60

  query = jsonencode([
    {
      type  = "filter"
      query = { "$and" = [{ "$regex" = ["ServiceName", "probe-.*"] }] }
    },
    {
      type       = "aggregate"
      aggregates = [{ function = { "$count" = [] }, as = "probe_up" }]
      groupby = {
        "attributes['check_name']" = "check_name"
        "attributes['location']"   = "location"
      }
      window = ["1", "minutes"]
    }
  ])
}
