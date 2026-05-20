# Changelog

All notable changes to the Last9 Terraform Provider will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **last9_entity** - `renotify_enabled`, `renotify_interval_seconds`, and `renotify_occurrences` fields for per-alert-group repeat notification control (ENG-899)
  - `renotify_enabled = false` — notify-once: only the first firing notification and the resolved notification are sent
  - `renotify_interval_seconds` — seconds between repeat notifications while firing (must be ≥ 1)
  - `renotify_occurrences` — cap on repeat notifications per firing episode (`-1` = unlimited, or ≥ 1)
  - Omitting all three fields inherits the tenant default (re-notify every hour)
  - Removing all three fields from an existing config resets the group to tenant defaults

## [0.4.1] - 2026-05-19

### Fixed

- **last9_dashboard** - `panel.unit` and `query.unit` no longer silently drop `unit = ""` from the API payload (`json:"unit,omitempty"` removed). Previously the Last9 API retained the previously-stored unit, defaulting to `"percent"` on stat panels and `"seconds"` on timeseries with no error.
- **last9_dashboard** - `panel.unit` schema no longer has `Computed: true`; `terraform plan` now correctly diffs between `""` and any previously-set unit value.
- **last9_dashboard** - `panel.alert.greater_than` / `less_than` thresholds of `0` are now supported. Previously `GetOk` returned `(0.0, false)` for zero-value floats, making `threshold = 0` indistinguishable from an omitted value.

### Changed

- **last9_dashboard** - `panel.unit` and `query.unit` now validate against an allowlist at plan time. **Breaking for configs using unrecognised values.** Migrate Grafana-style IDs before upgrading: `"ms"` → `"milliseconds"`, `"s"` → `"seconds"`, `"percentunit"` → `"percent"` (and multiply the PromQL value by 100). Any other unrecognised string should be removed or replaced with `""`.

## [0.4.0] - 2026-05-15

### Added

- **last9_notification_channel** - Slack App mode support (`slack_app` type with `channel_id` and bot token authentication) (ENG-910)

## [0.3.0] - 2026-05-06

### Added

- **last9_dashboard** - Manage Last9 dashboards as code (ENG-1013)
- **last9_remapping_rule** - Configure OpenTelemetry remapping rules via the `otel_settings` API

## [0.2.2] - 2026-04-17

### Changed
- Re-release of v0.2.1 (same source commit) to refresh registry artifacts

## [0.2.1] - 2026-03-10

### Added
- Webhook headers support for `generic_webhook` notification channels (#8)
- Debug logging and response validation in the HTTP client

### Changed
- Examples updated to reference the published `last9/last9` provider (v0.2)
- README version references aligned with the v0.2.1 release (#12)
- Dependency bump: `github.com/cloudflare/circl` 1.6.1 → 1.6.3 (#10)
- Dependency bump: `google.golang.org/grpc` to 1.79.3 (#13)

### Documentation
- Documented `entity_class` requirement for alert groups (#11)

## [0.2.0] - 2026-01-23

### Changed

#### Documentation
- Improved authentication documentation with token expiry details and delete token requirements
- Added links to Last9 documentation throughout provider docs
- Reframed `last9_entity` as "alert groups" for clarity
- Added "Alerting Lifecycle" section explaining metric-based and log-based alerting flows
- Updated notification channel docs with supported types table

#### Removed
- Removed `last9_policy` resource (unused, not integrated with other features)
- Removed Opsgenie from documented notification channel types

### Added
- Documentation links to Last9 docs:
  - [API Getting Started](https://last9.io/docs/getting-started-with-api/)
  - [Control Plane](https://last9.io/docs/control-plane/)
  - [Drop Rules](https://last9.io/docs/control-plane-drop/)
  - [Forward Rules](https://last9.io/docs/control-plane-forward/)
  - [PromQL Macros](https://last9.io/docs/promql-macros/)
  - [Alerting Overview](https://last9.io/docs/alerting-overview/)
  - [Notification Channels](https://last9.io/docs/notification-channels/)

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
