#!/bin/bash
# Test Phase 3: Verify E2E infrastructure and run tests

set -e  # Exit on error

cd /Users/dgarcia/go/src/github.com/infobloxopen/apx

echo "=========================================="
echo "Phase 3 Test Suite"
echo "=========================================="
echo ""

# Step 1: Check if Docker is running
echo "1. Checking Docker..."
if ! docker info > /dev/null 2>&1; then
    echo "❌ ERROR: Docker is not running!"
    echo "   Please start Docker Desktop and try again."
    exit 1
fi
echo "✓ Docker is running"
echo ""

# Step 2: Check if k3d is installed
echo "2. Checking k3d installation..."
if ! command -v k3d &> /dev/null; then
    echo "⚠️  k3d is not installed. Installing..."
    make install-e2e-deps
    if ! command -v k3d &> /dev/null; then
        echo "❌ ERROR: Failed to install k3d"
        exit 1
    fi
fi
k3d version
echo "✓ k3d is installed"
echo ""

# Step 3: Check if kubectl is installed
echo "3. Checking kubectl installation..."
if ! command -v kubectl &> /dev/null; then
    echo "⚠️  kubectl is not installed. Installing..."
    make install-e2e-deps
    if ! command -v kubectl &> /dev/null; then
        echo "❌ ERROR: Failed to install kubectl"
        exit 1
    fi
fi
kubectl version --client --short 2>/dev/null || kubectl version --client
echo "✓ kubectl is installed"
echo ""

# Step 4: Verify test files exist
echo "4. Verifying Phase 3 files..."
files=(
    "testdata/script/e2e/e2e_basic_setup.txt"
    "testdata/script/e2e/e2e_complete_workflow.txt"
    "tests/e2e/README.md"
    "tests/e2e/main_test.go"
    "tests/e2e/k3d/cluster.go"
    "tests/e2e/gitea/lifecycle.go"
)

for file in "${files[@]}"; do
    if [ ! -f "$file" ]; then
        echo "❌ ERROR: Missing file: $file"
        exit 1
    fi
done
echo "✓ All Phase 3 files present"
echo ""

# Step 5: Build the apx binary (needed for tests)
echo "5. Building apx binary..."
make build
if [ ! -f "bin/apx" ]; then
    echo "❌ ERROR: Failed to build apx binary"
    exit 1
fi
echo "✓ apx binary built"
echo ""

# Step 6: Add apx to PATH for tests
export PATH="$PWD/bin:$PATH"

# Step 7: Run E2E tests
echo "6. Running E2E tests..."
echo "   This will:"
echo "   - Create a k3d cluster (apx-e2e-XXXXXXXX)"
echo "   - Deploy Gitea to the cluster"
echo "   - Run testscript scenarios"
echo "   - Cleanup resources automatically"
echo ""
echo "   Expected duration: 2-5 minutes"
echo ""

# Set timeout to 15 minutes
E2E_ENABLED=1 go test -v -timeout 15m ./tests/e2e/... 2>&1 | tee test-output.log

# Check test result
if [ ${PIPESTATUS[0]} -eq 0 ]; then
    echo ""
    echo "=========================================="
    echo "✅ Phase 3 Tests PASSED!"
    echo "=========================================="
    echo ""
    echo "Summary:"
    echo "  - E2E infrastructure validated"
    echo "  - Basic setup test passed"
    echo "  - Complete workflow test passed"
    echo ""
    echo "Next steps:"
    echo "  1. Commit Phase 3: bash commit-phase3.sh"
    echo "  2. Review test output: cat test-output.log"
    echo "  3. Proceed to Phase 4 (User Story 2)"
    echo ""
else
    echo ""
    echo "=========================================="
    echo "❌ Phase 3 Tests FAILED!"
    echo "=========================================="
    echo ""
    echo "Troubleshooting:"
    echo "  1. Check test output: cat test-output.log"
    echo "  2. Verify Docker has enough resources (2GB RAM)"
    echo "  3. Check cleanup: make clean-e2e"
    echo "  4. Re-run: bash test-phase3.sh"
    echo ""
    exit 1
fi
