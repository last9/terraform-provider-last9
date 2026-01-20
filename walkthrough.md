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

### Issue 11: Entity Update Uses Separate Endpoints
- **Discovery**: Entity PUT only updates core fields (name, type, description, etc.)
- **Metadata fields** (tags, labels, team, links, adhoc_filter) require separate `PUT /entities/{id}/metadata` endpoint
- **Solution**: Split `resourceEntityUpdate` into two API calls based on which fields changed

### Issue 12: Entity Create Doesn't Accept Metadata
- **Discovery**: POST /entities API doesn't accept tags, labels, team, links, adhoc_filter
- **Solution**: Create entity first, then call `PUT /entities/{id}/metadata` to set metadata fields

### Issue 13: Alert mute Field Causes Drift
- **Discovery**: API may return different values for `mute` than what was sent
- **Solution**: Removed `mute` field from Terraform schema entirely (not managed by Terraform)

### Issue 14: Alert expression Field Drift
- **Discovery**: The `expression` field is computed by the API based on KPI name
- **Solution**: Changed `expression` field to Computed only (not user-settable)

### Issue 15: Alert is_disabled Field Drift
- **Discovery**: API returns different values for `is_disabled` than what was sent
- **Solution**: Don't read `is_disabled` from API - preserve user's config value

### Issue 16: Entity Metadata Returned in Nested Object
- **Discovery**: GET /entities/{id} returns tags, labels, team, links in nested `metadata` object
- **Solution**: Added `EntityMetadata` struct and read from `entity.Metadata` in resourceEntityRead

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
| Added `EntityMetadataUpdateRequest` type |
| Added `UpdateEntityMetadata()` method for separate metadata endpoint |
| Added `EntityMetadata` struct for nested metadata in API response |

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
| Schema | Removed `mute` field (causes drift, not managed by Terraform) |
| Schema | Changed `expression` to Computed only (computed by API from KPI name) |
| `resourceAlertCreate()` | Create KPI first, then alert with KPI ID in expression_args |
| `resourceAlertUpdate()` | Create new KPI on name/query change, update alert, delete old KPI; update resource ID from response |
| `resourceAlertUpdate()` | Always send all required fields (PUT doesn't support partial updates) |
| `resourceAlertDelete()` | Delete alert, then delete associated KPI |

### `internal/provider/resource_entity.go`

| Function | Change |
|----------|--------|
| Schema | Made `data_source_id` Computed to avoid update drift |
| `resourceEntityCreate()` | Call `UpdateEntityMetadata()` after create to set tags, labels, team, links, adhoc_filter |
| `resourceEntityRead()` | Read metadata from nested `entity.Metadata` object in API response |
| `resourceEntityRead()` | Don't set team="default" in state (use only when user specifies team) |
| `resourceEntityUpdate()` | Split into two API calls: core fields via `UpdateEntity()`, metadata via `UpdateEntityMetadata()` |
| `resourceEntityUpdate()` | Core fields: name, type, description, data_source_id, namespace, tier, workspace, ui_readonly |
| `resourceEntityUpdate()` | Metadata fields: tags, labels, team, links, adhoc_filter |

## Trade-offs Considered

1. **Version prefix location**: Chose path builder over base URL default to avoid breaking existing user configurations.

2. **Separate delete token**: The Last9 API requires separate tokens for different scopes. While not ideal for UX, this matches the API's security model.

3. **KPI lifecycle management**: KPIs are automatically created/deleted with alerts rather than requiring users to manage them separately. This simplifies the Terraform configuration at the cost of some flexibility.

4. **Alert ID changes on update**: The API deletes and recreates alerts on update. The provider handles this by updating the resource ID, which may cause issues with external references to the alert ID.

5. **mute field removed**: The `mute` field was removed from the Terraform schema because the API returns inconsistent values. Users who need mute functionality must manage it outside Terraform.

6. **is_disabled not read from API**: The `is_disabled` value is preserved from config rather than read from API, because the API may return different values. This means external changes to is_disabled won't be detected by Terraform.

## Live API Testing Results (2026-01-20)

| Operation | Status | Notes |
|-----------|--------|-------|
| Entity Create | ✅ Success | With tags and labels |
| Entity Read | ✅ Success | |
| Entity Update (metadata) | ✅ Success | Tags updated via separate metadata endpoint |
| KPI Create | ✅ Success | Auto-created with alert |
| Alert Create | ✅ Success | With KPI reference in expression_args |
| Alert Update (name change) | ✅ Success | New KPI created, old KPI deleted, alert ID updated |
| Alert Delete | ✅ Success | KPI also deleted |
| Entity Delete | ✅ Success | |
| No Drift Check | ✅ Success | `terraform plan` shows no changes after apply |

## Resolved Drift Issues

All drift issues have been fixed:

1. **Expression field drift** - Fixed by making `expression` Computed only
2. **mute drift** - Fixed by removing `mute` field from schema
3. **is_disabled drift** - Fixed by not reading `is_disabled` from API response
4. **Entity tags/labels drift** - Fixed by reading from nested `metadata` object and setting metadata via separate API call during create

## Test Configuration

Test configuration is in `llmtemp/api-test/`:
- `main.tf` - Entity and alert resources
- `variables.tf` - access_token and delete_token variables
- `terraform.tfvars` - Token values (gitignored)
- `dev.tfrc` - Provider dev overrides
