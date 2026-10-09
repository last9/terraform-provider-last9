---
page_title: "last9_cluster Data Source - Last9"
subcategory: ""
description: |-
  Looks up a Last9 cluster by region, id, or name.
---

# last9_cluster (Data Source)

Resolves a cluster in the configured region. Set at most one of `id` and `name`; if neither is set, returns the default cluster for that region.

```terraform
data "last9_cluster" "default" {
  region = "ap-south-1"
}
```

## Schema

### Required

- `region` (String) Region used to look up the cluster and its default cluster.

### Optional

- `id` (String) Cluster ID to look up. Cannot be set with `name`.
- `name` (String) Cluster name to look up. Cannot be set with `id`.

### Read-Only

- `default` (Boolean) Whether this is the default cluster for the region.
