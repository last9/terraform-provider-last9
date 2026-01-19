# Terraform Provider for Last9 - Examples

This directory contains example Terraform configurations demonstrating how to use the Last9 Terraform provider.

## Examples

### Basic Examples

- **main.tf** - Complete example with all resource types
- **dashboard/** - Dashboard-only configuration
- **alerts/** - Alert configuration examples
- **macros/** - Macro configuration examples
- **policies/** - Policy configuration examples
- **scheduled-search-alerts/** - Log-based scheduled search alert examples

## Usage

1. Set up your Last9 credentials:

```bash
export LAST9_API_TOKEN="your-api-token"
export LAST9_ORG="your-org-slug"
```

Or create a `terraform.tfvars` file:

```hcl
last9_api_token = "your-api-token"
last9_org       = "your-org-slug"
```

2. Initialize Terraform:

```bash
terraform init
```

3. Review the plan:

```bash
terraform plan
```

4. Apply the configuration:

```bash
terraform apply
```

## Example Configurations

### Dashboard Example

Creates a dashboard with multiple panels for monitoring service metrics.

### Alert Example

Demonstrates both static threshold alerts and expression-based alerts.

### Macro Example

Shows how to define cluster-level macros for query templating.

### Policy Example

Creates control plane policies with multiple compliance rules.

### Scheduled Search Alert Example

Demonstrates log-based alerts using custom queries, aggregation functions, and notification routing.

## Notes

- Replace placeholder values (like `entity_id`, `cluster_id`) with actual values from your Last9 organization
- Review and adjust thresholds and configurations to match your requirements
- Some resources may depend on others (e.g., alerts depend on entities)

