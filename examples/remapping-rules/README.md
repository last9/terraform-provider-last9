# Remapping Rule Examples

End-to-end examples for `last9_remapping_rule`. Four rules covering all three types.

## What's Inside

**`logs_extract_pattern`** — regex pattern with named capture groups. The regex itself goes in `remap_keys[0]`; named groups (e.g. `(?P<level>...)`) become extracted attributes.

**`logs_extract_json`** — JSON body parser, gated on a `preconditions` block (only fires when `severity = "error"`). `remap_keys[0]` is the source field path.

**`logs_map_service`** — promote one of multiple incoming attributes (`svc`, `app_name`) to the standard `service` field.

**`traces_map_service`** — same shape, but for spans. Only valid `target_attributes` is `service`.

## target_attributes per type

| type | valid `target_attributes` |
|------|---------------------------|
| `logs_extract` | `log_attributes`, `resource_attributes` |
| `logs_map` | `service`, `severity`, `resource_deployment.environment` |
| `traces_map` | `service` |

## Running Locally Against Your Built Provider

Build the provider:

```bash
go build -o /tmp/terraform-provider-last9 .
```

Add to `~/.terraformrc` (or pass via `TF_CLI_CONFIG_FILE`):

```hcl
provider_installation {
  dev_overrides {
    "last9/last9" = "/tmp"
  }
  direct {}
}
```

## Running

```bash
cd examples/remapping-rules

export LAST9_REFRESH_TOKEN=...
export LAST9_DELETE_REFRESH_TOKEN=...
export LAST9_ORG=your-org

terraform plan \
  -var "last9_refresh_token=$LAST9_REFRESH_TOKEN" \
  -var "last9_delete_refresh_token=$LAST9_DELETE_REFRESH_TOKEN" \
  -var "last9_org=$LAST9_ORG"

terraform apply -auto-approve \
  -var "last9_refresh_token=$LAST9_REFRESH_TOKEN" \
  -var "last9_delete_refresh_token=$LAST9_DELETE_REFRESH_TOKEN" \
  -var "last9_org=$LAST9_ORG"

terraform destroy -auto-approve \
  -var "last9_refresh_token=$LAST9_REFRESH_TOKEN" \
  -var "last9_delete_refresh_token=$LAST9_DELETE_REFRESH_TOKEN" \
  -var "last9_org=$LAST9_ORG"
```

## Importing an Existing Rule

```bash
terraform import last9_remapping_rule.logs_extract_pattern ap-south-1:logs_extract:<rule-id>
```

Composite import ID is `region:type:id` — region and type are required because they pick the right API endpoint.

## Schema Notes

- For `pattern` extracts, `remap_keys[0]` is a Go-syntax regex with at least one named capture group. Multiple keys are not allowed (server enforces).
- For `json` extracts, `remap_keys[0]` is the source field/path on the log record.
- `preconditions` only applies to `logs_extract`. Maximum 1 block.
- `extract_type`, `action`, `prefix`, `preconditions` are forbidden on `logs_map` and `traces_map`.
