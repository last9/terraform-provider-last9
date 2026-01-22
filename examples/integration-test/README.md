# Last9 Terraform Provider - Integration Test Example

This directory contains a comprehensive integration test example that demonstrates all features of the Last9 Terraform provider. It creates a complete monitoring setup including dashboards, alerts, macros, policies, and log management rules.

## 🚀 Quick Start

### For Local Development/Testing (Before Provider Release)

1. **Build the provider locally:**
   ```bash
   # From the root of the terraform-provider repository
   make build
   make install  # Installs to local Terraform plugins directory
   ```

2. **Use local development configuration:**
   ```bash
   # Use versions.tf instead of the main terraform configuration
   mv main.tf main.tf.published
   mv versions.tf main.tf
   ```

### For Published Provider Usage

1. **Copy the example configuration:**
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   ```

2. **Edit `terraform.tfvars` with your Last9 credentials and configuration:**
   ```bash
   # Required fields
   last9_org = "your-org-name"
   last9_refresh_token = "your-refresh-token"  # or use last9_api_token
   cluster_id = "your-cluster-id"
   entity_name = "your-entity-name"

   # Optional customizations
   environment = "integration-test"
   service_name = "my-api-service"
   ```

3. **Initialize and apply:**
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

## 📋 What This Example Creates

### 🎛️ **Dashboard Resources**
- **Production Monitoring Dashboard** with panels for:
  - Request Rate (req/min)
  - Error Rate (%)
  - Response Time P95 (ms)
  - Service Availability (%)

### 🚨 **Alert Resources**
- **High Error Rate Alert**: Triggers when error rate > 5% (configurable)
- **Low Availability Alert**: Triggers when availability < 99.5% (configurable)
- **High Response Time Alert**: Expression-based alert for P95 latency

### 🔧 **Macro Resources**
- **Environment Macros**: Pre-configured query templates for:
  - Service name and environment variables
  - Error rate calculation query
  - Request rate calculation query
  - P95 latency query
  - Availability calculation query

### 📋 **Policy Resources**
- **SLO Compliance Policy** with rules for:
  - Availability SLO monitoring (99.9% target)
  - Latency SLO monitoring (200ms target)
  - Alert coverage requirements (minimum 2 alerts)

### 📝 **Log Management Resources**
- **Drop Rules**:
  - Drop debug-level logs to reduce costs
  - Drop test environment logs (in non-test environments)
- **Forward Rules**:
  - Forward critical errors to external systems
  - Forward security logs to SIEM
  - Forward distributed traces for analysis

## 🔧 Configuration Options

### **Required Variables**
```hcl
last9_org                 # Your Last9 organization slug
last9_refresh_token      # Authentication token (recommended)
# OR last9_api_token      # Legacy authentication method
cluster_id               # Last9 cluster ID for macros
entity_name              # Entity to monitor
```

### **Environment Configuration**
```hcl
environment              # Environment name (default: "integration-test")
region                   # AWS/deployment region (default: "us-west-2")
service_name            # Service name (default: "api-service")
team_name               # Responsible team (default: "platform")
```

### **Alert Thresholds**
```hcl
error_rate_threshold     # Error rate % (default: 5.0)
availability_threshold   # Availability % (default: 99.5)
response_time_threshold  # P95 latency ms (default: 500)
alert_bad_minutes       # Bad duration (default: 5)
alert_total_minutes     # Eval window (default: 10)
```

### **External Integrations**
```hcl
runbook_base_url        # Runbook documentation URL
external_log_destination # Critical error forwarding URL
security_log_destination # Security log forwarding URL
trace_destination       # Trace forwarding URL
```

## 🧪 Integration Testing

### **Validation Commands**
```bash
# Validate configuration
terraform validate

# Check what will be created
terraform plan

# Apply and create resources
terraform apply

# View outputs
terraform output

# Destroy resources (cleanup)
terraform destroy
```

### **Output Validation**
The example provides comprehensive outputs for testing:

```bash
# View all resource IDs
terraform output alert_ids
terraform output dashboard_id
terraform output policy_id

# View configuration summary
terraform output integration_test_summary

# View validation URLs
terraform output validation_urls
```

### **Manual Verification**
After applying, verify resources in the Last9 console:

1. **Dashboard**: Visit the dashboard URL from outputs
2. **Alerts**: Check alerts are created and configured correctly
3. **Policies**: Verify SLO compliance policy rules
4. **Log Rules**: Confirm drop and forward rules are active

## 📊 Expected Outputs

### **Resource Summary**
```yaml
Resources Created:
- 1 Dashboard (with 4 panels)
- 3 Alerts (error rate, availability, response time)
- 1 Macro set (5 query templates)
- 1 Policy (3 compliance rules)
- 1-2 Drop rules (debug logs, test logs)
- 0-3 Forward rules (based on configuration)
```

### **Integration Test Summary**
The `integration_test_summary` output provides:
- Resource counts and validation
- Configuration verification
- External integration status
- Authentication method confirmation

## 🔍 Troubleshooting

### **Common Issues**

1. **Authentication Errors**
   ```bash
   Error: failed to create Last9 client
   ```
   - Verify `last9_refresh_token` or `last9_api_token` is correct
   - Check `last9_org` matches your organization slug

2. **Entity Not Found**
   ```bash
   Error: entity not found
   ```
   - Verify `entity_name` exists in your Last9 organization
   - Check entity is in the correct cluster/environment

3. **Cluster ID Invalid**
   ```bash
   Error: cluster not found
   ```
   - Get correct `cluster_id` from Last9 console
   - Ensure you have access to the cluster

### **Debug Mode**
Enable Terraform debug logging:
```bash
export TF_LOG=DEBUG
terraform plan
```

### **Provider Debug**
Enable provider debug mode by setting:
```bash
export TF_LOG_PROVIDER=DEBUG
```

## 🔒 Security Best Practices

### **Credential Management**
- Never commit `terraform.tfvars` to version control
- Use environment variables for sensitive data:
  ```bash
  export TF_VAR_last9_refresh_token="your-token"
  ```
- Consider using Terraform Cloud or encrypted backends

### **Resource Isolation**
- Use different workspaces for different environments
- Apply principle of least privilege for API tokens
- Regular token rotation

## 🚀 Advanced Usage

### **Custom Environments**
```bash
# Create multiple environment configs
cp terraform.tfvars terraform.tfvars.staging
cp terraform.tfvars terraform.tfvars.production

# Use workspace isolation
terraform workspace new staging
terraform apply -var-file="terraform.tfvars.staging"
```

### **CI/CD Integration**
```bash
# In your CI pipeline
terraform init -backend-config="backend.hcl"
terraform plan -var-file="$ENVIRONMENT.tfvars"
terraform apply -auto-approve -var-file="$ENVIRONMENT.tfvars"
```

### **Module Usage**
This example can be converted to a reusable module:
```hcl
module "last9_monitoring" {
  source = "./examples/integration-test"

  last9_org = var.organization
  cluster_id = var.cluster_id
  environment = "production"
  # ... other variables
}
```

## 📝 Notes

- This example is designed to be safe to run multiple times
- Resources are tagged with `integration-test` for easy identification
- All resource names include the environment prefix to avoid conflicts
- Conditional resources (forward rules) are only created when destinations are provided
- The example follows Last9 and Terraform best practices

## 🤝 Contributing

To improve this integration test:
1. Test with your Last9 environment
2. Report issues or suggest improvements
3. Add additional use cases or scenarios
4. Enhance validation and error handling

## 📚 Related Documentation

- [Last9 Terraform Provider Documentation](../../README.md)
- [Last9 API Documentation](https://docs.last9.io)
- [Terraform Best Practices](https://www.terraform.io/docs/cloud/guides/recommended-practices/index.html)