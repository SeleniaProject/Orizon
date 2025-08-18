#!/bin/bash
# Self-hosting Stage 0: Build Orizon using Go implementation
# This script builds the Orizon compiler and tools using the current Go implementation

set -e

echo "=== Orizon Self-hosting Stage 0 ==="
echo "Building Orizon using Go implementation..."

# Set up environment
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$ROOT_DIR/build/stage0"
ARTIFACTS_DIR="$ROOT_DIR/artifacts/selfhost"

echo "Root directory: $ROOT_DIR"
echo "Build directory: $BUILD_DIR"
echo "Artifacts directory: $ARTIFACTS_DIR"

# Clean and create build directories
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"
mkdir -p "$ARTIFACTS_DIR"

cd "$ROOT_DIR"

# Build all Orizon tools using Go
echo ""
echo "=== Building Orizon Tools ==="

tools=(
    "orizon"
    "orizon-compiler" 
    "orizon-bootstrap"
    "orizon-fmt"
    "orizon-lsp"
    "orizon-test"
    "orizon-pkg"
)

for tool in "${tools[@]}"; do
    echo "Building $tool..."
    go build -o "$BUILD_DIR/$tool" "./cmd/$tool"
    if [ $? -eq 0 ]; then
        echo "✓ $tool built successfully"
    else
        echo "✗ Failed to build $tool"
        exit 1
    fi
done

# Run basic validation tests
echo ""
echo "=== Validating Built Tools ==="

# Test orizon-fmt
echo "Testing orizon-fmt..."
echo "fn main() { let x = 1 + 2; }" | "$BUILD_DIR/orizon-fmt" -stdin
if [ $? -eq 0 ]; then
    echo "✓ orizon-fmt validation passed"
else
    echo "✗ orizon-fmt validation failed"
    exit 1
fi

# Test orizon-test
echo "Testing orizon-test..."
"$BUILD_DIR/orizon-test" --help > /dev/null
if [ $? -eq 0 ]; then
    echo "✓ orizon-test validation passed"
else
    echo "✗ orizon-test validation failed"
    exit 1
fi

# Test orizon-pkg
echo "Testing orizon-pkg..."
"$BUILD_DIR/orizon-pkg" help > /dev/null
if [ $? -eq 0 ]; then
    echo "✓ orizon-pkg validation passed"
else
    echo "✗ orizon-pkg validation failed"
    exit 1
fi

# Generate build metadata
echo ""
echo "=== Generating Build Metadata ==="

BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
GO_VERSION=$(go version)

cat > "$ARTIFACTS_DIR/stage0-metadata.json" << EOF
{
  "stage": 0,
  "description": "Orizon built using Go implementation",
  "build_time": "$BUILD_TIME",
  "git_commit": "$GIT_COMMIT",
  "go_version": "$GO_VERSION",
  "built_tools": [
$(printf '    "%s"' "${tools[0]}")
$(printf ',\n    "%s"' "${tools[@]:1}")
  ],
  "build_directory": "$BUILD_DIR",
  "artifacts_directory": "$ARTIFACTS_DIR"
}
EOF

# Copy tools to artifacts directory
echo "Copying tools to artifacts directory..."
cp "$BUILD_DIR"/* "$ARTIFACTS_DIR/"

# Run comprehensive tests
echo ""
echo "=== Running Comprehensive Tests ==="

# Run Go tests
echo "Running Go test suite..."
go test ./...
if [ $? -eq 0 ]; then
    echo "✓ Go test suite passed"
else
    echo "✗ Go test suite failed"
    exit 1
fi

# Generate test report
"$BUILD_DIR/orizon-test" -packages "./..." -json > "$ARTIFACTS_DIR/stage0-test-results.json" 2>/dev/null || echo "Test results generation completed"

# Create summary
echo ""
echo "=== Stage 0 Build Summary ==="
echo "Build completed successfully!"
echo "Tools built: ${#tools[@]}"
echo "Build artifacts: $ARTIFACTS_DIR"
echo "Build time: $BUILD_TIME"
echo "Git commit: $GIT_COMMIT"

ls -la "$ARTIFACTS_DIR"

echo ""
echo "Stage 0 completed. Ready for Stage 1 (self-compilation)."
