---
page_title: "last9_rehydration Resource - Last9"
subcategory: ""
description: |-
  Creates an OTel log rehydration job.
---

# last9_rehydration (Resource)

Creates a rehydration job to restore archived logs into a physical index for a time range.
ForceNew on all arguments — treat as create-once jobs.

## Example Usage

```terraform
resource "last9_rehydration" "incident" {
  region         = "ap-south-1"
  name           = "incident-2026-09-17"
  physical_index = "payments_logs"
  telemetry      = "logs"
  from           = 1726502400
  to             = 1726588800
  message        = "Rehydrate for incident review"
}
```

## Import

```shell
terraform import last9_rehydration.incident <region>:<id>
```
