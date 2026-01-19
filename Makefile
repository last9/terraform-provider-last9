# Makefile for Terraform Provider

.PHONY: build install test clean fmt vet lint install-local integration-test integration-test-minimal validate-integration

# Build the provider
build:
	go build -o terraform-provider-last9

# Install the provider locally
install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/last9/last9/1.0.0/$(shell go env GOOS)_$(shell go env GOARCH)
	cp terraform-provider-last9 ~/.terraform.d/plugins/registry.terraform.io/last9/last9/1.0.0/$(shell go env GOOS)_$(shell go env GOARCH)/

# Run unit tests only (no API calls)
test:
	go test ./internal/provider -run '^Test(Resource|Provider)' -v

# Run all tests including unit tests
test-all:
	go test ./internal/provider -v

# Run acceptance tests (requires API credentials)
testacc:
	TF_ACC=1 go test ./internal/provider -v -run '^TestAcc'

# Run specific test
test-specific:
	@echo "Usage: make test-specific TEST=TestAccDashboard_basic"
	@if [ -z "$(TEST)" ]; then echo "Error: TEST variable required"; exit 1; fi
	TF_ACC=1 go test ./internal/provider -v -run '^$(TEST)'

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -f terraform-provider-last9
	go clean

# Run all checks
check: fmt vet test

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o dist/terraform-provider-last9_linux_amd64
	GOOS=darwin GOARCH=amd64 go build -o dist/terraform-provider-last9_darwin_amd64
	GOOS=darwin GOARCH=arm64 go build -o dist/terraform-provider-last9_darwin_arm64
	GOOS=windows GOARCH=amd64 go build -o dist/terraform-provider-last9_windows_amd64.exe

# Install provider for local development (uses custom namespace)
install-local:
	./install-local-provider.sh

# Install provider for local development with verbose output
install-local-verbose:
	./install-local-provider.sh --verbose

# Force reinstall provider for local development
install-local-force:
	./install-local-provider.sh --force --verbose

# Run comprehensive integration tests (requires environment variables)
integration-test: install-local
	@echo "Running comprehensive integration tests..."
	@echo "Required environment variables:"
	@echo "  LAST9_ORG, LAST9_REFRESH_TOKEN, LAST9_CLUSTER_ID, LAST9_ENTITY_NAME"
	@if [ -z "$(LAST9_ORG)" ]; then echo "Error: LAST9_ORG environment variable required"; exit 1; fi
	@if [ -z "$(LAST9_REFRESH_TOKEN)" ]; then echo "Error: LAST9_REFRESH_TOKEN environment variable required"; exit 1; fi
	@if [ -z "$(LAST9_CLUSTER_ID)" ]; then echo "Error: LAST9_CLUSTER_ID environment variable required"; exit 1; fi
	@if [ -z "$(LAST9_ENTITY_NAME)" ]; then echo "Error: LAST9_ENTITY_NAME environment variable required"; exit 1; fi
	cd examples/integration-test && ./run-integration-test.sh --verbose

# Run integration tests with dry-run (plan only, no apply)
integration-test-dry: install-local
	@echo "Running integration test dry-run (plan only)..."
	@if [ -z "$(LAST9_ORG)" ]; then echo "Error: LAST9_ORG environment variable required"; exit 1; fi
	@if [ -z "$(LAST9_REFRESH_TOKEN)" ]; then echo "Error: LAST9_REFRESH_TOKEN environment variable required"; exit 1; fi
	@if [ -z "$(LAST9_CLUSTER_ID)" ]; then echo "Error: LAST9_CLUSTER_ID environment variable required"; exit 1; fi
	@if [ -z "$(LAST9_ENTITY_NAME)" ]; then echo "Error: LAST9_ENTITY_NAME environment variable required"; exit 1; fi
	cd examples/integration-test && ./run-integration-test.sh --dry-run --verbose

# Run minimal integration test (quick smoke test)
integration-test-minimal: install-local
	@echo "Running minimal integration test..."
	@if [ -z "$(LAST9_ORG)" ]; then echo "Error: LAST9_ORG environment variable required"; exit 1; fi
	@if [ -z "$(LAST9_REFRESH_TOKEN)" ]; then echo "Error: LAST9_REFRESH_TOKEN environment variable required"; exit 1; fi
	@if [ -z "$(LAST9_ENTITY_NAME)" ]; then echo "Error: LAST9_ENTITY_NAME environment variable required"; exit 1; fi
	@echo "Creating minimal test configuration..."
	@cd examples/minimal-test && \
	cp terraform.tfvars.example terraform.tfvars && \
	sed -i '' 's/your-org-name/$(LAST9_ORG)/g' terraform.tfvars && \
	sed -i '' 's/your-refresh-token/$(LAST9_REFRESH_TOKEN)/g' terraform.tfvars && \
	sed -i '' 's/your-entity-name/$(LAST9_ENTITY_NAME)/g' terraform.tfvars && \
	terraform init && \
	terraform plan && \
	terraform apply -auto-approve && \
	terraform output && \
	terraform destroy -auto-approve && \
	rm terraform.tfvars .terraform.lock.hcl && \
	rm -rf .terraform/

# Validate integration test results (run after integration-test with --no-cleanup)
validate-integration:
	@echo "Validating integration test results..."
	cd examples/integration-test && ./validate-results.sh --verbose

# Integration test help
integration-help:
	@echo "Integration Test Targets:"
	@echo ""
	@echo "  make install-local              Install provider for local development"
	@echo "  make install-local-force        Force reinstall local provider"
	@echo "  make integration-test           Run comprehensive integration tests"
	@echo "  make integration-test-dry       Run integration test plan only"
	@echo "  make integration-test-minimal   Run minimal smoke test"
	@echo "  make validate-integration       Validate integration test results"
	@echo ""
	@echo "Required Environment Variables:"
	@echo "  LAST9_ORG                       Your Last9 organization slug"
	@echo "  LAST9_REFRESH_TOKEN             Your Last9 refresh token"
	@echo "  LAST9_CLUSTER_ID                Your Last9 cluster ID (full tests only)"
	@echo "  LAST9_ENTITY_NAME               Entity name for testing"
	@echo ""
	@echo "Example Usage:"
	@echo "  export LAST9_ORG='my-org'"
	@echo "  export LAST9_REFRESH_TOKEN='token-here'"
	@echo "  export LAST9_CLUSTER_ID='cluster-123'"
	@echo "  export LAST9_ENTITY_NAME='production-api'"
	@echo "  make integration-test-dry"
	@echo ""

