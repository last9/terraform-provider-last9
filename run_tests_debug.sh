#!/bin/bash
# Test runner with better error output

set -e

cd "$(dirname "$0")"

echo "Running tests..."
echo "================"

# Try to run tests and capture output
go test ./internal/provider -run '^Test(Provider|Resource)' -v 2>&1 | tee test_output.txt || {
    echo ""
    echo "Tests failed. Checking for compilation errors..."
    go build ./internal/provider 2>&1 || true
    exit 1
}

echo ""
echo "Tests completed successfully!"

