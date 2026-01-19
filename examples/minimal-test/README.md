# Minimal Last9 Provider Test

This directory contains a minimal test example for the Last9 Terraform provider. It creates only essential resources to verify the provider is working correctly.

## 🎯 Purpose

This minimal test is designed to:
- Quickly verify provider installation and configuration
- Test core provider functionality with minimal resources
- Provide fast feedback during development
- Serve as a smoke test for CI/CD pipelines

## 📋 What This Creates

- **1 Dashboard** - Simple dashboard with one panel
- **1 Alert** - Basic alert rule
- **1 Drop Rule** - Simple log drop rule
- **1 Data Source** - Entity lookup to verify connectivity

## 🚀 Quick Start

### 1. Setup Configuration

```bash
# Copy example configuration
cp terraform.tfvars.example terraform.tfvars

# Edit with your credentials
nano terraform.tfvars
```

Required variables:
```hcl
last9_org = "your-org-name"
last9_refresh_token = "your-refresh-token"
entity_name = "your-entity-name"
```

### 2. Run Test

```bash
# Initialize
terraform init

# Plan
terraform plan

# Apply
terraform apply

# Verify outputs
terraform output

# Cleanup
terraform destroy
```

### 3. Automated Test

Use the automated script from the parent integration-test directory:

```bash
# Set environment variables
export LAST9_ORG="your-org"
export LAST9_REFRESH_TOKEN="your-token"
export LAST9_ENTITY_NAME="your-entity"

# Run from integration-test directory
cd ../integration-test
./run-integration-test.sh --dry-run
```

## ✅ Expected Results

After successful apply, you should see outputs like:

```json
{
  "test_results": {
    "alert_created": "alert-id-here",
    "dashboard_created": "dashboard-id-here",
    "drop_rule_created": "rule-id-here",
    "entity_found": true,
    "test_suffix": "abc12345"
  },
  "validation_summary": {
    "data_sources_working": true,
    "provider_working": true,
    "resources_created": 3,
    "test_id": "abc12345"
  }
}
```

## 🔍 Validation

The test validates:
- ✅ Provider connectivity to Last9 API
- ✅ Authentication working correctly
- ✅ Entity data source functionality
- ✅ Dashboard resource creation
- ✅ Alert resource creation
- ✅ Drop rule resource creation
- ✅ Resource naming and tagging

## 🚨 Troubleshooting

### Common Issues

**Authentication Error:**
```
Error: failed to create Last9 client
```
- Check `last9_refresh_token` is correct
- Verify `last9_org` matches your organization

**Entity Not Found:**
```
Error: entity not found
```
- Verify `entity_name` exists in your Last9 organization
- Check entity is accessible with your credentials

**Resource Conflicts:**
```
Error: resource already exists
```
- Run `terraform destroy` to clean up previous test resources
- The test uses random suffixes to avoid conflicts

## 🔧 Development Usage

This minimal test is perfect for:

- **Local Development** - Quick provider verification
- **CI/CD Pipelines** - Smoke testing before full integration tests
- **Debugging** - Isolating provider issues
- **Learning** - Understanding basic provider usage

## ⏱️ Performance

Expected runtime:
- Plan: ~10 seconds
- Apply: ~30 seconds
- Destroy: ~20 seconds

Total test time: ~1 minute

## 📝 Notes

- Resources are tagged with `minimal-test` for easy identification
- Random suffixes prevent naming conflicts
- All resources are safe to create and destroy repeatedly
- This test uses minimal API quota