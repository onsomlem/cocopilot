#!/usr/bin/env bash
# fresh-machine-test.sh — Validates the project works from a clean clone.
#
# Usage:
#   ./scripts/fresh-machine-test.sh [REPO_URL]
#
# If REPO_URL is omitted, uses the current directory's remote origin.
# Creates a temporary directory, clones, builds, tests, and packages.
# Exits 0 if everything passes, non-zero on first failure.

set -euo pipefail

REPO_URL="${1:-$(git config --get remote.origin.url 2>/dev/null || echo '')}"
if [ -z "$REPO_URL" ]; then
  echo "FAIL: No repo URL provided and no git remote found"
  exit 1
fi

WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

echo "=== Fresh Machine Test ==="
echo "Repo: $REPO_URL"
echo "Work: $WORK_DIR"
echo ""

# Step 1: Clone
echo "[1/6] Cloning..."
git clone --depth 1 "$REPO_URL" "$WORK_DIR/cocopilot" >/dev/null 2>&1
echo "  OK"

cd "$WORK_DIR/cocopilot"

# Step 2: Build
echo "[2/6] Building..."
go build -o cocopilot ./cmd/cocopilot
echo "  OK"

# Step 3: Verify binary runs
echo "[3/6] Verifying binary starts..."
timeout 3 ./cocopilot &
SERVER_PID=$!
sleep 1
if kill -0 "$SERVER_PID" 2>/dev/null; then
  kill "$SERVER_PID" 2>/dev/null || true
  wait "$SERVER_PID" 2>/dev/null || true
  echo "  OK"
else
  echo "  FAIL: Server exited unexpectedly"
  exit 1
fi

# Step 4: Tests
echo "[4/6] Running tests..."
go test -race -timeout 180s ./... >/dev/null 2>&1
echo "  OK"

# Step 5: Lint
echo "[5/6] Linting..."
go vet ./...
echo "  OK"

# Step 6: Package
echo "[6/6] Packaging..."
if [ -f scripts/package.sh ]; then
  bash scripts/package.sh >/dev/null 2>&1
  make verify-release 2>/dev/null
  echo "  OK"
else
  echo "  SKIP: no package.sh found"
fi

echo ""
echo "=== FRESH MACHINE TEST PASSED ==="
