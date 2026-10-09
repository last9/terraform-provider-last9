---
page_title: "last9_streaming_aggregation Resource - Last9"
subcategory: ""
description: |-
  Manages a Levitate streaming aggregation rule.
---

# last9_streaming_aggregation (Resource)

Creates a streaming aggregation pipeline on a cluster (`/clusters/{id}/streaming_aggregations`).
Supports `metrics` and `events` telemetry. Use scheduled search for logs/traces.

## Example Usage

```terraform
resource "last9_streaming_aggregation" "http_by_service" {
  region        = "ap-south-1"
  name          = "http-requests-by-service"
  telemetry     = "metrics"
  metric        = "http_requests_total"
  resolution    = "1m"
  aggregation   = "sum"
  clause        = "without"
  labels        = ["instance"]
  output_metric = "http_requests_by_service"
}
```

## Schema

### Required

- `aggregation` (String) Aggregation function. Valid values: `max`, `sum`, `sum2`, `sum_counter`.
- `clause` (String) Label clause. Valid values: `with` (by), `without`.
- `labels` (List of String) Labels that `clause` applies to. At least one label is required.
- `metric` (String) Source metric name.
- `name` (String) Name of the streaming aggregation.
- `output_metric` (String) Prometheus-style output metric name.
- `region` (String) Last9 region. Sent as the `region` request header and used in the resource ID. Changing this forces a new resource.
- `resolution` (String) Resolution, such as `1m`, `5m`, or `1h`.
- `telemetry` (String) Telemetry type. Valid values: `metrics`, `events`. Use scheduled search for logs and traces.

### Optional

- `cluster_id` (String) Cluster ID, used in the `/clusters/{id}/streaming_aggregations` path. If not set, the provider uses the default cluster for the region. Changing this forces a new resource.
- `with_name` (String) Maps to the API `properties.with_name` field.
- `with_value` (String) Maps to the API `properties.with_value` field.

### Read-Only

- `id` (String) The ID of this resource, in the format `region:cluster_id:id`.

## Import

```shell
terraform import last9_streaming_aggregation.http_by_service <region>:<cluster_id>:<id>
```
