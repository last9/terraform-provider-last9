# Dashboard Examples

End-to-end examples for `last9_dashboard`. Two dashboards covering all v1 visualization types and key patterns.

## What's Inside

**`aws_cost_explorer`** — multi-section dashboard demonstrating:
- `section` panels (dividers without queries/layout)
- `stat` panel with `stat_config` thresholds
- `bar` panel with `bar_config` (vertical, stacked)
- `label`-type variables wired into queries via `$account`, `$region`
- `relative_time` (7 days)
- `metadata` (category, type, tags)

**`mixed_telemetry`** — different telemetry sources demonstrating:
- `timeseries` panel with `timeseries_config` (display_type)
- `timeseries` panel with multiple queries (LogQL across services)
- `table` panel with `table_config` (using JSON-pipeline traces query)
- All three telemetries: metrics (PromQL), logs (LogQL), traces (JSON pipeline)

## Running Locally Against Your Built Provider

Build the provider and use a `dev_overrides` block so Terraform skips the registry:

```bash
# From the repo root
go build -o /tmp/terraform-provider-last9 .
```

Add to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "last9/last9" = "/tmp"
  }
  direct {}
}
```

## Running

```bash
cd examples/dashboards

export LAST9_REFRESH_TOKEN=...
export LAST9_DELETE_REFRESH_TOKEN=...
export LAST9_ORG=your-org

terraform plan \
  -var "last9_refresh_token=$LAST9_REFRESH_TOKEN" \
  -var "last9_delete_refresh_token=$LAST9_DELETE_REFRESH_TOKEN" \
  -var "last9_org=$LAST9_ORG"

terraform apply \
  -var "last9_refresh_token=$LAST9_REFRESH_TOKEN" \
  -var "last9_delete_refresh_token=$LAST9_DELETE_REFRESH_TOKEN" \
  -var "last9_org=$LAST9_ORG" \
  -auto-approve

terraform destroy \
  -var "last9_refresh_token=$LAST9_REFRESH_TOKEN" \
  -var "last9_delete_refresh_token=$LAST9_DELETE_REFRESH_TOKEN" \
  -var "last9_org=$LAST9_ORG" \
  -auto-approve
```

## Importing an Existing Dashboard

```bash
terraform import last9_dashboard.aws_cost_explorer ap-south-1:<dashboard-id>
```

The composite import ID is `region:dashboard_id` — region is required because the GET API takes it as a query param.

## Schema Notes

- `panel` is a `TypeList` (ordered) — section dividers and panels interleave at the position you write them in.
- Panel `id` is `Computed` — set by the API, round-tripped on update so panel-level URLs remain stable.
- `legend` is flattened: `legend_type`, `legend_value`, `legend_placement` instead of a nested block. `legend_sort_field` and `legend_sort_direction` are also flat.
- `expr` is an opaque string. For `query_type = "promql"` it's PromQL; for `log_json`/`trace_json` it's a serialized JSON pipeline (filter → aggregate stages).
- `region` is the query-time scope used to look up active integrations for query rendering. It's not stored with the dashboard, so changing it doesn't recreate the resource — Terraform just refreshes against the new region next plan.
- Time range is exposed as either `relative_time` (minutes) or `absolute_from` + `absolute_to` (Unix millis). The two are mutually exclusive; absolute requires both bounds.
- `variable.current_values` is `Optional + Computed`. Set an initial selection in HCL; subsequent UI changes won't drift back to your default.

### Panel reorder caveat

Panels are matched by index across reads. If you reorder `panel { ... }` blocks in HCL, panel UUIDs swap with their neighbors — the panel at index 0 keeps its UUID, gains panel 2's content. External references to panel UUIDs (embed URLs, drill-down links) will then point at the wrong panel. If you need to reorder panels visually, change the `layout.y` coordinate instead of moving the HCL block.

### Opaque blob fields

The Last9 API stores some fields as untyped JSON blobs. To prevent silent data loss when users edit these in the UI, we expose them as opaque JSON strings rather than typed blocks:

- `visualization.table_config_json` — table panel config (column widths, thresholds, density, etc.)
- `query.matrix_json` — query result transformation rules

Use `jsonencode({...})` so the HCL stays readable. Whatever you put here round-trips verbatim through the API. If you don't manage these in HCL, Terraform won't fight UI edits — but won't restore them either.
