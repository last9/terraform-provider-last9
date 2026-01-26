---
page_title: "last9_remapping_rule Resource - Last9"
subcategory: ""
description: |-
  Manages a Last9 remapping rule for transforming telemetry attributes.
---

# last9_remapping_rule (Resource)

Manages a Last9 remapping rule. Remapping rules transform telemetry data by extracting fields or mapping attributes to standard formats.

## Remapping Rule Types

| Type | Description | Use Case |
|------|-------------|----------|
| `logs_extract` | Extract fields from log messages using regex or JSON parsing | Parse structured data from log bodies |
| `logs_map` | Map log attributes to standard fields | Normalize service names, severity levels |
| `traces_map` | Map trace attributes to standard fields | Normalize service names in traces |

## Example Usage

### Logs Extract - Pattern (Regex)

Extract fields from log messages using regex patterns:

```terraform
resource "last9_remapping_rule" "extract_request_id" {
  region            = "us-west-2"
  type              = "logs_extract"
  name              = "extract-request-id"
  remap_keys        = ["body"]
  target_attributes = "log_attributes"
  action            = "upsert"
  extract_type      = "pattern"

  preconditions {
    key      = "attributes[\"service.name\"]"
    value    = "api-gateway"
    operator = "equals"
  }
}
```

### Logs Extract - JSON

Extract fields from JSON-formatted log bodies:

```terraform
resource "last9_remapping_rule" "extract_json_fields" {
  region            = "us-west-2"
  type              = "logs_extract"
  name              = "extract-json-metadata"
  remap_keys        = ["body"]
  target_attributes = "log_attributes"
  action            = "insert"
  extract_type      = "json"
  prefix            = "parsed_"
}
```

### Logs Map - Service Name

Map a custom attribute to the standard service name:

```terraform
resource "last9_remapping_rule" "map_service_name" {
  region            = "us-west-2"
  type              = "logs_map"
  name              = "map-app-to-service"
  remap_keys        = ["attributes[\"app.name\"]"]
  target_attributes = "service"
  action            = "upsert"
}
```

### Logs Map - Severity

Map custom severity levels to standard severity:

```terraform
resource "last9_remapping_rule" "map_severity" {
  region            = "us-west-2"
  type              = "logs_map"
  name              = "map-log-level"
  remap_keys        = ["attributes[\"log.level\"]", "attributes[\"level\"]"]
  target_attributes = "severity"
  action            = "upsert"
}
```

### Traces Map - Service Name

Map trace attributes to standard service name:

```terraform
resource "last9_remapping_rule" "map_trace_service" {
  region            = "us-west-2"
  type              = "traces_map"
  name              = "map-trace-service"
  remap_keys        = ["resource.attributes[\"k8s.deployment.name\"]"]
  target_attributes = "service"
  action            = "upsert"
}
```

## Schema

### Required

- `region` (String) Region for the remapping rule.
- `type` (String) Remapping rule type: `logs_extract`, `logs_map`, or `traces_map`.
- `name` (String) Name of the remapping rule (unique within type and region).
- `remap_keys` (List of String) Source field(s) to remap from.
- `target_attributes` (String) Target attribute to map to:
  - For `logs_extract`: `log_attributes` or `resource_attributes`
  - For `logs_map`: `service`, `severity`, or `resource_deployment.environment`
  - For `traces_map`: `service`

### Optional

- `action` (String) Action to take: `insert` (only if not exists) or `upsert` (insert or update). Default: `upsert`.
- `extract_type` (String) Extraction type for `logs_extract`: `pattern` (regex) or `json`. Required for `logs_extract` type.
- `prefix` (String) Optional prefix for extracted values (`logs_extract` only).
- `preconditions` (Block List) Conditional rules for when to apply extraction (`logs_extract` only). See [Preconditions](#preconditions) below.

### Read-Only

- `id` (String) The ID of the remapping rule (format: `region:type:name`).
- `created_at` (Number) Creation timestamp.
- `updated_at` (Number) Last update timestamp.

### Preconditions

Preconditions define when an extraction rule should be applied. Only available for `logs_extract` type.

- `key` (String, Required) Attribute key to match.
- `value` (String, Required) Value to match against.
- `operator` (String, Required) Match operator: `equals`, `not_equals`, or `like` (regex).

## Target Attributes Reference

### logs_extract
| Target | Description |
|--------|-------------|
| `log_attributes` | Extract to log-level attributes |
| `resource_attributes` | Extract to resource-level attributes |

### logs_map
| Target | Description |
|--------|-------------|
| `service` | Map to standard service name |
| `severity` | Map to standard severity level |
| `resource_deployment.environment` | Map to deployment environment |

### traces_map
| Target | Description |
|--------|-------------|
| `service` | Map to standard service name |

## Import

Remapping rules can be imported using the format `region:type:name`:

```shell
terraform import last9_remapping_rule.example us-west-2:logs_extract:my-rule
```
