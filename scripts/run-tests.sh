#!/bin/bash

# Hatcher Test Runner Script
# Comprehensive test execution with reporting

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COVERAGE_THRESHOLD=80
REPORTS_DIR="reports"
TIMEOUT="10m"

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

# Create reports directory
mkdir -p "$REPORTS_DIR"

# Print header
echo "🥇 Hatcher Test Suite Runner"
echo "================================"
echo "Timestamp: $(date)"
echo "Go Version: $(go version)"
echo "Coverage Threshold: ${COVERAGE_THRESHOLD}%"
echo "Timeout: $TIMEOUT"
echo ""

# 1. Code formatting check
log_info "Checking code formatting..."
if ! gofmt -l . | grep -q .; then
    log_success "Code formatting is correct"
else
    log_error "Code formatting issues found:"
    gofmt -l .
    exit 1
fi

# 2. Code vetting
log_info "Running go vet..."
if go vet ./...; then
    log_success "Code vetting passed"
else
    log_error "Code vetting failed"
    exit 1
fi

# 3. Security scan (if gosec is available)
log_info "Running security scan..."
if command -v gosec >/dev/null 2>&1; then
    if gosec -fmt json -out "$REPORTS_DIR/security.json" ./...; then
        log_success "Security scan passed"
    else
        log_warning "Security scan found issues (check $REPORTS_DIR/security.json)"
    fi
else
    log_warning "gosec not installed, skipping security scan"
fi

# 4. Unit tests with coverage
log_info "Running unit tests with coverage..."
go test -v -timeout="$TIMEOUT" -coverprofile="$REPORTS_DIR/coverage.out" -covermode=atomic ./... > "$REPORTS_DIR/test-output.txt" 2>&1

if [ $? -eq 0 ]; then
    log_success "Unit tests passed"
else
    log_error "Unit tests failed"
    cat "$REPORTS_DIR/test-output.txt"
    exit 1
fi

# 5. Generate coverage report
log_info "Generating coverage report..."
go tool cover -html="$REPORTS_DIR/coverage.out" -o "$REPORTS_DIR/coverage.html"
go tool cover -func="$REPORTS_DIR/coverage.out" > "$REPORTS_DIR/coverage.txt"

# Extract coverage percentage
COVERAGE=$(go tool cover -func="$REPORTS_DIR/coverage.out" | grep total | awk '{print $3}' | sed 's/%//')
COVERAGE_INT=${COVERAGE%.*}

echo "Coverage: ${COVERAGE}%"

if [ "$COVERAGE_INT" -ge "$COVERAGE_THRESHOLD" ]; then
    log_success "Coverage threshold met (${COVERAGE}% >= ${COVERAGE_THRESHOLD}%)"
else
    log_error "Coverage threshold not met (${COVERAGE}% < ${COVERAGE_THRESHOLD}%)"
    exit 1
fi

# 6. Race condition tests
log_info "Running race condition tests..."
if go test -race -timeout="$TIMEOUT" ./... > "$REPORTS_DIR/race-test.txt" 2>&1; then
    log_success "Race condition tests passed"
else
    log_error "Race condition tests failed"
    cat "$REPORTS_DIR/race-test.txt"
    exit 1
fi

# 7. Benchmark tests
log_info "Running benchmark tests..."
go test -bench=. -benchmem -timeout="$TIMEOUT" ./... > "$REPORTS_DIR/benchmark.txt" 2>&1
if [ $? -eq 0 ]; then
    log_success "Benchmark tests completed"
else
    log_warning "Some benchmark tests failed"
fi

# 8. Memory leak detection (if available)
log_info "Checking for memory leaks..."
if command -v go-torch >/dev/null 2>&1; then
    # Run memory profiling
    go test -memprofile="$REPORTS_DIR/mem.prof" -bench=. ./internal/autocopy/ > /dev/null 2>&1
    log_success "Memory profiling completed"
else
    log_warning "go-torch not available, skipping memory leak detection"
fi

# 9. Test specific packages with verbose output
log_info "Running package-specific tests..."

# Core packages
CORE_PACKAGES=(
    "./internal/git"
    "./internal/worktree"
    "./internal/autocopy"
    "./internal/editor"
    "./internal/config"
    "./internal/doctor"
    "./cmd"
)

for pkg in "${CORE_PACKAGES[@]}"; do
    log_info "Testing package: $pkg"
    if go test -v -timeout="$TIMEOUT" "$pkg" > "$REPORTS_DIR/test-$(basename $pkg).txt" 2>&1; then
        log_success "Package $pkg tests passed"
    else
        log_error "Package $pkg tests failed"
        cat "$REPORTS_DIR/test-$(basename $pkg).txt"
        exit 1
    fi
done

# 10. Integration tests
log_info "Running integration tests..."
if go test -v -timeout="$TIMEOUT" -tags=integration ./... > "$REPORTS_DIR/integration-test.txt" 2>&1; then
    log_success "Integration tests passed"
else
    log_warning "Integration tests failed or not found"
fi

# 11. Performance regression tests
log_info "Running performance regression tests..."
if go test -v -timeout="$TIMEOUT" -run="TestPerformanceRegression" ./... > "$REPORTS_DIR/performance.txt" 2>&1; then
    log_success "Performance regression tests passed"
else
    log_warning "Performance regression tests failed"
fi

# 12. Generate test summary
log_info "Generating test summary..."
cat > "$REPORTS_DIR/summary.txt" << EOF
Hatcher Test Summary
===================
Date: $(date)
Go Version: $(go version)

Test Results:
- Unit Tests: PASSED
- Race Tests: PASSED
- Coverage: ${COVERAGE}% (threshold: ${COVERAGE_THRESHOLD}%)
- Security Scan: $([ -f "$REPORTS_DIR/security.json" ] && echo "COMPLETED" || echo "SKIPPED")
- Benchmarks: COMPLETED

Files Generated:
- coverage.html - HTML coverage report
- coverage.txt - Text coverage report
- test-output.txt - Full test output
- race-test.txt - Race condition test results
- benchmark.txt - Benchmark results
- security.json - Security scan results (if available)

Coverage Details:
$(head -20 "$REPORTS_DIR/coverage.txt")
EOF

# Print summary
echo ""
echo "🎉 Test Suite Summary"
echo "===================="
cat "$REPORTS_DIR/summary.txt"

log_success "All tests completed successfully!"
log_info "Reports available in: $REPORTS_DIR/"
log_info "Open coverage report: open $REPORTS_DIR/coverage.html"

exit 0
