---
page_title: "Last9 Terraform Provider Coverage"
description: |-
  Coverage matrix of Last9 control-plane APIs vs Terraform provider vs Datadog vs l9iac.
---

# Coverage Matrix

Once-and-for-all customer IaC coverage for the Last9 Terraform provider.
Ground truth for API routes: `last9-api/api/routes_v4.go` (OpenAPI is incomplete).

## Scope decisions

| In scope | Out of scope |
|----------|--------------|
| Alerting (entities, metric alerts, scheduled search, snooze) | SLOs / slo_detectors |
| Notifications | Macros |
| Dashboards | Entity relationships |
| Synthetics | Levitate tenants / sources / tokens / access policies |
| Changeboards | Dashboard shares / snapshots / preferences |
| Users + roles | Anomaly catalog, component templates, Grafana bridge |
| OTel pipeline (drop, forward, remapping, sensitive data, rehydration, physical index) | Alert episode claims / live inventory |
| Cluster + datasource lookups | |

## Resource coverage

| Domain | Last9 API | Terraform | l9iac | Datadog analogue | Status |
|--------|-----------|-----------|-------|------------------|--------|
| Alert groups | `/entities` | `last9_entity` | entity | service catalog / monitor grouping | ✅ |
| Metric alerts | `/entities/{id}/alert-rules` + KPIs | `last9_alert` | alert | `datadog_monitor` | ✅ |
| Log alerts | `/logs_settings/scheduled_search` | `last9_scheduled_search_alert` | — | `datadog_monitor` (log) | ✅ |
| Entity snooze | `/entities/{id}/snooze`, `/alert-rules/snooze` | `last9_alert_snooze` | — | `datadog_downtime` | ✅ |
| Notification channels | `/notification_settings` | `last9_notification_channel` | notification_channel | integrations / monitor notify | ✅ |
| Dashboards | `/dashboards` | `last9_dashboard` | — | `datadog_dashboard` | ✅ |
| Synthetics | `/synthetic/checks` | `last9_synthetic_check` | — | `datadog_synthetics_test` | ✅ |
| Changeboards | `/changeboards` | `last9_changeboard` | — | (no direct) | ✅ |
| Relationships | `/entities/{id}/relationships` | — | relationship | service definitions | ❌ skipped |
| Drop rules | `/otel_settings/drop` | `last9_drop_rule` | — | `datadog_logs_*` pipelines | ✅ (otel) |
| Forward rules | `/otel_settings/forward` | `last9_forward_rule` | — | logs custom destination | ✅ (otel) |
| Remapping | `/otel_settings/remapping/*` | `last9_remapping_rule` | — | metric/tag pipelines | ✅ |
| Sensitive data | `/otel_settings/sensitive_data` | `last9_sensitive_data_rule` | — | logs scrubbing | ✅ |
| Rehydration | `/otel_settings/rehydration` | `last9_rehydration` | — | logs archive rehydrate | ✅ |
| Physical index | `/otel_settings/physical_index` | `last9_physical_index` | — | `datadog_logs_index` | ✅ |
| Cluster lookup | `/clusters` | `data.last9_cluster` | — | — | ✅ |
| Datasource lookup | `/datasources` | `data.last9_datasource` | — | — | ✅ |
| Users | `/users`, `/users/invite`, `/users/{id}/roles` | `last9_user` / `data.last9_user` | — | `datadog_user` | ✅ |
| SLOs | `/entities/{id}/slo` | — | slo | `datadog_service_level_objective` | ❌ skipped |
| Macros | `/clusters/{id}/macros` | — | — | — | ❌ skipped |
| Streaming aggregations | `/clusters/{id}/streaming_aggregations` | — | — | metrics pipelines | ❌ deferred |
| Cold storage / S3 ingest | `/otel_settings/cold_storage`, `/s3_ingest` | — | — | logs archives | ❌ deferred |

## Datadog parity (observability core)

| Datadog | Last9 TF | Gap |
|---------|----------|-----|
| Monitors | alerts + scheduled search | covered |
| Downtime | alert snooze | covered |
| SLOs | — | intentionally skipped |
| Synthetics | synthetic check | covered |
| Dashboards | dashboard | covered |
| Logs pipelines / indexes | drop/forward/remap/sensitive/physical_index | covered |
| Service catalog edges | — | intentionally skipped (relationships) |
| Users / roles | `last9_user` | covered |
| Security / RUM / cost / on-call | N/A | product gap, not TF gap |

## Legend

- ✅ Implemented in this coverage sweep (or already present)
- ❌ Explicitly out of scope
- Deferred = API exists; revisit if customer demand
