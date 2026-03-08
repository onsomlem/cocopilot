#!/usr/bin/env bash
# check-contract-drift.sh
# Checks that API endpoints declared in docs/api/openapi-v2.yaml have
# corresponding routes registered in server/, that MCP tools in
# tools/cocopilot-mcp/src/index.ts map to known v2 API methods, and
# that shared TypeScript types cover the OpenAPI schemas.
#
# Exit code: 0 = OK, 1 = DRIFT DETECTED

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SPEC_FILE="$SCRIPT_DIR/docs/api/openapi-v2.yaml"
ROUTES_GO="$SCRIPT_DIR/server/routes.go"
SERVER_DIR="$SCRIPT_DIR/server"
MCP_INDEX="$SCRIPT_DIR/tools/cocopilot-mcp/src/index.ts"
SHARED_TYPES="$SCRIPT_DIR/tools/shared/types.ts"

drift=0

# ---------------------------------------------------------------------------
# 1. Check that each /api/v2/... path in the OpenAPI spec exists in main.go
# ---------------------------------------------------------------------------
echo "=== Checking OpenAPI spec paths against server/ routes ==="

while IFS= read -r raw_path; do
  # Strip leading whitespace and the leading '/'
  path="${raw_path#"${raw_path%%[! ]*}"}"
  path="${path#/}"

  # Replace OpenAPI path parameters {id} -> variable search token
  # We just check that the static prefix is routed.
  static_prefix=$(echo "$path" | sed 's/{[^}]*}.*//')
  static_prefix="/${static_prefix%/}"

  # Check if server/ has a HandleFunc or route detection for this path prefix.
  if ! grep -rqF "$static_prefix" "$SERVER_DIR"/*.go; then
    echo "  MISSING ROUTE: $static_prefix (from spec path /$path)"
    drift=1
  fi
done < <(grep -E '^\s+/api/v2/' "$SPEC_FILE" | grep -v '#' | sed 's/:.*//' | sort -u)

# ---------------------------------------------------------------------------
# 2. Check that MCP tools map to known v2 API endpoints
# ---------------------------------------------------------------------------
echo ""
echo "=== Checking MCP tool endpoints against server/ ==="

check_mcp_tool() {
  local tool="$1"
  local expected_path="$2"
  if ! grep -qF "\"$tool\"" "$MCP_INDEX"; then
    echo "  MCP TOOL MISSING: $tool"
    drift=1
    return
  fi
  if ! grep -rqF "$expected_path" "$SERVER_DIR"/*.go; then
    echo "  ROUTE MISSING FOR MCP TOOL: $tool (expected path: $expected_path)"
    drift=1
  fi
}

check_mcp_tool "coco.project.list" "/api/v2/projects"
check_mcp_tool "coco.project.create" "/api/v2/projects"
check_mcp_tool "coco.project.get" "/api/v2/projects/"
check_mcp_tool "coco.project.update" "/api/v2/projects/"
check_mcp_tool "coco.project.delete" "/api/v2/projects/"
check_mcp_tool "coco.config.get" "/api/v2/config"
check_mcp_tool "coco.version.get" "/api/v2/version"
check_mcp_tool "coco.health.get" "/api/v2/health"
check_mcp_tool "coco.agent.list" "/api/v2/agents"
check_mcp_tool "coco.agent.get" "/api/v2/agents/"
check_mcp_tool "coco.agent.delete" "/api/v2/agents/"
check_mcp_tool "coco.project.tasks.list" "/api/v2/projects/"
check_mcp_tool "coco.project.memory.query" "/api/v2/projects/"
check_mcp_tool "coco.project.memory.put" "/api/v2/projects/"
check_mcp_tool "coco.project.audit.list" "/api/v2/projects/"
check_mcp_tool "coco.policy.list" "/api/v2/projects/"
check_mcp_tool "coco.policy.get" "/api/v2/projects/"
check_mcp_tool "coco.policy.create" "/api/v2/projects/"
check_mcp_tool "coco.policy.update" "/api/v2/projects/"
check_mcp_tool "coco.policy.delete" "/api/v2/projects/"
check_mcp_tool "coco.project.events.replay" "/api/v2/projects/"
check_mcp_tool "coco.project.automation.rules" "/api/v2/projects/"
check_mcp_tool "coco.project.automation.simulate" "/api/v2/projects/"
check_mcp_tool "coco.project.automation.replay" "/api/v2/projects/"
check_mcp_tool "coco.context_pack.create" "/api/v2/projects/"

# ---------------------------------------------------------------------------
# 3. Check that shared TypeScript types cover key OpenAPI schemas
# ---------------------------------------------------------------------------
echo ""
echo "=== Checking shared TypeScript types against OpenAPI schemas ==="

if [ ! -f "$SHARED_TYPES" ]; then
  echo "  MISSING: $SHARED_TYPES does not exist"
  drift=1
else
  # Check for key interfaces that must exist in shared types
  for iface in Project TaskV2 Run Lease Agent Event Memory Policy \
               ContextPack Artifact RunStep RunLog ToolInvocation \
               AssignmentEnvelope CompletionContract TaskContext RepoFile V2Error \
               DashboardData DashboardRecommendation DashboardStalledTask; do
    if ! grep -qE "^export interface $iface " "$SHARED_TYPES"; then
      echo "  MISSING TYPE: interface $iface not found in shared/types.ts"
      drift=1
    fi
  done

  # Check for key type aliases
  for alias in TaskStatusV2 TaskType RunStatus AgentStatus StepStatus; do
    if ! grep -qE "^export type $alias " "$SHARED_TYPES"; then
      echo "  MISSING TYPE: type $alias not found in shared/types.ts"
      drift=1
    fi
  done
fi

# ---------------------------------------------------------------------------
# 4. Check dashboard endpoint exists in both spec and main.go
# ---------------------------------------------------------------------------
echo ""
echo "=== Checking dashboard endpoint ==="
if ! grep -rqF "v2ProjectDashboardHandler" "$SERVER_DIR"/*.go; then
  echo "  MISSING: dashboard handler not registered in server/"
  drift=1
fi

# ---------------------------------------------------------------------------
# 5. Reverse check: main.go routes should appear in the OpenAPI spec
# ---------------------------------------------------------------------------
echo ""
echo "=== Checking server/ routes have matching OpenAPI spec entries ==="

while IFS= read -r route; do
  # Extract the path from HandleFunc("/api/v2/...")
  path=$(echo "$route" | sed -n 's/.*HandleFunc("\(\/api\/v2\/[^"]*\)".*/\1/p')
  if [ -z "$path" ]; then
    continue
  fi

  # Normalize: strip trailing slash for matching
  norm_path="${path%/}"

  if ! grep -qF "$norm_path" "$SPEC_FILE"; then
    echo "  SPEC MISSING: $norm_path (registered in server/ but not in OpenAPI spec)"
    drift=1
  fi
done < <(grep -r 'HandleFunc("/api/v2/' "$SERVER_DIR"/*.go)

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
if [ $drift -eq 0 ]; then
  echo "OK"
  exit 0
else
  echo "DRIFT DETECTED"
  exit 1
fi
