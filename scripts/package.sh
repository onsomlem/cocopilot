#!/usr/bin/env bash
set -euo pipefail

# Release packaging script for Cocopilot.
# Builds the Go binary and creates a clean zip free of development artifacts.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BUILD_DIR="$ROOT_DIR/dist"
BINARY_NAME="cocopilot"
ZIP_NAME="cocopilot-release.zip"

# Forbidden patterns that must never appear in a release zip.
FORBIDDEN_PATTERNS=(
    ".git/"
    "__MACOSX/"
    ".DS_Store"
    "tasks.db"
    "tasks.db-wal"
    "tasks.db-shm"
    "coverage.out"
    "*.exe~"
)

echo "=== Cocopilot Release Packaging ==="

# Step 1: Build the Go binary.
echo "[1/4] Building Go binary..."
cd "$ROOT_DIR"
VERSION=$(cat "$ROOT_DIR/VERSION" 2>/dev/null || echo dev)
CGO_ENABLED=0 go build -ldflags "-X github.com/onsomlem/cocopilot/server.Version=${VERSION}" -o "$BUILD_DIR/$BINARY_NAME" ./cmd/cocopilot
echo "  Built: $BUILD_DIR/$BINARY_NAME"

# Step 2: Stage release files.
echo "[2/4] Staging release files..."
STAGING_DIR=$(mktemp -d)
trap 'rm -rf "$STAGING_DIR"' EXIT

# Copy relevant files to staging, excluding forbidden patterns.
rsync -a --exclude='.git/' \
         --exclude='__MACOSX/' \
         --exclude='.DS_Store' \
         --exclude='tasks.db*' \
         --exclude='coverage.out' \
         --exclude='*.exe~' \
         --exclude='dist/' \
         --exclude='tmp/' \
         --exclude='task-server' \
         --exclude='*.test' \
         --exclude='node_modules/' \
         "$ROOT_DIR/" "$STAGING_DIR/cocopilot/"

# Place the built binary in the staging directory.
cp "$BUILD_DIR/$BINARY_NAME" "$STAGING_DIR/cocopilot/$BINARY_NAME"

# Step 3: Create the zip.
echo "[3/4] Creating release zip..."
mkdir -p "$BUILD_DIR"
cd "$STAGING_DIR"
zip -rq "$BUILD_DIR/$ZIP_NAME" cocopilot/
echo "  Created: $BUILD_DIR/$ZIP_NAME"

# Step 4: Verify the zip is clean.
echo "[4/4] Verifying zip contents..."
VERIFY_DIR=$(mktemp -d)
# Extend trap to clean up both temp dirs.
trap 'rm -rf "$STAGING_DIR" "$VERIFY_DIR"' EXIT

unzip -q "$BUILD_DIR/$ZIP_NAME" -d "$VERIFY_DIR"

VIOLATIONS=0
for pattern in "${FORBIDDEN_PATTERNS[@]}"; do
    # Use find with -path or -name depending on pattern type.
    if [[ "$pattern" == *"/"* ]]; then
        # Directory pattern: search by path component.
        dir_name="${pattern%/}"
        if find "$VERIFY_DIR" -type d -name "$dir_name" 2>/dev/null | grep -q .; then
            echo "  VIOLATION: found forbidden directory '$pattern'"
            VIOLATIONS=$((VIOLATIONS + 1))
        fi
    elif [[ "$pattern" == *"*"* ]]; then
        # Glob pattern.
        if find "$VERIFY_DIR" -name "$pattern" 2>/dev/null | grep -q .; then
            echo "  VIOLATION: found forbidden file pattern '$pattern'"
            VIOLATIONS=$((VIOLATIONS + 1))
        fi
    else
        # Exact name match.
        if find "$VERIFY_DIR" -name "$pattern" 2>/dev/null | grep -q .; then
            echo "  VIOLATION: found forbidden file '$pattern'"
            VIOLATIONS=$((VIOLATIONS + 1))
        fi
    fi
done

if [ "$VIOLATIONS" -gt 0 ]; then
    echo ""
    echo "FAIL: $VIOLATIONS forbidden artifact(s) found in release zip!"
    exit 1
fi

FILE_COUNT=$(find "$VERIFY_DIR" -type f | wc -l | tr -d ' ')
ZIP_SIZE=$(du -sh "$BUILD_DIR/$ZIP_NAME" | cut -f1)
echo "  Clean: $FILE_COUNT files, $ZIP_SIZE total"
echo ""
echo "=== Release package ready: $BUILD_DIR/$ZIP_NAME ==="
