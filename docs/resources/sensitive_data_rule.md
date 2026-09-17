---
page_title: "last9_sensitive_data_rule Resource - Last9"
subcategory: ""
description: |-
  Manages an OTel sensitive-data scanning rule.
---

# last9_sensitive_data_rule (Resource)

Scans logs for email, phone, or credit-card patterns and optionally redacts matches.

## Example Usage

```terraform
resource "last9_sensitive_data_rule" "pii" {
  region          = "ap-south-1"
  name            = "redact-pii"
  telemetry       = "logs"
  order           = 1
  scan_email      = true
  scan_phone_number = true
  action_name     = "redact"
}
```

## Import

```shell
terraform import last9_sensitive_data_rule.pii <region>:<id>
```
