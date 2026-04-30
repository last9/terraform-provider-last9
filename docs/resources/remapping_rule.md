---
page_title: "last9_remapping_rule Resource - Last9"
subcategory: ""
description: |-
  Manages a Last9 remapping rule for extracting fields from logs and mapping attributes for logs and traces.
---

# last9_remapping_rule (Resource)

Manages a Last9 remapping rule. Remapping rules transform telemetry at the ingestion layer:

- **`logs_extract`** — extract fields from log lines using a regex pattern (with named capture groups) or a JSON parser.
- **`logs_map`** — promote one or more attributes to standard log fields (`service`, `severity`, or `resource_deployment.environment`).
- **`traces_map`** — promote one or more attributes to the standard `service` field on spans.

Rules are applied without redeployment. Multiple rules of the same type cascade in creation order.

## Example Usage

### Extract Fields with a Regex Pattern

```terraform
resource "last9_remapping_rule" "nginx_access" {
  region            = "ap-south-1"
  type              = "logs_extract"
  name              = "nginx-access-log-parser"
  extract_type      = "pattern"
  action            = "upsert"
  remap_keys        = ["(?P<method>\\w+) (?P<path>/[^\\s]*) HTTP/(?P<version>[\\d.]+)"]
  target_attributes = "log_attributes"
}
```

The `remap_keys[0]` value is the regex itself. Named capture groups (e.g., `(?P<method>...)`) become extracted attributes.

### Extract Fields with a JSON Parser (with Precondition)

```terraform
resource "last9_remapping_rule" "json_body" {
  region            = "ap-south-1"
  type              = "logs_extract"
  name              = "extract-json-body"
  extract_type      = "json"
  action            = "upsert"
  remap_keys        = ["attributes[\"json_body\"]"]
  target_attributes = "log_attributes"

  preconditions {
    key      = "severity"
    value    = "error"
    operator = "equals"
  }
}
```

The `remap_keys[0]` value is the source field path. The `preconditions` block gates extraction so it fires only on matching events.

### Map an Attribute to the Standard `service` Field

```terraform
resource "last9_remapping_rule" "service_name" {
  region            = "ap-south-1"
  type              = "logs_map"
  name              = "map-svc-to-service"
  remap_keys        = ["svc", "app_name"]
  target_attributes = "service"
}
```

Multiple keys are tried in order; the first non-empty value wins.

### Map an Attribute on Traces

```terraform
resource "last9_remapping_rule" "trace_service" {
  region            = "ap-south-1"
  type              = "traces_map"
  name              = "map-service-name"
  remap_keys        = ["service.name", "k8s.deployment.name"]
  target_attributes = "service"
}
```

## Schema

### Required

- `region` (String, ForceNew) Region for the remapping rule.
- `type` (String, ForceNew) One of `logs_extract`, `logs_map`, `traces_map`.
- `name` (String) Rule name.
- `remap_keys` (List of String, MinItems: 1) Source values:
  - For `logs_extract` with `extract_type = "pattern"`: a single regex with named capture groups.
  - For `logs_extract` with `extract_type = "json"`: a single source field path.
  - For `logs_map` / `traces_map`: ordered list of attribute keys to try.
- `target_attributes` (String) Target attribute. Valid values per type:
  - `logs_extract`: `log_attributes`, `resource_attributes`
  - `logs_map`: `service`, `severity`, `resource_deployment.environment`
  - `traces_map`: `service`

### Optional (logs_extract only)

- `extract_type` (String) `pattern` (regex) or `json`. Required for `logs_extract`.
- `action` (String) `insert` or `upsert`. Defaults to `upsert`.
- `prefix` (String) Optional prefix prepended to extracted attribute keys.
- `preconditions` (Block List, Max: 1) Conditional gate. See [Preconditions](#preconditions).

### Read-Only

- `id` (String) Server-assigned rule UUID.
- `created_at` (Number) Creation timestamp (Unix seconds).
- `updated_at` (Number) Last-update timestamp (Unix seconds).
- `created_by` (String) UUID of the user who created the rule.
- `status` (String) Rule status (e.g., `pending`, `active`).

### Preconditions

The `preconditions` block supports:

- `key` (String, Required) Attribute key to test.
- `value` (String, Required) Expected value.
- `operator` (String, Required) `equals`, `not_equals`, or `like` (regex).

## Type-Specific Constraints

`logs_map` and `traces_map` reject `extract_type`, `action`, `prefix`, and `preconditions`. `logs_extract` requires `extract_type`. These are enforced at plan time.

## Import

Remapping rules can be imported using the format `region:type:id`:

```shell
terraform import last9_remapping_rule.example ap-south-1:logs_extract:7fe32a34-5d1d-495c-87b5-f4020e34c7db
```

The composite ID is required because each rule type uses a distinct API endpoint.
