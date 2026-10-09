---
page_title: "last9_synthetic_check Resource - Last9"
subcategory: ""
description: |-
  Manages a Last9 synthetic monitoring check.
---

# last9_synthetic_check (Resource)

Manages a synthetic check (HTTP, TCP, DNS, ICMP, API, or script) evaluated on a schedule from one or more locations.

## Example Usage

```terraform
resource "last9_synthetic_check" "homepage" {
  name        = "homepage-health"
  description = "Probe the marketing homepage"
  type        = "http"
  schedule    = "*/5 * * * *"
  timeout     = 30
  frequency   = 60
  locations   = ["us-east-1"]

  config = jsonencode({
    url    = "https://example.com"
    method = "GET"
  })

  tags = {
    env = "prod"
  }
}
```

## Argument Reference

* `name` - (Required) Check name.
* `type` - (Required, ForceNew) One of `http`, `https`, `api`, `script`, `tcp`, `dns`, `icmp`.
* `schedule` - (Required) Cron expression or `every Nm` interval.
* `config` - (Required) JSON string with type-specific settings (`url`, `host`, `script`, etc.).
* `timeout` - (Required) Timeout in seconds (1–300).
* `frequency` - (Required) Frequency in seconds (minimum 60).
* `locations` - (Required) List of runner locations.
* `description` - (Optional) Description.
* `tags` - (Optional) Map of tags.
* `status` - (Optional) `active` or `paused`.

## Import

```shell
terraform import last9_synthetic_check.homepage <check_id>
```
