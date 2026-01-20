# Walkthrough: Fix API Endpoint Paths in Terraform Provider

## Task Description

Fix critical issues with API endpoint paths in the Last9 Terraform provider to match the OpenAPI specification at `api/docs/openapi.yaml`.

## Issues Fixed

### Issue 1: OAuth Endpoint Path
- **Before**: `{baseURL}/organizations/{org}/oauth/access_token`
- **After**: `{baseURL}/v4/oauth/access_token`
- **Reason**: The OAuth endpoint per the OpenAPI spec does not require the organization prefix and must include the `/v4` version prefix.

### Issue 2: Missing `/v4` API Version Prefix
- **Before**: `{baseURL}/organizations/{org}{path}`
- **After**: `{baseURL}/v4/organizations/{org}{path}`
- **Reason**: All API endpoints require the `/v4` version prefix as per the OpenAPI specification.

### Issue 3: Missing `/api/` Prefix in URL Path
- **Before**: `{baseURL}/v4/organizations/{org}{path}`
- **After**: `{baseURL}/api/v4/organizations/{org}{path}`
- **Discovery**: Testing against alpha.last9.io showed that the `/api/` prefix is required.

### Issue 4: Entity Timestamps as Numbers
- **Before**: `CreatedAt` and `UpdatedAt` fields in Entity struct were `string`
- **After**: Changed to `int64`
- **Reason**: API returns Unix timestamps as numbers, not ISO strings

### Issue 5: Entity Update Method
- **Before**: `UpdateEntity` used PATCH method
- **After**: Changed to PUT method
- **Reason**: PATCH returns 405 Method Not Allowed

### Issue 6: KPIs Required for Alerts
- **Discovery**: Alerts require KPIs to be created separately via `/entities/{entity_id}/kpis`
- **Alert `expression_args`** must include KPI ID: `{"kpi_name": {"id": "kpi-uuid"}}`

### Issue 7: Severity Validation
- **Allowed values**: Only `"threat"` or `"breach"` (not "info")

### Issue 8: Separate Delete Token Required
- **Discovery**: Delete operations require a token with `delete` scope
- **API scopes**: `read | write | delete` are separate
- **Solution**: Added `delete_token` provider configuration

### Issue 9: Alert Update Recreates with New ID
- **Discovery**: PUT on alert-rules deletes and recreates the alert (per OpenAPI spec)
- **Solution**: Update resource ID from response after update operation

### Issue 10: Alert PUT Requires All Fields
- **Discovery**: Partial updates not supported for alert-rules
- **Solution**: Always send all required fields in update request

## Files Modified

### `internal/client/client.go`

| Function/Type | Change |
|---------------|--------|
| `Config` struct | Added `DeleteToken` field |
| `Client` struct | Added `deleteToken` field |
| `NewClient()` | Set deleteToken from config |
| `Delete()` | Use separate delete token for DELETE requests |
| `refreshAccessToken()` | Changed URL to `%s/v4/oauth/access_token` |
| `doRequest()` | Changed URL to `%s/api/v4/organizations/%s%s` |
| `Entity` struct | Changed timestamps from `string` to `int64` |
| `UpdateEntity()` | Changed from `Patch` to `Put` |
| Added `KPI`, `KPIDefinition`, `KPICreateRequest`, `KPIUpdateRequest` types |
| Added `CreateKPI()`, `GetKPI()`, `UpdateKPI()`, `DeleteKPI()` methods |

### `internal/provider/provider.go`

| Change |
|--------|
| Added `delete_token` schema field with `LAST9_DELETE_TOKEN` env var |
| Read delete_token in `configureProvider()` and pass to client config |

### `internal/provider/resource_alert.go`

| Function | Change |
|----------|--------|
| Added `generateKPIName()` helper | Generates KPI name as `{alert_name}-{random_token}` |
| Schema | Added `query` (Required), `kpi_id` (Computed), `kpi_name` (Computed) fields |
| Schema | Changed `indicator` from Required to Computed |
| Schema | Fixed severity validation: only `threat` or `breach` |
| `resourceAlertCreate()` | Create KPI first, then alert with KPI ID in expression_args |
| `resourceAlertUpdate()` | Create new KPI on name/query change, update alert, delete old KPI; update resource ID from response |
| `resourceAlertUpdate()` | Always send all required fields (PUT doesn't support partial updates) |
| `resourceAlertDelete()` | Delete alert, then delete associated KPI |

### `internal/provider/resource_entity.go`

| Function | Change |
|----------|--------|
| Schema | Made `data_source_id` Computed to avoid update drift |
| `resourceEntityUpdate()` | Only send non-empty `data_source_id` |

## Trade-offs Considered

1. **Version prefix location**: Chose path builder over base URL default to avoid breaking existing user configurations.

2. **Separate delete token**: The Last9 API requires separate tokens for different scopes. While not ideal for UX, this matches the API's security model.

3. **KPI lifecycle management**: KPIs are automatically created/deleted with alerts rather than requiring users to manage them separately. This simplifies the Terraform configuration at the cost of some flexibility.

4. **Alert ID changes on update**: The API deletes and recreates alerts on update. The provider handles this by updating the resource ID, which may cause issues with external references to the alert ID.

## Live API Testing Results (2026-01-20)

| Operation | Status | Notes |
|-----------|--------|-------|
| Entity Create | ✅ Success | |
| Entity Read | ✅ Success | |
| KPI Create | ✅ Success | Auto-created with alert |
| Alert Create | ✅ Success | With KPI reference in expression_args |
| Alert Update (name change) | ✅ Success | New KPI created, old KPI deleted, alert ID updated |
| Alert Delete | ✅ Success | KPI also deleted |
| Entity Delete | ✅ Success | |

## Outstanding Issues

1. **Entity Update**: PUT method requires all fields but provider only sends changed fields. Tags and labels in entity update may not work correctly. Entity metadata fields (tags, labels) may need to be nested in a `metadata` object per API structure.

2. **Expression field drift**: The `expression` field shows as removed in plans but is computed from KPI name.

3. **is_disabled/mute drift**: These fields may show changes in plans even when not modified in config.

## Test Configuration

Test configuration is in `llmtemp/api-test/`:
- `main.tf` - Entity and alert resources
- `variables.tf` - access_token and delete_token variables
- `terraform.tfvars` - Token values (gitignored)
- `dev.tfrc` - Provider dev overrides
