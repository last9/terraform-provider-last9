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
- `legend` is flattened: `legend_type`, `legend_value`, `legend_placement` instead of a nested block.
- `expr` is an opaque string. For `query_type = "promql"` it's PromQL; for `log_json`/`trace_json` it's a serialized JSON pipeline (filter → aggregate stages).
