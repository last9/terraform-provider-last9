#!/bin/bash
# Test runner script

set -e

cd "$(dirname "$0")"

echo "=========================================="
echo "Running Terraform Provider Tests"
echo "=========================================="
echo ""

echo "1. Checking Go installation..."
if ! command -v go &> /dev/null; then
    echo "ERROR: Go is not installed"
    exit 1
fi
go version
echo ""

echo "2. Downloading dependencies..."
go mod download 2>&1 | tail -5 || echo "Dependencies already downloaded"
echo ""

echo "3. Running unit tests..."
echo "----------------------------------------"
go test ./internal/provider -run '^Test(Provider|Resource)' -v 2>&1 || {
    echo ""
    echo "Unit tests completed (some may have failed - check output above)"
}
echo ""

echo "4. Running all tests..."
echo "----------------------------------------"
go test ./internal/provider -v 2>&1 | tail -30 || {
    echo ""
    echo "Tests completed (check output above for results)"
}
echo ""

echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo "Note: Acceptance tests require TF_ACC=1 and API credentials"
echo "Run: TF_ACC=1 go test ./internal/provider -v -run '^TestAcc'"

