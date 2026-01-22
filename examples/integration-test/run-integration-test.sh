#!/bin/bash

# run-integration-test.sh - Automated Integration Test Runner for Last9 Terraform Provider
# This script sets up the local provider, runs integration tests, and validates results

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROVIDER_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
TEST_DIR="${SCRIPT_DIR}"
TF_DATA_DIR="${TEST_DIR}/.terraform"

# Default values
DRY_RUN=false
CLEANUP=true
VERBOSE=false
USE_EXISTING_VARS=false

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Run integration tests for the Last9 Terraform Provider

OPTIONS:
    -d, --dry-run       Run terraform plan only, don't apply
    -n, --no-cleanup    Don't destroy resources after test
    -v, --verbose       Enable verbose output
    -e, --use-existing  Use existing terraform.tfvars file
    -h, --help          Show this help message

ENVIRONMENT VARIABLES:
    LAST9_ORG                Required: Your Last9 organization
    LAST9_REFRESH_TOKEN     Required: Your Last9 refresh token
    LAST9_CLUSTER_ID        Required: Your Last9 cluster ID
    LAST9_ENTITY_NAME       Required: Entity name to monitor

EXAMPLE:
    export LAST9_ORG="my-org"
    export LAST9_REFRESH_TOKEN="your-token"
    export LAST9_CLUSTER_ID="cluster-123"
    export LAST9_ENTITY_NAME="production-api"
    $0 --verbose

EOF
}

check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if terraform is installed
    if ! command -v terraform &> /dev/null; then
        log_error "Terraform is not installed. Please install Terraform first."
        exit 1
    fi

    # Check if go is installed
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed. Please install Go first."
        exit 1
    fi

    # Check terraform version
    local tf_version=$(terraform version -json | jq -r '.terraform_version')
    log_info "Found Terraform version: $tf_version"

    # Check required environment variables
    if [[ -z "${LAST9_ORG:-}" ]]; then
        log_error "LAST9_ORG environment variable is required"
        exit 1
    fi

    if [[ -z "${LAST9_REFRESH_TOKEN:-}" ]]; then
        log_error "LAST9_REFRESH_TOKEN environment variable is required"
        exit 1
    fi

    if [[ -z "${LAST9_CLUSTER_ID:-}" ]]; then
        log_error "LAST9_CLUSTER_ID environment variable is required"
        exit 1
    fi

    if [[ -z "${LAST9_ENTITY_NAME:-}" ]]; then
        log_error "LAST9_ENTITY_NAME environment variable is required"
        exit 1
    fi

    log_success "Prerequisites check passed"
}

build_and_install_provider() {
    log_info "Building and installing Last9 provider locally..."

    cd "$PROVIDER_ROOT"

    # Build the provider
    if [[ "$VERBOSE" == "true" ]]; then
        go build -o terraform-provider-last9
    else
        go build -o terraform-provider-last9 > /dev/null 2>&1
    fi

    # Install to local plugins directory
    local os_arch=$(go env GOOS)_$(go env GOARCH)
    local plugin_dir="$HOME/.terraform.d/plugins/hashicorp.com/edu/last9/1.0.0/$os_arch"

    mkdir -p "$plugin_dir"
    cp terraform-provider-last9 "$plugin_dir/"

    log_success "Provider built and installed to $plugin_dir"
}

setup_terraform_config() {
    log_info "Setting up Terraform configuration..."

    cd "$TEST_DIR"

    # Use local provider configuration for testing
    if [[ -f "main.tf" ]]; then
        mv main.tf main.tf.published
    fi

    if [[ -f "versions.tf" ]]; then
        cp versions.tf main.tf
    else
        log_error "versions.tf not found for local provider configuration"
        exit 1
    fi

    # Create terraform.tfvars if not using existing
    if [[ "$USE_EXISTING_VARS" == "false" || ! -f "terraform.tfvars" ]]; then
        log_info "Creating terraform.tfvars from environment variables..."

        cat > terraform.tfvars << EOF
# Generated terraform.tfvars for integration test
last9_org = "${LAST9_ORG}"
last9_refresh_token = "${LAST9_REFRESH_TOKEN}"
cluster_id = "${LAST9_CLUSTER_ID}"
entity_name = "${LAST9_ENTITY_NAME}"

# Integration test configuration
environment = "integration-test-$(date +%s)"
region = "us-west-2"
service_name = "integration-test-service"
team_name = "integration-test"

# Alert thresholds for testing
error_rate_threshold = 10.0
availability_threshold = 99.0
response_time_threshold = 1000

# Enable all features for comprehensive testing
enable_log_forwarding = true
enable_debug_log_dropping = true
enable_advanced_alerts = true

# Test-specific tags
additional_tags = ["integration-test", "automated", "$(date +%Y%m%d)"]
EOF
        log_success "Created terraform.tfvars"
    else
        log_info "Using existing terraform.tfvars"
    fi
}

run_terraform_init() {
    log_info "Initializing Terraform..."
    cd "$TEST_DIR"

    if [[ "$VERBOSE" == "true" ]]; then
        terraform init
    else
        terraform init > /dev/null
    fi

    log_success "Terraform initialized"
}

run_terraform_validate() {
    log_info "Validating Terraform configuration..."
    cd "$TEST_DIR"

    if terraform validate; then
        log_success "Terraform configuration is valid"
    else
        log_error "Terraform configuration validation failed"
        exit 1
    fi
}

run_terraform_plan() {
    log_info "Running Terraform plan..."
    cd "$TEST_DIR"

    local plan_file="integration-test.tfplan"

    if terraform plan -out="$plan_file"; then
        log_success "Terraform plan completed successfully"

        # Show plan summary
        log_info "Plan summary:"
        terraform show -no-color "$plan_file" | grep -E "(Plan:|No changes)" || true

        return 0
    else
        log_error "Terraform plan failed"
        return 1
    fi
}

run_terraform_apply() {
    log_info "Applying Terraform configuration..."
    cd "$TEST_DIR"

    local plan_file="integration-test.tfplan"

    if [[ -f "$plan_file" ]]; then
        if terraform apply "$plan_file"; then
            log_success "Terraform apply completed successfully"
            return 0
        else
            log_error "Terraform apply failed"
            return 1
        fi
    else
        log_error "Plan file not found. Run plan first."
        return 1
    fi
}

validate_resources() {
    log_info "Validating created resources..."
    cd "$TEST_DIR"

    # Get outputs
    local dashboard_id=$(terraform output -raw dashboard_id 2>/dev/null || echo "")
    local alert_ids=$(terraform output -json alert_ids 2>/dev/null || echo "{}")
    local policy_id=$(terraform output -raw policy_id 2>/dev/null || echo "")

    local validation_passed=true

    # Validate dashboard
    if [[ -n "$dashboard_id" ]]; then
        log_success "Dashboard created with ID: $dashboard_id"
    else
        log_error "Dashboard ID not found in outputs"
        validation_passed=false
    fi

    # Validate alerts
    local alert_count=$(echo "$alert_ids" | jq -r '. | length' 2>/dev/null || echo "0")
    if [[ "$alert_count" -ge 3 ]]; then
        log_success "Created $alert_count alerts as expected"
    else
        log_error "Expected 3 alerts, found $alert_count"
        validation_passed=false
    fi

    # Validate policy
    if [[ -n "$policy_id" ]]; then
        log_success "Policy created with ID: $policy_id"
    else
        log_error "Policy ID not found in outputs"
        validation_passed=false
    fi

    # Show integration test summary
    log_info "Integration test summary:"
    terraform output integration_test_summary 2>/dev/null || log_warning "Integration test summary not available"

    if [[ "$validation_passed" == "true" ]]; then
        log_success "Resource validation passed"
        return 0
    else
        log_error "Resource validation failed"
        return 1
    fi
}

run_terraform_destroy() {
    log_info "Destroying Terraform resources..."
    cd "$TEST_DIR"

    if terraform destroy -auto-approve; then
        log_success "Terraform destroy completed successfully"
        return 0
    else
        log_error "Terraform destroy failed"
        return 1
    fi
}

cleanup() {
    log_info "Cleaning up..."
    cd "$TEST_DIR"

    # Restore original configuration
    if [[ -f "main.tf.published" ]]; then
        mv main.tf.published main.tf
    fi

    # Remove generated files
    rm -f terraform.tfvars.generated
    rm -f integration-test.tfplan
    rm -rf .terraform/
    rm -f .terraform.lock.hcl

    log_success "Cleanup completed"
}

main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -n|--no-cleanup)
                CLEANUP=false
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -e|--use-existing)
                USE_EXISTING_VARS=true
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done

    log_info "Starting Last9 Terraform Provider Integration Test"
    log_info "Configuration: DRY_RUN=$DRY_RUN, CLEANUP=$CLEANUP, VERBOSE=$VERBOSE"

    # Set up trap for cleanup
    trap cleanup EXIT

    # Run the integration test
    check_prerequisites
    build_and_install_provider
    setup_terraform_config
    run_terraform_init
    run_terraform_validate

    if run_terraform_plan; then
        if [[ "$DRY_RUN" == "false" ]]; then
            if run_terraform_apply; then
                if validate_resources; then
                    log_success "Integration test PASSED!"

                    if [[ "$CLEANUP" == "true" ]]; then
                        run_terraform_destroy
                    else
                        log_warning "Resources left running (use --cleanup to destroy)"
                        log_info "To manually destroy: cd $TEST_DIR && terraform destroy"
                    fi
                else
                    log_error "Integration test FAILED - resource validation failed"
                    exit 1
                fi
            else
                log_error "Integration test FAILED - terraform apply failed"
                exit 1
            fi
        else
            log_info "Dry run completed - no resources were created"
        fi
    else
        log_error "Integration test FAILED - terraform plan failed"
        exit 1
    fi

    log_success "Integration test completed successfully!"
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi