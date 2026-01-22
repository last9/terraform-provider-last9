# Terraform Provider for Last9 - Examples

This directory contains example Terraform configurations demonstrating how to use the Last9 Terraform provider.

## Examples

- **entity-with-alerts/** - Entity with alert configuration
- **alerts/** - Alert configuration examples
- **macros/** - Macro configuration examples
- **policies/** - Policy configuration examples
- **scheduled-search-alerts/** - Log-based scheduled search alert examples

## Usage

1. Set up your Last9 credentials:

```bash
export LAST9_REFRESH_TOKEN="your-refresh-token"
export LAST9_ORG="your-org-slug"
export LAST9_API_BASE_URL="https://app.last9.io"
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

## Notes

- Replace placeholder values with actual values from your Last9 organization
- Review and adjust thresholds and configurations to match your requirements
- Some resources may depend on others (e.g., alerts depend on entities)
