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
