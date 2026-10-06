---
page_title: "last9_cluster Data Source - Last9"
subcategory: ""
description: |-
  Looks up a Last9 cluster by region, id, or name.
---

# last9_cluster (Data Source)

Resolves a cluster for a region. If neither `id` nor `name` is set, returns the default cluster.

```terraform
data "last9_cluster" "default" {
  region = "ap-south-1"
}
```

## Schema

### Required

- `region` (String) Region to look up clusters in.

### Optional

- `id` (String) Cluster ID to look up. Takes precedence over `name`. If neither `id` nor `name` is set, the default cluster for the region is returned.
- `name` (String) Cluster name to look up.

### Read-Only

- `default` (Boolean) Whether this is the default cluster for the region.
