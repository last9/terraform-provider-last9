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

## Schema

### Required

- `action_name` (String) Action when a match is found. Valid values: `none`, `redact`. Sent as `properties.action.name`.
- `name` (String) Name of the rule.
- `order` (Number) Rule evaluation order. Minimum value: 1.
- `region` (String) Last9 region. Sent as the `region` query parameter and used in the resource ID. Changing this forces a new resource.
- `telemetry` (String) Telemetry type. Valid values: `logs`.

### Optional

- `labels` (Map of String) Maps to the API `properties.labels` field.
- `scan_credit_card` (Boolean) Scan for credit card numbers. Sent as `properties.scan_rules.credit_card_number`. Default: `false`.
- `scan_email` (Boolean) Scan for email addresses. Sent as `properties.scan_rules.email`. Default: `false`.
- `scan_phone_number` (Boolean) Scan for phone numbers. Sent as `properties.scan_rules.phone_number`. Default: `false`.

### Read-Only

- `id` (String) The ID of this resource, in the format `region:id`.
- `status` (String) Status returned by the API.

## Import

```shell
terraform import last9_sensitive_data_rule.pii <region>:<id>
```
