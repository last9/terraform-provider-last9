---
page_title: "last9_dashboard Resource - Last9"
subcategory: ""
description: |-
  Manages a Last9 custom dashboard with panels, queries, variables, and layout grid.
---

# last9_dashboard (Resource)

Manages a Last9 custom dashboard. Dashboards organize panels (timeseries, stat, bar, table, section) over a time range with optional template variables. Each panel runs queries against metrics (PromQL), logs (LogQL or JSON pipeline), or traces (TraceQL or JSON pipeline) and renders the result with type-specific visualization config.

## Example Usage

### Stat Panel with PromQL

```terraform
resource "last9_dashboard" "memory" {
  region        = "ap-south-1"
  name          = "Container Memory"
  relative_time = 60 # last 1 hour

  panel {
    name = "Container Memory Usage"
    unit = "bytes-iec"

    layout {
      x = 0
      y = 0
      w = 6
      h = 6
    }

    visualization {
      type = "stat"

      stat_config {
        threshold {
          value = 0
          color = "#22c55e"
        }
        threshold {
          value = 1000000000 # 1 GB
          color = "#ef4444"
        }
      }
    }

    query {
      name             = "A"
      expr             = "avg(container_memory_usage_bytes)"
      telemetry        = "metrics"
      query_type       = "promql"
      legend_type      = "auto"
      legend_placement = "right"
    }
  }
}
```

### Multi-Panel Dashboard with Sections and Variables

```terraform
resource "last9_dashboard" "aws_cost" {
  region        = "ap-south-1"
  name          = "AWS Cost Explorer"
  relative_time = 10080 # last 7 days

  metadata {
    category = "custom"
    type     = "metrics"
    tags     = ["aws", "cost"]
  }

  variable {
    display_name   = "Account"
    target         = "account"
    type           = "label"
    source         = "aws_account_id"
    matches        = ["aws_cost_unblended_USD{cost_date!=\"\"}"]
    multiple       = true
    current_values = [".*"]
  }

  panel {
    name = "Spend at a Glance"
    visualization {
      type       = "section"
      full_width = true
    }
  }

  panel {
    name = "Total Spend"
    unit = "USD"

    layout {
      x = 0
      y = 0
      w = 3
      h = 6
    }

    visualization {
      type = "stat"
    }

    query {
      name       = "A"
      expr       = "sum(aws_cost_unblended_USD{aws_account_id=~\"$account\"})"
      telemetry  = "metrics"
      query_type = "promql"
    }
  }

  panel {
    name = "Cost by Service"
    unit = "USD"

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
      name             = "A"
      expr             = "sum by (aws_service) (aws_cost_unblended_USD)"
      telemetry        = "metrics"
      query_type       = "promql"
      legend_type      = "custom"
      legend_value     = "{{aws_service}}"
      legend_placement = "bottom"
    }
  }
}
```

### Logs Panel with LogQL

```terraform
resource "last9_dashboard" "logs" {
  region        = "ap-south-1"
  name          = "Logs Per Service"
  relative_time = 60

  panel {
    name      = "Service Log Volume"
    telemetry = "logs"

    layout {
      x = 0
      y = 0
      w = 12
      h = 6
    }

    visualization {
      type = "timeseries"

      timeseries_config {
        display_type = "line"
      }
    }

    query {
      name             = "A"
      expr             = "sum by (service) (count_over_time({service=~\".+\"} [1m]))"
      telemetry        = "logs"
      query_type       = "log_ql"
      legend_type      = "custom"
      legend_value     = "{{service}}"
      legend_placement = "right"
    }
  }
}
```

### Traces Table with JSON Pipeline

```terraform
resource "last9_dashboard" "failing_spans" {
  region        = "ap-south-1"
  name          = "Failing Spans by Service"
  relative_time = 60

  panel {
    name      = "Errors"
    telemetry = "traces"

    layout {
      x = 0
      y = 0
      w = 12
      h = 6
    }

    visualization {
      type = "table"

      table_config_json = jsonencode({
        density          = "compact"
        showColumnFilter = true
        showSummary      = false
        transpose        = false
      })
    }

    query {
      name       = "A"
      expr       = jsonencode([
        {
          query = {
            "$and" = [
              { "$eq" = ["SpanKind", "SPAN_KIND_CLIENT"] },
              { "$neq" = ["StatusCode", "STATUS_CODE_OK"] }
            ]
          }
          type = "filter"
        },
        {
          type = "aggregate"
          aggregates = [
            {
              function = { "$count" = [] }
              as       = "_count"
            }
          ]
          groupby = {
            ServiceName = "service"
            StatusCode  = "status"
          }
        }
      ])
      telemetry  = "traces"
      query_type = "trace_json"
    }
  }
}
```

## Schema

### Required

- `region` (String) Region used to look up active integrations for query rendering. Not stored with the dashboard, so changing it does not recreate the resource.
- `name` (String) Dashboard name.
- `panel` (Block List, Min: 1) Ordered list of panels. Sections and panels interleave at the position written. See [Panel](#panel).

### Optional

- `relative_time` (Number) Relative time window in minutes (e.g., `60` = last 1 hour, `10080` = last 7 days). Mutually exclusive with `absolute_from`/`absolute_to`.
- `absolute_from` (Number) Absolute time range start (Unix millis). Must be set together with `absolute_to`. Mutually exclusive with `relative_time`.
- `absolute_to` (Number) Absolute time range end (Unix millis). Must be set together with `absolute_from`.
- `metadata` (Block List, Max: 1) Dashboard metadata. See [Metadata](#metadata).
- `variable` (Block List) Template variables propagated into queries via `$varname`. See [Variable](#variable).

### Read-Only

- `id` (String) Dashboard UUID.
- `created_at` (Number) Creation timestamp (Unix seconds).
- `updated_at` (Number) Last-update timestamp (Unix seconds).
- `created_by` (String) UUID of the user who created the dashboard.
- `readonly` (Boolean) Whether the dashboard is system-managed and cannot be edited.

### Panel

The `panel` block supports:

- `name` (String, Required) Panel title.
- `visualization` (Block List, Max: 1, Required) Visualization config. See [Visualization](#visualization).
- `layout` (Block List, Max: 1) Position and size in the grid. Required for all non-section panels.
- `query` (Block List) Queries powering the panel. Required for non-section panels. See [Query](#query).
- `datasource_id` (String) Datasource UUID.
- `telemetry` (String) Panel-level default telemetry (`metrics` | `logs` | `traces`).
- `unit` (String) Display unit (e.g., `USD`, `bytes-iec`, `ms`).
- `version` (Number) Panel schema version. Defaults to `1`. Setting to `0` causes the API to silently drop `query.telemetry` and `query.query_type` — leave as default unless you have a specific reason.
- `id` (String, Read-Only) Server-assigned UUID. Round-tripped on update so external panel-level URLs (embeds, drill-downs) stay stable across edits.

#### Layout

The `layout` block supports:

- `x` (Number, Required) Column index.
- `y` (Number, Required) Row index.
- `w` (Number, Required) Width in grid units.
- `h` (Number, Required) Height in grid units.
- `extra_json` (String) Reserved opaque JSON for layout fields the API may add (e.g., `minH`, `static`, `i`). Round-tripped verbatim. Use `jsonencode()` if you need to set keys beyond `x`/`y`/`w`/`h`. Invalid JSON fails at plan time.

#### Visualization

The `visualization` block supports:

- `type` (String, Required) One of `timeseries`, `stat`, `bar`, `table`, `section`.
- `full_width` (Boolean) Whether the panel spans full dashboard width.
- `timeseries_config` (Block List, Max: 1) Optional config for timeseries. Has `display_type` (`line` | `area`).
- `bar_config` (Block List, Max: 1) Optional config for bar. Has `orientation` (`vertical` | `horizontal`) and `stacked` (Boolean).
- `stat_config` (Block List, Max: 1) Optional config for stat. Has `threshold` (Block List) entries with `value` (Float) and `color` (String).
- `table_config_json` (String) Raw `table_config` JSON. The backend stores this as an untyped blob (`columnConfig`, `density`, `thresholds`, `transpose`, etc.); use `jsonencode({...})` so any field set in the UI round-trips verbatim. Invalid JSON fails at plan time.

Section panels (`type = "section"`) must have no `query` and no `layout` blocks.

#### Query

The `query` block supports:

- `name` (String, Required) Query identifier (e.g., `A`, `B`).
- `expr` (String, Required) Query expression. PromQL for `query_type = "promql"`, LogQL for `log_ql`, or a serialized JSON pipeline for `log_json` / `trace_json`. The provider treats `expr` as opaque.
- `type` (String) Query type. Defaults to `range`.
- `unit` (String) Display unit.
- `telemetry` (String) `metrics` | `logs` | `traces`. Required when panel `version >= 1`.
- `query_type` (String) Query language. Required when panel `version >= 1`. Valid combinations:
  - `metrics` → `promql`
  - `logs` → `log_ql`, `log_json`
  - `traces` → `trace_ql`, `trace_json`
- `legend_type` (String) `auto` | `custom`. Defaults to `auto`.
- `legend_value` (String) Legend template (e.g., `{{service}}`). Used when `legend_type = "custom"`.
- `legend_placement` (String) `bottom` | `left` | `right`. Defaults to `bottom`.
- `legend_sort_field` (String) Field to sort legend entries by.
- `legend_sort_direction` (String) `asc` | `desc`.
- `matrix_json` (String) Raw matrix JSON for query result transformation. Opaque blob.

### Variable

The `variable` block supports:

- `display_name` (String, Required) Label shown above the variable selector.
- `target` (String, Required) Variable name used in queries (e.g., `account` for `$account`).
- `type` (String, Required) `label` (values fetched from a metric label) or `static` (values supplied inline).
- `source` (String) Label name to fetch values from. Required when `type = "label"`.
- `matches` (List of String) Selector(s) for fetching label values, e.g., `["my_metric{env=~\"$env\"}"]`.
- `values` (List of String) Static values. Required when `type = "static"`.
- `multiple` (Boolean) Allow multi-select.
- `internal` (Boolean) Hide from dashboard UI controls.
- `current_values` (List of String) Selected values. Initial values honored on create. Subsequent UI selections are read back into state on refresh, so plan won't show drift on UI changes — but if you keep a value in HCL, the next apply will revert UI selections to your HCL value. Omit from HCL to let the UI own selection state.

### Metadata

The `metadata` block supports:

- `category` (String) Defaults to `custom`.
- `type` (String) Defaults to `metrics`.
- `tags` (List of String) Free-form tags for filtering dashboards.

## Panel Reorder Caveat

Panels are matched by index across reads. If you reorder `panel { ... }` blocks in HCL, panel UUIDs swap with their neighbors — the panel at index 0 keeps its UUID and gains panel 2's content. External references to panel UUIDs (embed URLs, drill-down links) will then point at the wrong panel. To reorder visually, change the `layout.y` coordinate instead of moving the HCL block.

## Import

Dashboards can be imported using the format `region:dashboard_id`:

```shell
terraform import last9_dashboard.example ap-south-1:1384eae9-a90e-4e23-85ec-428d73911556
```

The composite import ID is required because the GET API takes `region` as a query parameter.
