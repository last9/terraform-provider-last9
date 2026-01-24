---
page_title: "last9_forward_rule Resource - Last9"
subcategory: ""
description: |-
  Manages a Last9 log forward rule for routing logs to external destinations.
---

# last9_forward_rule (Resource)

Manages a Last9 forward rule. Forward rules route telemetry matching specified criteria to external destinations like object storage (AWS S3), SIEM systems, or data lakes. Forwarded data is automatically compressed and stored in `.gz` format.

For more information, see the [Forward Rules documentation](https://last9.io/docs/control-plane-forward/).

~> **Warning** Data matching forward rules is NOT stored in Last9 and cannot be recovered from Last9. Use the "View Logs" preview feature in the Last9 UI to verify filters before saving.

## Prerequisites for S3 Forwarding

Before creating forward rules to AWS S3, configure the S3 backend in Last9:

1. **Create an S3 bucket** for storing forwarded telemetry
2. **Configure IAM AssumeRole** permissions for Last9:
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Principal": {
           "AWS": "arn:aws:iam::LAST9_ACCOUNT_ID:root"
         },
         "Action": "sts:AssumeRole",
         "Condition": {
           "StringEquals": {
             "sts:ExternalId": "<your-external-id>"
           }
         }
       }
     ]
   }
   ```
3. **Add Cold Storage configuration** in Last9 Control Plane → Cold Storage with your IAM AssumeRole ARN
4. Create forward rules that route to the configured S3 backend

-> **Note** Contact Last9 support for the exact `LAST9_ACCOUNT_ID` and setup assistance.

## Example Usage

### Forward Critical Logs

```terraform
resource "last9_forward_rule" "forward_critical" {
  region      = "ap-south-1"
  name        = "forward-critical-logs"
  telemetry   = "logs"
  destination = "https://logs.external-system.com/webhook"

  filters {
    key         = "SeverityText"
    value       = "CRITICAL"
    operator    = "equals"
    conjunction = "and"
  }
}
```

### Forward Security Logs to SIEM

```terraform
resource "last9_forward_rule" "security_logs" {
  region      = "ap-south-1"
  name        = "forward-security-logs"
  telemetry   = "logs"
  destination = "https://siem.example.com/api/logs"

  filters {
    key         = "attributes[\"category\"]"
    value       = "security"
    operator    = "equals"
    conjunction = "and"
  }

  filters {
    key         = "SeverityText"
    value       = "WARNING"
    operator    = "not_equals"
    conjunction = "and"
  }
}
```

## Schema

### Required

- `region` (String) Region for the forward rule (e.g., "ap-south-1", "us-west-2").
- `name` (String) Name of the forward rule.
- `telemetry` (String) Telemetry type. Valid values: `logs`, `traces`, `metrics`.
- `destination` (String) Destination URL to forward logs to.
- `filters` (Block List, Min: 1) Filter conditions. See [Filters](#filters) below.

### Optional

- `cluster_id` (String) Cluster ID. If not provided, uses the default cluster for the region.

### Read-Only

- `id` (String) The ID of the forward rule.

### Filters

The `filters` block supports:

- `key` (String, Required) The field key to filter on (e.g., "SeverityText", "attributes[\"service\"]").
- `value` (String, Required) The value to match.
- `operator` (String, Required) Comparison operator. Valid values: `equals`, `not_equals`, `like` (regex match).
- `conjunction` (String, Required) Logical conjunction. Valid values: `and`, `or`.

## Import

Forward rules can be imported using the format `region:cluster_id:name`:

```shell
terraform import last9_forward_rule.example ap-south-1:cluster-123:forward-critical-logs
```
