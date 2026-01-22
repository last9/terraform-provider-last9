# Changelog

All notable changes to the Last9 Terraform Provider will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-01-14

### Added

#### Resources
- **last9_entity** - Manage entities (services, components) with KPIs and alerts
- **last9_alert** - Configure alert rules with static thresholds or expressions
- **last9_notification_channel** - Manage notification channels for alerts
- **last9_macro** - Manage cluster-level macros for query templating
- **last9_policy** - Define and enforce control plane policies
- **last9_drop_rule** - Configure log drop rules for filtering and cost optimization
- **last9_forward_rule** - Set up log forwarding to external destinations
- **last9_scheduled_search_alert** - Create log-based scheduled search alerts with:
  - Custom LogJSON query pipelines
  - Aggregation functions ($count, $sum, $avg, $max, $min)
  - Grouping capabilities
  - Configurable search frequency
  - Threshold-based alerting
  - Multiple notification destinations

#### Data Sources
- **last9_entity** - Query entity information
- **last9_notification_destination** - Query notification destinations for alerts

#### Authentication
- Refresh token support (recommended) with automatic token refresh
- Direct API token support (legacy)
- JWT-based authentication with 3-day access tokens
- Thread-safe token management with double-checked locking

#### Features
- Comprehensive input validation for all resources
- Import support for all resources
- Detailed inline documentation with 17+ Description fields per resource
- Thread-safe concurrent operations
- Proper error handling with error wrapping

#### Documentation
- Complete README with usage examples
- Examples for all resources including:
  - Basic usage examples
  - Entity configurations
  - Alert setups
  - Log management rules
  - Scheduled search alerts with multiple patterns

#### Testing
- Unit tests for helper functions
- Acceptance tests for all resources
- Test coverage for:
  - Basic CRUD operations
  - Update scenarios
  - Complex configurations (grouping, multiple aggregates)
  - Import functionality
  - Edge cases and error conditions

### Technical Details

#### API Compatibility
- Last9 API v1
- Supports multi-region deployments
- Compatible with all Last9 telemetry types (logs, traces, metrics)

#### Provider Configuration
- Organization-based authentication
- Configurable API base URL
- Environment variable support (LAST9_REFRESH_TOKEN, LAST9_API_TOKEN, LAST9_ORG)

#### Security
- No hardcoded credentials
- Secure token storage and refresh
- No sensitive data in logs
- Proper input sanitization
- Thread-safe operations

### Breaking Changes
- Initial release, no breaking changes

### Known Limitations
- Acceptance tests require macOS workaround for dyld LC_UUID issue
- Tests must be run in CI/CD or Linux environment for full validation

### Contributors
- Prathamesh (@prathamesh2_)
- Nishant

---

## Release Notes

This is the first stable release (v1.0.0) of the Last9 Terraform Provider. The provider enables infrastructure-as-code management of Last9 resources including entities, alerts, log management rules, and scheduled search alerts.

### Installation

Add to your Terraform configuration:

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

### Getting Started

1. Set up authentication:
```bash
export LAST9_REFRESH_TOKEN="your-refresh-token"
export LAST9_ORG="your-org-slug"
export LAST9_API_BASE_URL="https://app.last9.io"
```

2. Configure the provider:
```hcl
provider "last9" {
  refresh_token = var.last9_refresh_token
  org           = var.last9_org
  api_base_url  = var.last9_api_base_url
}
```

3. Start creating resources! See [examples/](./examples/) for complete examples.

### Upgrading

This is the first release, no upgrade path needed.

### Support

- Documentation: https://docs.last9.io
- GitHub Issues: https://github.com/last9/terraform-provider-last9/issues
- Examples: [examples/](./examples/)

[1.0.0]: https://github.com/last9/terraform-provider-last9/releases/tag/v1.0.0
