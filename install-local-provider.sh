#!/bin/bash

# install-local-provider.sh - Install Last9 Terraform Provider for Local Development
# This script builds and installs the provider to the local Terraform plugins directory

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROVIDER_NAME="last9"
PROVIDER_VERSION="1.0.0"
PROVIDER_NAMESPACE="hashicorp.com/edu"

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

Build and install the Last9 Terraform Provider for local development

OPTIONS:
    -f, --force         Force reinstall even if provider already exists
    -v, --verbose       Enable verbose output
    -c, --clean         Clean build artifacts after installation
    -h, --help          Show this help message

DESCRIPTION:
    This script:
    1. Builds the terraform-provider-last9 binary
    2. Creates the appropriate plugin directory structure
    3. Installs the provider for local development use
    4. Validates the installation

PLUGIN DIRECTORY:
    The provider will be installed to:
    ~/.terraform.d/plugins/${PROVIDER_NAMESPACE}/${PROVIDER_NAME}/${PROVIDER_VERSION}/OS_ARCH/

USAGE AFTER INSTALLATION:
    In your Terraform configuration, use:

    terraform {
      required_providers {
        last9 = {
          source = "${PROVIDER_NAMESPACE}/${PROVIDER_NAME}"
        }
      }
    }

EOF
}

check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if go is installed
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed. Please install Go first."
        exit 1
    fi

    # Check go version
    local go_version=$(go version | cut -d' ' -f3)
    log_info "Found Go version: $go_version"

    # Check if we're in the right directory
    if [[ ! -f "go.mod" ]] || [[ ! -f "main.go" ]]; then
        log_error "This script must be run from the root of the terraform-provider repository"
        exit 1
    fi

    # Check go.mod contains correct module
    if ! grep -q "terraform-provider-last9" go.mod; then
        log_error "This doesn't appear to be the Last9 terraform provider repository"
        exit 1
    fi

    log_success "Prerequisites check passed"
}

get_platform_info() {
    local os_name=$(go env GOOS)
    local arch_name=$(go env GOARCH)

    echo "${os_name}_${arch_name}"
}

build_provider() {
    log_info "Building Last9 Terraform Provider..."
    cd "$SCRIPT_DIR"

    local binary_name="terraform-provider-${PROVIDER_NAME}"

    if [[ "$VERBOSE" == "true" ]]; then
        go build -o "$binary_name"
    else
        go build -o "$binary_name" > /dev/null 2>&1
    fi

    if [[ -f "$binary_name" ]]; then
        log_success "Provider binary built successfully: $binary_name"

        # Make it executable
        chmod +x "$binary_name"

        return 0
    else
        log_error "Failed to build provider binary"
        return 1
    fi
}

create_plugin_directory() {
    log_info "Creating plugin directory structure..."

    local platform=$(get_platform_info)
    local plugin_dir="$HOME/.terraform.d/plugins/${PROVIDER_NAMESPACE}/${PROVIDER_NAME}/${PROVIDER_VERSION}/${platform}"

    if [[ -d "$plugin_dir" ]]; then
        if [[ "$FORCE_INSTALL" == "true" ]]; then
            log_warning "Plugin directory already exists, removing due to --force"
            rm -rf "$plugin_dir"
        else
            log_warning "Plugin directory already exists: $plugin_dir"
            log_warning "Use --force to overwrite, or remove manually"
            return 1
        fi
    fi

    mkdir -p "$plugin_dir"
    log_success "Created plugin directory: $plugin_dir"

    echo "$plugin_dir"
}

install_provider() {
    log_info "Installing provider to plugin directory..."

    local binary_name="terraform-provider-${PROVIDER_NAME}"
    local plugin_dir="$1"

    if [[ ! -f "$binary_name" ]]; then
        log_error "Provider binary not found: $binary_name"
        return 1
    fi

    cp "$binary_name" "$plugin_dir/"

    if [[ -f "$plugin_dir/$binary_name" ]]; then
        log_success "Provider installed successfully to: $plugin_dir/$binary_name"
        return 0
    else
        log_error "Failed to copy provider binary to plugin directory"
        return 1
    fi
}

validate_installation() {
    log_info "Validating installation..."

    local platform=$(get_platform_info)
    local plugin_path="$HOME/.terraform.d/plugins/${PROVIDER_NAMESPACE}/${PROVIDER_NAME}/${PROVIDER_VERSION}/${platform}/terraform-provider-${PROVIDER_NAME}"

    if [[ -f "$plugin_path" ]]; then
        log_success "Provider binary found at: $plugin_path"

        # Check if binary is executable
        if [[ -x "$plugin_path" ]]; then
            log_success "Provider binary is executable"
        else
            log_warning "Provider binary is not executable, fixing..."
            chmod +x "$plugin_path"
        fi

        # Get file info
        local file_size=$(ls -lh "$plugin_path" | awk '{print $5}')
        log_info "Provider binary size: $file_size"

        return 0
    else
        log_error "Provider binary not found at expected location"
        return 1
    fi
}

cleanup_build_artifacts() {
    log_info "Cleaning up build artifacts..."

    local binary_name="terraform-provider-${PROVIDER_NAME}"

    if [[ -f "$binary_name" ]]; then
        rm "$binary_name"
        log_success "Removed build artifact: $binary_name"
    fi
}

show_usage_instructions() {
    log_info "Installation completed successfully!"
    echo
    echo -e "${GREEN}NEXT STEPS:${NC}"
    echo
    echo "1. In your Terraform configuration, use:"
    echo
    echo "   terraform {"
    echo "     required_providers {"
    echo "       last9 = {"
    echo "         source = \"${PROVIDER_NAMESPACE}/${PROVIDER_NAME}\""
    echo "       }"
    echo "     }"
    echo "   }"
    echo
    echo "2. To test the installation:"
    echo "   cd examples/minimal-test"
    echo "   cp terraform.tfvars.example terraform.tfvars"
    echo "   # Edit terraform.tfvars with your credentials"
    echo "   terraform init"
    echo "   terraform plan"
    echo
    echo "3. For comprehensive integration testing:"
    echo "   cd examples/integration-test"
    echo "   ./run-integration-test.sh --help"
    echo
}

main() {
    local FORCE_INSTALL=false
    local VERBOSE=false
    local CLEAN_ARTIFACTS=false

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -f|--force)
                FORCE_INSTALL=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -c|--clean)
                CLEAN_ARTIFACTS=true
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

    log_info "Starting Last9 Terraform Provider Local Installation"
    log_info "Configuration: FORCE=$FORCE_INSTALL, VERBOSE=$VERBOSE, CLEAN=$CLEAN_ARTIFACTS"

    # Run installation steps
    check_prerequisites

    if build_provider; then
        if plugin_dir=$(create_plugin_directory); then
            if install_provider "$plugin_dir"; then
                if validate_installation; then
                    if [[ "$CLEAN_ARTIFACTS" == "true" ]]; then
                        cleanup_build_artifacts
                    fi
                    show_usage_instructions
                else
                    log_error "Installation validation failed"
                    exit 1
                fi
            else
                log_error "Provider installation failed"
                exit 1
            fi
        else
            log_error "Plugin directory creation failed"
            exit 1
        fi
    else
        log_error "Provider build failed"
        exit 1
    fi

    log_success "Local provider installation completed successfully!"
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi