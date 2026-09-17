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

## Import

```shell
terraform import last9_streaming_aggregation.http_by_service <region>:<cluster_id>:<id>
```
