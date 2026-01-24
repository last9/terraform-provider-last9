#!/bin/bash

# validate-results.sh - Validation script for Last9 Terraform Provider Integration Tests
# This script validates that all resources were created correctly and are functioning

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="${SCRIPT_DIR}"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Validate Last9 Terraform Provider Integration Test Results

OPTIONS:
    -v, --verbose       Enable verbose output
    -j, --json          Output results in JSON format
    -h, --help          Show this help message

DESCRIPTION:
    This script validates the results of integration tests by:
    - Checking Terraform state for created resources
    - Validating resource configurations
    - Verifying outputs are correct
    - Checking resource counts and types
    - Validating naming conventions

EOF
}

check_terraform_state() {
    log_info "Checking Terraform state..."
    cd "$TEST_DIR"

    if [[ ! -f "terraform.tfstate" ]]; then
        log_error "No terraform.tfstate found. Run integration test first."
        return 1
    fi

    # Check if state has resources
    local resource_count=$(terraform state list | wc -l)
    if [[ "$resource_count" -eq 0 ]]; then
        log_error "No resources found in Terraform state"
        return 1
    fi

    log_success "Found $resource_count resources in Terraform state"
    return 0
}

validate_dashboard_resources() {
    log_info "Validating dashboard resources..."
    cd "$TEST_DIR"

    local dashboard_count=0
    local validation_passed=true

    # Check for dashboard resources
    if terraform state list | grep -q "last9_dashboard"; then
        dashboard_count=$(terraform state list | grep -c "last9_dashboard" || echo "0")
        log_success "Found $dashboard_count dashboard(s)"

        # Validate dashboard outputs
        local dashboard_id=$(terraform output -raw dashboard_id 2>/dev/null || echo "")
        if [[ -n "$dashboard_id" ]]; then
            log_success "Dashboard ID output exists: $dashboard_id"
        else
            log_error "Dashboard ID output missing or empty"
            validation_passed=false
        fi
    else
        log_error "No dashboard resources found"
        validation_passed=false
    fi

    return $([[ "$validation_passed" == "true" ]] && echo 0 || echo 1)
}

validate_alert_resources() {
    log_info "Validating alert resources..."
    cd "$TEST_DIR"

    local alert_count=0
    local validation_passed=true

    # Check for alert resources
    if terraform state list | grep -q "last9_alert"; then
        alert_count=$(terraform state list | grep -c "last9_alert" || echo "0")
        log_success "Found $alert_count alert(s)"

        # Validate alert outputs
        local alert_ids=$(terraform output -json alert_ids 2>/dev/null || echo "{}")
        local alert_output_count=$(echo "$alert_ids" | jq -r '. | length' 2>/dev/null || echo "0")

        if [[ "$alert_output_count" -ge 1 ]]; then
            log_success "Alert IDs output exists with $alert_output_count alert(s)"
        else
            log_error "Alert IDs output missing or empty"
            validation_passed=false
        fi
    else
        log_error "No alert resources found"
        validation_passed=false
    fi

    return $([[ "$validation_passed" == "true" ]] && echo 0 || echo 1)
}

validate_macro_resources() {
    log_info "Validating macro resources..."
    cd "$TEST_DIR"

    local macro_count=0
    local validation_passed=true

    # Check for macro resources
    if terraform state list | grep -q "last9_macro"; then
        macro_count=$(terraform state list | grep -c "last9_macro" || echo "0")
        log_success "Found $macro_count macro(s)"

        # Validate macro cluster ID
        local cluster_id=$(terraform output -raw macro_cluster_id 2>/dev/null || echo "")
        if [[ -n "$cluster_id" ]]; then
            log_success "Macro cluster ID output exists: $cluster_id"
        else
            log_warning "Macro cluster ID output missing (may be expected)"
        fi
    else
        log_warning "No macro resources found (may be expected)"
    fi

    return 0  # Macros are optional in some configurations
}

validate_policy_resources() {
    log_info "Validating policy resources..."
    cd "$TEST_DIR"

    local policy_count=0
    local validation_passed=true

    # Check for policy resources
    if terraform state list | grep -q "last9_policy"; then
        policy_count=$(terraform state list | grep -c "last9_policy" || echo "0")
        log_success "Found $policy_count policy/policies"

        # Validate policy outputs
        local policy_id=$(terraform output -raw policy_id 2>/dev/null || echo "")
        if [[ -n "$policy_id" ]]; then
            log_success "Policy ID output exists: $policy_id"
        else
            log_error "Policy ID output missing or empty"
            validation_passed=false
        fi
    else
        log_warning "No policy resources found (may be expected)"
    fi

    return 0  # Policies might be optional
}

validate_log_management_resources() {
    log_info "Validating log management resources..."
    cd "$TEST_DIR"

    local drop_rule_count=0
    local forward_rule_count=0
    local validation_passed=true

    # Check for drop rules
    if terraform state list | grep -q "last9_drop_rule"; then
        drop_rule_count=$(terraform state list | grep -c "last9_drop_rule" || echo "0")
        log_success "Found $drop_rule_count drop rule(s)"
    else
        log_warning "No drop rules found (may be expected)"
    fi

    # Check for forward rules
    if terraform state list | grep -q "last9_forward_rule"; then
        forward_rule_count=$(terraform state list | grep -c "last9_forward_rule" || echo "0")
        log_success "Found $forward_rule_count forward rule(s)"
    else
        log_warning "No forward rules found (may be expected)"
    fi

    # Validate log management outputs if they exist
    local drop_rule_ids=$(terraform output -json drop_rule_ids 2>/dev/null || echo "{}")
    local forward_rule_ids=$(terraform output -json forward_rule_ids 2>/dev/null || echo "{}")

    if [[ "$drop_rule_ids" != "{}" || "$forward_rule_ids" != "{}" ]]; then
        log_success "Log management outputs exist"
    fi

    return 0
}

validate_data_sources() {
    log_info "Validating data sources..."
    cd "$TEST_DIR"

    local data_source_count=0
    local validation_passed=true

    # Check for data sources
    if terraform state list | grep -q "data\."; then
        data_source_count=$(terraform state list | grep -c "data\." || echo "0")
        log_success "Found $data_source_count data source(s)"

        # Validate entity data source output
        local entity_info=$(terraform output -json entity_info 2>/dev/null || echo "{}")
        if [[ "$entity_info" != "{}" ]]; then
            local entity_id=$(echo "$entity_info" | jq -r '.entity_id' 2>/dev/null || echo "")
            if [[ -n "$entity_id" && "$entity_id" != "null" ]]; then
                log_success "Entity data source working: $entity_id"
            else
                log_error "Entity data source not returning valid ID"
                validation_passed=false
            fi
        else
            log_warning "Entity info output not available"
        fi
    else
        log_warning "No data sources found"
    fi

    return $([[ "$validation_passed" == "true" ]] && echo 0 || echo 1)
}

validate_outputs() {
    log_info "Validating integration test outputs..."
    cd "$TEST_DIR"

    local validation_passed=true

    # Check for integration test summary
    local summary=$(terraform output -json integration_test_summary 2>/dev/null || echo "{}")
    if [[ "$summary" != "{}" ]]; then
        log_success "Integration test summary output exists"

        # Parse and validate summary
        local environment=$(echo "$summary" | jq -r '.environment' 2>/dev/null || echo "")
        local resources_created=$(echo "$summary" | jq -r '.resources_created' 2>/dev/null || echo "{}")

        if [[ -n "$environment" && "$environment" != "null" ]]; then
            log_success "Environment configured: $environment"
        fi

        if [[ "$resources_created" != "{}" ]]; then
            local dashboards=$(echo "$resources_created" | jq -r '.dashboards // 0' 2>/dev/null || echo "0")
            local alerts=$(echo "$resources_created" | jq -r '.alerts // 0' 2>/dev/null || echo "0")

            log_success "Resource summary: $dashboards dashboard(s), $alerts alert(s)"
        fi
    else
        log_error "Integration test summary output missing"
        validation_passed=false
    fi

    # Check validation URLs
    local urls=$(terraform output -json validation_urls 2>/dev/null || echo "{}")
    if [[ "$urls" != "{}" ]]; then
        log_success "Validation URLs output exists"
    else
        log_warning "Validation URLs output missing"
    fi

    return $([[ "$validation_passed" == "true" ]] && echo 0 || echo 1)
}

validate_naming_conventions() {
    log_info "Validating naming conventions..."
    cd "$TEST_DIR"

    local validation_passed=true
    local environment=$(terraform output -json integration_test_summary 2>/dev/null | jq -r '.environment' 2>/dev/null || echo "")

    # Check resource names contain environment prefix
    terraform state list | while read -r resource; do
        if [[ "$resource" =~ ^(last9_dashboard|last9_alert|last9_policy|last9_drop_rule|last9_forward_rule)\. ]]; then
            local resource_name=$(terraform state show "$resource" | grep -E '^\s+name\s*=' | head -1 | sed 's/.*= "//' | sed 's/".*//' || echo "")
            if [[ -n "$resource_name" && -n "$environment" ]]; then
                if [[ "$resource_name" =~ ^$environment- ]]; then
                    log_success "Resource $resource follows naming convention: $resource_name"
                else
                    log_warning "Resource $resource may not follow naming convention: $resource_name"
                fi
            fi
        fi
    done

    return 0
}

generate_validation_report() {
    log_info "Generating validation report..."
    cd "$TEST_DIR"

    local report_file="validation-report.json"
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Get basic counts
    local total_resources=$(terraform state list | wc -l)
    local dashboards=$(terraform state list | grep -c "last9_dashboard" || echo "0")
    local alerts=$(terraform state list | grep -c "last9_alert" || echo "0")
    local macros=$(terraform state list | grep -c "last9_macro" || echo "0")
    local policies=$(terraform state list | grep -c "last9_policy" || echo "0")
    local drop_rules=$(terraform state list | grep -c "last9_drop_rule" || echo "0")
    local forward_rules=$(terraform state list | grep -c "last9_forward_rule" || echo "0")
    local data_sources=$(terraform state list | grep -c "data\." || echo "0")

    # Get integration test summary
    local integration_summary=$(terraform output -json integration_test_summary 2>/dev/null || echo "{}")

    cat > "$report_file" << EOF
{
  "validation_report": {
    "timestamp": "$timestamp",
    "terraform_state": {
      "total_resources": $total_resources,
      "resource_types": {
        "dashboards": $dashboards,
        "alerts": $alerts,
        "macros": $macros,
        "policies": $policies,
        "drop_rules": $drop_rules,
        "forward_rules": $forward_rules,
        "data_sources": $data_sources
      }
    },
    "integration_test_summary": $integration_summary,
    "validation_status": "completed"
  }
}
EOF

    log_success "Validation report generated: $report_file"
    return 0
}

main() {
    local VERBOSE=false
    local JSON_OUTPUT=false

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -j|--json)
                JSON_OUTPUT=true
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

    log_info "Starting Last9 Terraform Provider Integration Test Validation"

    local overall_passed=true

    # Run all validations
    check_terraform_state || overall_passed=false
    validate_dashboard_resources || overall_passed=false
    validate_alert_resources || overall_passed=false
    validate_macro_resources || overall_passed=false
    validate_policy_resources || overall_passed=false
    validate_log_management_resources || overall_passed=false
    validate_data_sources || overall_passed=false
    validate_outputs || overall_passed=false
    validate_naming_conventions || overall_passed=false

    # Generate report
    generate_validation_report

    if [[ "$overall_passed" == "true" ]]; then
        log_success "All validations PASSED!"
        if [[ "$JSON_OUTPUT" == "true" ]]; then
            cat validation-report.json
        fi
        exit 0
    else
        log_error "Some validations FAILED!"
        exit 1
    fi
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi