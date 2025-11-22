#!/bin/bash
# Quick smoke test - just verify files compile

cd /Users/dgarcia/go/src/github.com/infobloxopen/apx

echo "Quick smoke test: Verifying E2E code compiles..."
echo ""

# Build E2E tests without running them
go test -c ./tests/e2e -o /tmp/e2e-test-binary

if [ $? -eq 0 ]; then
    echo "✅ E2E code compiles successfully!"
    echo ""
    echo "Files verified:"
    echo "  - tests/e2e/main_test.go"
    echo "  - tests/e2e/k3d/*.go"
    echo "  - tests/e2e/gitea/*.go"
    echo "  - tests/e2e/testhelpers/*.go"
    echo ""
    echo "Ready to run full E2E tests with: bash test-phase3.sh"
    rm -f /tmp/e2e-test-binary
    exit 0
else
    echo "❌ Compilation failed!"
    echo ""
    echo "Fix errors above before running E2E tests."
    exit 1
fi
