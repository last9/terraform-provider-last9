# Last9 Authentication Guide for Terraform Provider

## Overview

Last9 uses JWT-based authentication with refresh tokens and access tokens. The Terraform provider supports both authentication methods.

## Authentication Methods

### 1. Refresh Token (Recommended)

Refresh tokens are long-lived (2 years) and automatically generate short-lived access tokens (3 days). This is the recommended approach for automated systems.

**Advantages:**
- Automatic token refresh
- Better security (short-lived access tokens)
- No manual token management needed

**Configuration:**

```hcl
provider "last9" {
  refresh_token = var.last9_refresh_token
  org          = var.last9_org
}
```

**Environment Variable:**
```bash
export LAST9_REFRESH_TOKEN="your-refresh-token"
export LAST9_ORG="your-org-slug"
```

### 2. Direct Access Token (Legacy)

Direct access tokens can be used but require manual refresh when expired.

**Configuration:**

```hcl
provider "last9" {
  api_token = var.last9_api_token
  org      = var.last9_org
}
```

**Environment Variable:**
```bash
export LAST9_API_TOKEN="your-access-token"
export LAST9_ORG="your-org-slug"
```

## Getting Refresh Tokens

### Via Last9 UI

1. Log in to Last9 dashboard
2. Navigate to Settings → API Tokens
3. Create a new refresh token with desired scopes
4. Copy the refresh token

### Via API

```bash
curl -X POST "https://api.last9.io/organizations/{org}/refresh_token" \
  -H "Authorization: Bearer <your-auth0-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "scopes": ["read", "write", "delete"]
  }'
```

## Token Scopes

Last9 tokens support three scopes that control API access:

- **read**: Allows GET requests
- **write**: Allows GET, POST, PUT, PATCH requests
- **delete**: Allows GET, DELETE requests

**Example with scopes:**

```hcl
# Read-only access
provider "last9" {
  refresh_token = var.last9_refresh_token_readonly
  org          = var.last9_org
}

# Full access
provider "last9" {
  refresh_token = var.last9_refresh_token_full
  org          = var.last9_org
}
```

## How Token Refresh Works

When using refresh tokens:

1. Provider fetches an access token on initialization
2. Access token is cached and reused for API calls
3. Token is automatically refreshed when:
   - It expires (3 days)
   - It's about to expire (within 5 minutes)
4. Refresh happens transparently - no user intervention needed

## RBAC and Permissions

### Organization Access

Tokens are scoped to a specific organization. The `org` parameter must match the organization in the token.

### Scope-Based Access Control

The provider respects token scopes:

- **read scope**: Can read resources (GET)
- **write scope**: Can create/update resources (POST, PUT, PATCH)
- **delete scope**: Can delete resources (DELETE)

If a token lacks required scopes, API calls will fail with 403 Forbidden.

## Security Best Practices

1. **Use Refresh Tokens**: Prefer refresh tokens over direct access tokens
2. **Minimal Scopes**: Use the minimum scopes required for your use case
3. **Secure Storage**: Store tokens securely:
   - Use Terraform variables with `sensitive = true`
   - Use environment variables
   - Never commit tokens to version control
4. **Rotate Tokens**: Regularly rotate refresh tokens
5. **Separate Environments**: Use different tokens for different environments

## Example Configuration

### Complete Example

```hcl
terraform {
  required_providers {
    last9 = {
      source  = "last9/last9"
      version = "~> 1.0"
    }
  }
}

provider "last9" {
  refresh_token = var.last9_refresh_token
  org          = var.last9_org
  api_base_url = "https://api.last9.io"
}

variable "last9_refresh_token" {
  description = "Last9 refresh token"
  type        = string
  sensitive   = true
}

variable "last9_org" {
  description = "Last9 organization slug"
  type        = string
}
```

### Using Environment Variables

```bash
# Set environment variables
export LAST9_REFRESH_TOKEN="your-refresh-token"
export LAST9_ORG="your-org-slug"

# Run Terraform
terraform init
terraform plan
terraform apply
```

### Using Terraform Variables File

Create `terraform.tfvars`:

```hcl
last9_refresh_token = "your-refresh-token"
last9_org          = "your-org-slug"
```

**Important**: Add `terraform.tfvars` to `.gitignore`!

## Troubleshooting

### Token Expired Error

**Error**: `failed to refresh access token: API error: 401`

**Solution**: Your refresh token may have expired or been revoked. Generate a new refresh token.

### Invalid Organization Error

**Error**: `You do not have access to this organization`

**Solution**: Ensure the `org` parameter matches the organization in your token.

### Insufficient Permissions Error

**Error**: `API error: 403 Forbidden`

**Solution**: Your token lacks required scopes. Create a new token with appropriate scopes (read, write, delete).

### Token Format Error

**Error**: `invalid access token`

**Solution**: Ensure you're using the correct token format:
- Access tokens: Use directly
- Refresh tokens: Use with `refresh_token` parameter

## Migration from API Token to Refresh Token

If you're currently using direct API tokens:

1. Generate a refresh token via UI or API
2. Update your Terraform configuration:

```hcl
# Before
provider "last9" {
  api_token = var.last9_api_token
  org      = var.last9_org
}

# After
provider "last9" {
  refresh_token = var.last9_refresh_token
  org          = var.last9_org
}
```

3. Update environment variables or tfvars files
4. Run `terraform init` to reinitialize

## API Endpoints Reference

- **Create Refresh Token**: `POST /organizations/{org}/refresh_token`
- **Get Access Token**: `POST /organizations/{org}/oauth/access_token`
- **Token Header**: `X-LAST9-API-TOKEN: Bearer <access-token>`

## Additional Resources

- [Last9 API Documentation](https://apidocs.last9.io/)
- [JWT Token Guide](https://docs.last9.io/guides/api-tokens)

