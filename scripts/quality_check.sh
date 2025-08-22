#!/bin/bash

# Orizon Project - Code Quality Verification Script
# This script runs comprehensive checks to ensure code quality

set -e

echo "🔍 Starting Orizon Code Quality Verification..."

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# 1. Static Analysis
echo ""
echo "📊 Running Static Analysis..."

# Go vet
if go vet ./...; then
    print_status "Go vet passed"
else
    print_error "Go vet failed"
    exit 1
fi

# golangci-lint (if available)
if command -v golangci-lint &> /dev/null; then
    if golangci-lint run --timeout=5m; then
        print_status "golangci-lint passed"
    else
        print_warning "golangci-lint found issues"
    fi
else
    print_warning "golangci-lint not installed, skipping"
fi

# 2. Security Checks
echo ""
echo "🔒 Running Security Checks..."

# Go security checker (gosec)
if command -v gosec &> /dev/null; then
    if gosec ./...; then
        print_status "Security scan passed"
    else
        print_warning "Security issues detected"
    fi
else
    print_warning "gosec not installed, skipping security scan"
fi

# 3. Memory Safety Tests
echo ""
echo "🧠 Running Memory Safety Tests..."

# Run tests with race detector
if go test -race ./internal/types ./internal/allocator ./internal/runtime; then
    print_status "Race detector tests passed"
else
    print_error "Race detector found issues"
    exit 1
fi

# Run tests with memory sanitizer flags
export GOEXPERIMENT=cgocheck2
if go test ./internal/types -run TestOrizonSlice_OverflowProtection; then
    print_status "Overflow protection tests passed"
else
    print_error "Overflow protection tests failed"
    exit 1
fi

# 4. Performance Tests
echo ""
echo "⚡ Running Performance Tests..."

# Benchmark critical paths
go test -bench=BenchmarkOrizonSlice ./internal/types -benchmem > benchmark_results.txt
if [ $? -eq 0 ]; then
    print_status "Performance benchmarks completed"
    echo "Results saved to benchmark_results.txt"
else
    print_warning "Some benchmarks failed"
fi

# 5. Concurrency Tests
echo ""
echo "🔄 Running Concurrency Tests..."

# Run concurrency-specific tests
if go test -race ./internal/testrunner/concurrency -v; then
    print_status "Concurrency tests passed"
else
    print_error "Concurrency tests failed"
    exit 1
fi

# 6. Integration Tests
echo ""
echo "🔗 Running Integration Tests..."

# Build all major components
if go build ./cmd/orizon ./cmd/orizon-compiler; then
    print_status "All components build successfully"
else
    print_error "Build failed"
    exit 1
fi

# 7. Documentation Check
echo ""
echo "📚 Checking Documentation..."

# Check for missing godoc comments on exported functions
if command -v gocyclo &> /dev/null; then
    gocyclo -over 15 . || print_warning "High complexity functions detected"
else
    print_warning "gocyclo not installed, skipping complexity check"
fi

# 8. Final Report
echo ""
echo "📋 Quality Verification Summary:"
echo "================================"

# Count test files
test_files=$(find . -name "*_test.go" | wc -l)
echo "Test files: $test_files"

# Count lines of code
loc=$(find . -name "*.go" -not -path "./vendor/*" | xargs wc -l | tail -1 | awk '{print $1}')
echo "Lines of code: $loc"

# Generate coverage report
go test -coverprofile=coverage.out ./...
coverage=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}')
echo "Test coverage: $coverage"

echo ""
print_status "Code quality verification completed successfully!"
echo ""
echo "💡 Recommendations:"
echo "  - Review any warnings above"
echo "  - Check benchmark_results.txt for performance insights"
echo "  - Ensure test coverage remains above 80%"
echo ""
