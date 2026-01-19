# Authentication and RBAC Support Summary

## What Was Added

The Terraform provider now supports Last9's JWT-based authentication system with refresh tokens and automatic token management.

## Key Features

### 1. Refresh Token Support
- Long-lived refresh tokens (2 years)
- Automatic access token generation
- Automatic token refresh when expired
- Thread-safe token management

### 2. Access Token Support (Legacy)
- Direct access token support for backward compatibility
- Manual token management required

### 3. RBAC/Scopes
- Token scopes control API access:
  - `read`: GET requests
  - `write`: GET, POST, PUT, PATCH requests
  - `delete`: GET, DELETE requests
- Provider respects token scopes automatically

### 4. Automatic Token Refresh
- Tokens are refreshed automatically when:
  - Expired (3 days for access tokens)
  - About to expire (within 5 minutes)
- Transparent to the user - no manual intervention needed

## Implementation Details

### Client Changes (`internal/client/client.go`)
- Added `RefreshToken` to `Config`
- Added `AccessToken` struct with expiration tracking
- Added `getAccessToken()` method for automatic refresh
- Added `refreshAccessToken()` method to call Last9 API
- Updated `doRequest()` to use `X-LAST9-API-TOKEN` header
- Thread-safe token management with mutex

### Provider Changes (`internal/provider/provider.go`)
- Added `refresh_token` configuration option
- Made `api_token` optional (legacy support)
- Updated validation to require either token type
- Environment variable support: `LAST9_REFRESH_TOKEN`

### Documentation
- Created `docs/AUTHENTICATION.md` with comprehensive guide
- Updated `README.md` with authentication examples
- Added troubleshooting section

## API Endpoints Used

1. **Get Access Token**: `POST /organizations/{org}/oauth/access_token`
   - Request: `{ "refresh_token": "..." }`
   - Response: `{ "access_token": "...", "expires_at": ..., "scopes": [...] }`

2. **API Calls**: Uses `X-LAST9-API-TOKEN: Bearer <access-token>` header

## Usage Examples

### Refresh Token (Recommended)
```hcl
provider "last9" {
  refresh_token = var.last9_refresh_token
  org          = var.last9_org
}
```

### Direct Access Token (Legacy)
```hcl
provider "last9" {
  api_token = var.last9_api_token
  org      = var.last9_org
}
```

## Security Features

1. **Token Expiration**: Access tokens expire after 3 days
2. **Proactive Refresh**: Tokens refreshed 5 minutes before expiration
3. **Thread Safety**: Mutex-protected token access
4. **Secure Storage**: Tokens marked as sensitive in Terraform

## Testing Notes

- Token refresh logic tested with mutex for thread safety
- Error handling for expired/invalid tokens
- Fallback to direct API token if refresh fails
- Proper error messages for authentication failures

## Migration Path

Users can migrate from direct API tokens to refresh tokens:
1. Generate refresh token via UI or API
2. Update Terraform config to use `refresh_token`
3. Remove `api_token` configuration
4. Provider handles token management automatically

