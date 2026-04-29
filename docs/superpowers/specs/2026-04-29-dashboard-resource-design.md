# Design: `last9_dashboard` Terraform Resource

**Date:** 2026-04-29  
**Linear:** ENG-1013  
**Status:** Approved

---

## Overview

Add `last9_dashboard` resource to the Last9 Terraform provider. Enables IaC management of custom dashboards via `/api/v4/organizations/{org}/dashboards`.

---

## API Surface

| Operation | Method | Path | Notes |
|---|---|---|---|
| Create | POST | `/dashboards` | Body: `{dashboard, metadata}` |
| Read | GET | `/dashboards/{id}?region={region}` | region required |
| Update | PUT | `/dashboards/{id}` | Full replace |
| Delete | DELETE | `/dashboards/{id}` | Uses delete token |

Response shape: `{dashboard: {...}, metadata: {...}}` for create/update; same for read.

Metadata hardcoded on create/update: `{_category: "custom", _type: "logs"}`.

---

## Schema

### Top-level

| Field | Type | Required | Computed | Notes |
|---|---|---|---|---|
| `name` | string | yes | — | dashboard display name |
| `region` | string | yes | — | stored in state; needed for GET |
| `tags` | list(string) | — | — | optional metadata tags |
| `panels` | list(block) | — | — | ordered; TypeList preserves order |
| `variables` | list(block) | — | — | |
| `id` | string | — | yes | UUID assigned by API |
| `created_by` | string | — | yes | |
| `created_at` | int64 | — | yes | Unix timestamp |
| `updated_at` | int64 | — | yes | Unix timestamp |
| `readonly` | bool | — | yes | always false for user dashboards |

### `panels` block

| Field | Type | Required | Computed | Notes |
|---|---|---|---|---|
| `name` | string | yes | — | unique per dashboard |
| `datasource_id` | string | — | — | required unless `visualization.type = "section"` (CustomizeDiff) |
| `unit` | string | — | — | |
| `layout` | block | — | — | MaxItems=1 |
| `visualization` | block | — | — | MaxItems=1 |
| `queries` | list(block) | — | — | required unless section (CustomizeDiff) |
| `id` | string | — | yes | auto-generated; round-tripped on update |
| `created_at` | int64 | — | yes | |
| `updated_at` | int64 | — | yes | |

**CustomizeDiff on panel:** if `visualization.type != "section"`, then `datasource_id` must be non-empty and `queries` must have at least one entry.

### `panels.layout` block (MaxItems=1)

| Field | Type | Required | Notes |
|---|---|---|---|
| `x` | int | — | grid column |
| `y` | int | — | grid row |
| `w` | int | — | width |
| `h` | int | — | height |

### `panels.visualization` block (MaxItems=1)

| Field | Type | Required | Notes |
|---|---|---|---|
| `type` | string | yes | `chart`, `gauge`, `section` |
| `full_width` | bool | — | |
| `timeseries_config` | block | — | MaxItems=1 |
| `heatmap_config` | block | — | MaxItems=1 |
| `bar_config` | block | — | MaxItems=1 |
| `stat_config` | block | — | MaxItems=1 |
| `status_history_config` | block | — | MaxItems=1 |

**`timeseries_config`:** `display_type` string — `line` or `area`  
**`heatmap_config`:** `thresholds` list(block) — each has `value` float, `color` string  
**`bar_config`:** `orientation` string (`vertical`/`horizontal`), `stacked` bool  
**`stat_config`:** `thresholds` list(block) — each has `value` float, `color` string  
**`status_history_config`:** `thresholds` list(block) — each has `value` float, `color` string, `label` string (optional)

### `panels.queries` block

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | unique per panel |
| `expr` | string | yes | PromQL expression |
| `type` | string | yes | `range` or `instant` (validated) |
| `unit` | string | — | |
| `legend_placement` | string | — | `bottom`, `left`, `right` |
| `legend` | block | — | MaxItems=1; optional but always populated on read |

**`legend` block:** `type` string (`auto`/`custom`), `value` string

### `variables` block

| Field | Type | Required | Notes |
|---|---|---|---|
| `target` | string | yes | `$variable` name in PromQL |
| `display_name` | string | yes | label shown on dashboard |
| `type` | string | yes | `static` or `label` |
| `values` | list(string) | — | for static variables |
| `source` | string | — | for label variables |
| `matches` | list(string) | — | label filter matches |
| `multiple` | bool | — | allow multi-select |
| `internal` | bool | — | hide from dashboard UI |

---

## Import

```
terraform import last9_dashboard.example region:dashboard-uuid
```

Custom `ImportStateContext` splits on `:`, sets `region` and `d.Id()`.

---

## Client Types (to add to `client.go`)

```go
type Dashboard struct { ... }
type DashboardPanel struct { ... }
type DashboardPanelVisualization struct { ... }
type DashboardPanelQueryDetails struct { ... }
type DashboardPanelLegend struct { ... }
type TimeseriesConfig struct { ... }
type HeatmapConfig / BarConfig / StatConfig / StatusHistoryConfig struct { ... }
type DashboardVariable struct { ... }
type DashboardMetadata struct { ... }
type DashboardRequest struct { Dashboard *Dashboard; Metadata *DashboardMetadata }
type DashboardDetails struct { Dashboard *Dashboard; Metadata *DashboardMetadata }
```

Methods on `*Client`:
- `CreateDashboard(req *DashboardRequest) (*DashboardDetails, error)`
- `GetDashboard(id, region string) (*DashboardDetails, error)`
- `UpdateDashboard(id string, req *DashboardRequest) (*DashboardDetails, error)`
- `DeleteDashboard(id string) error`

---

## Files

| File | Action |
|---|---|
| `internal/client/client.go` | Add Dashboard types + 4 methods |
| `internal/provider/resource_dashboard.go` | New resource file |
| `internal/provider/provider.go` | Register `last9_dashboard` in ResourcesMap |
| `examples/resources/last9_dashboard/main.tf` | Usage example |

---

## Key Design Decisions

- **Panel ordering:** TypeList — order is significant, preserved by API
- **Panel IDs:** Computed, round-tripped on update to prevent UUID churn
- **Layout:** Typed block with int fields (not TypeMap with string values)
- **Legend:** Optional block; always populated on read by API's `PopulateDefaults()`
- **`section` panel:** Supported via CustomizeDiff (datasource_id + queries not required)
- **Metadata:** `category=custom`, `type=logs` hardcoded — not exposed in schema
- **`current_values`:** Not exposed (UI runtime state, not IaC config)
- **Import:** Composite `region:id` — region needed for GET

---

## Out of Scope (v1)

- `time` field (relative/absolute time range)
- `query_mapping` (queryset-based panels)
- `embedded_dashboard` token management
- Dashboard shares
