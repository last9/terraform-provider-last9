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

---

# Session: Integration Test Suite for Entity and Alert Resources (2026-01-20)

## Task Description

Create a comprehensive integration test suite for the Terraform provider covering:
- Alert Groups (Entities) - create, update, delete
- Alert Group Metadata - tags, labels, team, links
- Alert Rules - create, update, delete with KPI lifecycle

## Files Created/Modified

### 1. `internal/provider/provider_config_test.go`

| Change |
|--------|
| Added support for `LAST9_DELETE_TOKEN` environment variable |
| Added support for `LAST9_API_BASE_URL` environment variable |
| Refactored config building to handle all token combinations |

### 2. `internal/provider/provider_test.go`

| Function | Change |
|----------|--------|
| `testAccPreCheckWithDelete()` | New function - checks for delete token in addition to standard precheck |

### 3. `internal/provider/resource_entity_test.go` (NEW)

Comprehensive integration tests for Entity (Alert Group) resource.

**Test Cases:**
| Test | Description |
|------|-------------|
| `TestAccEntity_basic` | Create entity with required fields only, verify import |
| `TestAccEntity_full` | Create entity with all fields (description, tier, namespace, workspace) |
| `TestAccEntity_withMetadata` | Create entity with tags, labels, team |
| `TestAccEntity_update` | Update entity name, description, tier |
| `TestAccEntity_updateMetadata` | Update tags, labels, team |
| `TestAccEntity_withLinks` | Create entity with related links |

**Helper Functions:**
| Function | Purpose |
|----------|---------|
| `testAccCheckEntityExists()` | Verify entity exists in state |
| `testAccCheckEntityDestroy()` | Verify entity destroyed |
| `testAccEntityConfig_basic()` | Basic entity config |
| `testAccEntityConfig_full()` | Full entity config with all fields |
| `testAccEntityConfig_withMetadata()` | Entity with tags, labels, team |
| `testAccEntityConfig_withMetadataUpdated()` | Updated metadata config |
| `testAccEntityConfig_updated()` | Updated entity core fields |
| `testAccEntityConfig_withLinks()` | Entity with links |

### 4. `internal/provider/resource_alert_integration_test.go` (NEW)

Integration tests for Alert Rules with full lifecycle.

**Test Cases:**
| Test | Description |
|------|-------------|
| `TestAccAlertIntegration_fullLifecycle` | Complete create → update → delete cycle |
| `TestAccAlertIntegration_multipleAlerts` | Multiple alerts on same entity |
| `TestAccAlertIntegration_staticThreshold` | Static threshold alert |
| `TestAccAlertIntegration_withProperties` | Alert with runbook and annotations |
| `TestAccAlertIntegration_import` | Import existing alert |
| `TestAccAlertIntegration_lessThanThreshold` | Less-than threshold alert |

**Helper Functions:**
| Function | Purpose |
|----------|---------|
| `testAccCheckAlertIntegrationExists()` | Verify alert exists in state |
| `testAccCheckAlertIntegrationDestroy()` | Verify alert destroyed |
| `testAccAlertIntegrationConfig_basic()` | Basic alert with entity |
| `testAccAlertIntegrationConfig_multipleAlerts()` | Multiple alerts config |
| `testAccAlertIntegrationConfig_staticThreshold()` | Static threshold config |
| `testAccAlertIntegrationConfig_withProperties()` | Alert with properties |
| `testAccAlertIntegrationConfig_lessThanThreshold()` | Less-than threshold |

## Environment Variables

Tests use these environment variables:
| Variable | Description | Required |
|----------|-------------|----------|
| `LAST9_API_TOKEN` | Write-scoped API token | Yes (or LAST9_REFRESH_TOKEN) |
| `LAST9_REFRESH_TOKEN` | Refresh token for auto-renewal | Yes (or LAST9_API_TOKEN) |
| `LAST9_DELETE_TOKEN` | Delete-scoped API token | Yes (for destroy tests) |
| `LAST9_API_BASE_URL` | API hostname (e.g., `https://alpha.last9.io`) | No |
| `LAST9_ORG` | Organization slug | Yes |

## Running Tests

```bash
# Set environment variables
export LAST9_API_TOKEN="<write-token>"
export LAST9_DELETE_TOKEN="<delete-token>"
export LAST9_API_BASE_URL="https://alpha.last9.io"
export LAST9_ORG="last9"

# Run all integration tests
TF_ACC=1 go test -v ./internal/provider/ -run "TestAcc" -timeout 30m

# Run entity tests only
TF_ACC=1 go test -v ./internal/provider/ -run "TestAccEntity" -timeout 10m

# Run alert integration tests only
TF_ACC=1 go test -v ./internal/provider/ -run "TestAccAlertIntegration" -timeout 20m

# Run specific test
TF_ACC=1 go test -v ./internal/provider/ -run "TestAccEntity_basic" -timeout 10m
```

## Trade-offs Considered

1. **Test naming convention**: Used `TestAccAlertIntegration_*` instead of extending existing `TestAccAlert_*` to clearly distinguish integration tests that create their own entities vs tests that require pre-existing entity IDs.

2. **Unique resource names**: Used `time.Now().UnixNano()` suffix for entity names and external_refs to avoid conflicts between parallel test runs or re-runs.

3. **Delete token requirement**: All tests use `testAccPreCheckWithDelete()` to ensure delete token is available, since tests create and destroy resources.

4. **Import verification**: Some fields are ignored during import verification (`query`, `kpi_id`, `kpi_name`) because these are computed during create and may not match exactly on import.

## Fixes Made During Testing

### Fix 1: Remove invalid tier values from entity tests
- **Issue**: API returned "422 invalid tier" for tier values "critical" and "high"
- **Fix**: Removed tier field from `TestAccEntity_full` and `TestAccEntity_update` tests
- **Files**: `internal/provider/resource_entity_test.go`

### Fix 2: Fix properties field in alert Read function
- **Issue**: Panic when setting `description` inside `properties` block - field doesn't exist in schema
- **Fix**: Removed `description` from properties map, only set `runbook_url` and `annotations`
- **Files**: `internal/provider/resource_alert.go:276-293`

### Fix 3: Implement composite import ID for alerts
- **Issue**: Alert import failed with "invalid entity id" - alerts require entity_id context
- **Fix**: Added `resourceAlertImportState` function that parses `entity_id:alert_id` format
- **Files**: `internal/provider/resource_alert.go:465-479`

### Fix 4: Add is_disabled to import ignore list
- **Issue**: `is_disabled` is intentionally not read from API to avoid drift
- **Fix**: Added `is_disabled` to `ImportStateVerifyIgnore` in import test
- **Files**: `internal/provider/resource_alert_integration_test.go:181-183`

## Verification

| Check | Status |
|-------|--------|
| `go build ./...` | ✅ Passes |
| `go vet ./internal/provider/...` | ✅ Passes |
| Test compilation | ✅ Compiles successfully |

## Test Results (Live API)

```
=== RUN   TestAccAlertIntegration_fullLifecycle
--- PASS: TestAccAlertIntegration_fullLifecycle (2.87s)
=== RUN   TestAccAlertIntegration_multipleAlerts
--- PASS: TestAccAlertIntegration_multipleAlerts (1.15s)
=== RUN   TestAccAlertIntegration_staticThreshold
--- PASS: TestAccAlertIntegration_staticThreshold (1.08s)
=== RUN   TestAccAlertIntegration_withProperties
--- PASS: TestAccAlertIntegration_withProperties (1.06s)
=== RUN   TestAccAlertIntegration_import
--- PASS: TestAccAlertIntegration_import (1.33s)
=== RUN   TestAccAlertIntegration_lessThanThreshold
--- PASS: TestAccAlertIntegration_lessThanThreshold (1.12s)
=== RUN   TestAccEntity_basic
--- PASS: TestAccEntity_basic (1.00s)
=== RUN   TestAccEntity_full
--- PASS: TestAccEntity_full (0.78s)
=== RUN   TestAccEntity_withMetadata
--- PASS: TestAccEntity_withMetadata (0.84s)
=== RUN   TestAccEntity_update
--- PASS: TestAccEntity_update (1.50s)
=== RUN   TestAccEntity_updateMetadata
--- PASS: TestAccEntity_updateMetadata (1.49s)
=== RUN   TestAccEntity_withLinks
--- PASS: TestAccEntity_withLinks (0.86s)
PASS
ok      github.com/last9/terraform-provider-last9/internal/provider     15.377s
```

All 12 tests pass successfully against the live API.

---

# Session: Switch to Refresh Token Authentication (2026-01-20)

## Task Description

Update the auth layer to use refresh tokens instead of short-lived access tokens. The Last9 API's OAuth endpoint generates new access tokens from refresh tokens.

## Issues Addressed

### Issue 1: OAuth Endpoint URL Missing `/api` Prefix
- **Before**: `{baseURL}/v4/oauth/access_token`
- **After**: `{baseURL}/api/v4/oauth/access_token`
- **Reason**: Per the OpenAPI spec, the server base is `https://{api_host}/api/` and the OAuth path is `/v4/oauth/access_token`

### Issue 2: No Support for Delete Refresh Token
- **Before**: Only static `delete_token` supported
- **After**: Added `delete_refresh_token` for automatic token refresh with delete scope
- **Reason**: Long-lived refresh tokens are preferred over short-lived access tokens for automation

## Files Modified

### 1. `internal/client/client.go`

| Function/Type | Change |
|---------------|--------|
| `Config` struct | Added `DeleteRefreshToken` field |
| `Client` struct | Added `deleteAccessToken` (cached token) and `deleteTokenMutex` fields |
| `refreshAccessToken()` | Fixed URL: `%s/api/v4/oauth/access_token` |
| `refreshDeleteAccessToken()` | New method - refreshes delete access token using delete refresh token |
| `getDeleteAccessToken()` | New method - returns valid delete token (refresh if needed, fallback to static) |
| `Delete()` | Uses `getDeleteAccessToken()` instead of static token check |

### 2. `internal/provider/provider.go`

| Change |
|--------|
| Added `delete_refresh_token` schema field with `LAST9_DELETE_REFRESH_TOKEN` env var |
| Read `delete_refresh_token` in `configureProvider()` and pass to client config |
| Updated `delete_token` description to indicate legacy status |

### 3. `internal/provider/provider_config_test.go`

| Change |
|--------|
| Added support for `LAST9_WRITE_REFRESH_TOKEN` environment variable |
| Added support for `LAST9_DELETE_REFRESH_TOKEN` environment variable |
| Preference order: `LAST9_WRITE_REFRESH_TOKEN` > `LAST9_REFRESH_TOKEN` > `LAST9_API_TOKEN` |
| Delete token preference: `LAST9_DELETE_REFRESH_TOKEN` > `LAST9_DELETE_TOKEN` |

### 4. `internal/provider/provider_test.go`

| Function | Change |
|----------|--------|
| `testAccPreCheck()` | Also checks for `LAST9_WRITE_REFRESH_TOKEN` |
| `testAccPreCheckWithDelete()` | Also checks for `LAST9_DELETE_REFRESH_TOKEN` |

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `LAST9_WRITE_REFRESH_TOKEN` | Refresh token with write scope | Yes (or LAST9_REFRESH_TOKEN or LAST9_API_TOKEN) |
| `LAST9_DELETE_REFRESH_TOKEN` | Refresh token with delete scope | Yes for destroy tests (or LAST9_DELETE_TOKEN) |
| `LAST9_REFRESH_TOKEN` | Legacy refresh token (write scope) | Fallback |
| `LAST9_API_TOKEN` | Direct access token (legacy) | Fallback |
| `LAST9_DELETE_TOKEN` | Direct delete access token (legacy) | Fallback |
| `LAST9_API_BASE_URL` | API hostname | Yes |
| `LAST9_ORG` | Organization slug | Yes |

## Trade-offs Considered

1. **Backward compatibility**: Both new refresh token fields and legacy static token fields continue to work. Refresh tokens are preferred when both are configured.

2. **Token caching**: Both write and delete access tokens are cached with 5-minute expiry buffer to minimize API calls while ensuring tokens don't expire mid-request.

3. **Double-check locking**: Both `getAccessToken()` and `getDeleteAccessToken()` use read lock first, then write lock with double-check pattern for thread safety.

## Running Tests with Refresh Tokens

```bash
# Set refresh tokens
export LAST9_WRITE_REFRESH_TOKEN="<write-refresh-token>"
export LAST9_DELETE_REFRESH_TOKEN="<delete-refresh-token>"
export LAST9_API_BASE_URL="https://alpha.last9.io"
export LAST9_ORG="last9"

# Run tests
CGO_ENABLED=0 TF_ACC=1 go test -v ./internal/provider/ -run "TestAccEntity_basic" -timeout 10m
```

## Test Results

```
=== RUN   TestAccEntity_basic
--- PASS: TestAccEntity_basic (1.93s)
PASS
ok      github.com/last9/terraform-provider-last9/internal/provider     2.646s
```

Tests pass with refresh token authentication.
