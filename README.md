# Terraform Provider for Last9

A Terraform provider for managing Last9 resources including alerts, macros, control plane policies, log management rules, and scheduled search alerts.

## Features

- **Alerts**: Configure alert rules with static thresholds or expressions
- **Macros**: Manage cluster-level macros for query templating
- **Policies**: Define and enforce control plane rules
- **Drop Rules**: Configure log drop rules for filtering and cost optimization
- **Forward Rules**: Set up log forwarding to external destinations
- **Scheduled Search Alerts**: Create log-based alerts with custom queries and thresholds
- **Notification Destinations**: Query available notification channels (data source)

## Installation

### Using Terraform Registry (Recommended)

Add the provider to your Terraform configuration:

```hcl
terraform {
  required_providers {
    last9 = {
      source  = "last9/last9"
      version = "~> 1.0"
    }
  }
}
```

### Building from Source

```bash
git clone https://github.com/last9/terraform-provider-last9
cd terraform-provider-last9
go build -o terraform-provider-last9
```

Place the binary in your Terraform plugins directory:
- Linux/Mac: `~/.terraform.d/plugins/registry.terraform.io/last9/last9/1.0.0/linux_amd64/`
- Windows: `%APPDATA%\terraform.d\plugins\registry.terraform.io\last9\last9\1.0.0\windows_amd64\`

## Configuration

Configure the provider with your Last9 credentials. You can use either refresh tokens (recommended) or direct access tokens.

### Using Refresh Tokens (Recommended)

```hcl
provider "last9" {
  refresh_token = var.last9_refresh_token  # or use LAST9_REFRESH_TOKEN env var
  org           = var.last9_org            # or use LAST9_ORG env var
  api_base_url  = var.last9_api_base_url   # required - or use LAST9_API_BASE_URL env var
}
```

### Using Direct Access Tokens (Legacy)

```hcl
provider "last9" {
  api_token    = var.last9_api_token     # or use LAST9_API_TOKEN env var
  org          = var.last9_org           # or use LAST9_ORG env var
  api_base_url = var.last9_api_base_url  # required - or use LAST9_API_BASE_URL env var
}
```

### Environment Variables

- `LAST9_REFRESH_TOKEN` - Your Last9 refresh token (recommended)
- `LAST9_API_TOKEN` - Your Last9 API access token (legacy)
- `LAST9_ORG` - Your Last9 organization slug
- `LAST9_API_BASE_URL` - API base URL (required)

**Note**: Either `LAST9_REFRESH_TOKEN` or `LAST9_API_TOKEN` must be provided. Refresh tokens are recommended as they automatically handle token refresh.

See [Authentication Guide](docs/AUTHENTICATION.md) for detailed information about authentication, token scopes, and RBAC.

## Usage Examples

See the [examples](./examples/) directory for complete examples.

### Alert

```hcl
resource "last9_alert" "high_error_rate" {
  entity_id   = last9_entity.example.id
  name        = "High Error Rate"
  description = "Alert when error rate exceeds 100 req/min"
  indicator   = "error_rate"
  
  greater_than  = 100
  bad_minutes   = 5
  total_minutes = 10
  
  severity = "breach"
  
  properties {
    runbook_url = "https://wiki.example.com/runbooks/high-error-rate"
    annotations = {
      priority = "high"
      team     = "platform"
    }
  }
}
```

### Macro

```hcl
resource "last9_macro" "example" {
  cluster_id = "cluster-123"
  
  body = jsonencode({
    macros = {
      "service_name" = "$service"
      "environment"  = "$env"
    }
  })
}
```

### Policy

```hcl
resource "last9_policy" "slo_compliance" {
  name        = "SLO Compliance Policy"
  description = "Ensures all services meet SLO requirements"

  filters = {
    entity_type = "service"
    tags        = "production"
  }

  rules {
    type = "slo_compliance"
    config = {
      slo_name          = "availability"
      threshold         = "99.9"
      evaluation_window = "30d"
    }
  }
}
```

### Drop Rule

```hcl
resource "last9_drop_rule" "debug_logs" {
  region    = "us-west-2"
  name      = "drop-debug-logs"
  telemetry = "logs"

  filters {
    key         = "attributes[\"severity\"]"
    value       = "debug"
    operator    = "equals"
    conjunction = "and"
  }
}
```

### Forward Rule

```hcl
resource "last9_forward_rule" "external_logs" {
  region      = "us-west-2"
  name        = "forward-critical-logs"
  telemetry   = "logs"
  destination = "https://logs.external-system.com/webhook"

  filters {
    key         = "attributes[\"severity\"]"
    value       = "critical"
    operator    = "equals"
    conjunction = "and"
  }

  filters {
    key         = "attributes[\"service\"]"
    value       = "payment-service"
    operator    = "equals"
    conjunction = "and"
  }
}
```

## Resources

- `last9_entity` - Manage entities (services, components)
- `last9_alert` - Configure alert rules
- `last9_macro` - Manage cluster macros
- `last9_policy` - Define control plane policies
- `last9_drop_rule` - Configure log drop rules for filtering
- `last9_forward_rule` - Set up log forwarding to external destinations
- `last9_scheduled_search_alert` - Create log-based scheduled search alerts

## Data Sources

- `last9_entity` - Query entity information
- `last9_notification_destination` - Query notification destinations for alerts

## Development

### Prerequisites

- Go 1.21 or later
- Terraform 1.0 or later

### Building

```bash
go build -o terraform-provider-last9
```

### Testing

```bash
go test ./...
```

### Running Acceptance Tests

```bash
TF_ACC=1 go test ./... -v
```

## Contributing

Contributions are welcome! Please read our contributing guidelines first.

## License

MIT License - see LICENSE file for details

## Support

For issues and questions:
- GitHub Issues: https://github.com/last9/terraform-provider-last9/issues
- Documentation: https://docs.last9.io

