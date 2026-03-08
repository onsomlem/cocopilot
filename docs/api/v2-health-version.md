# API v2 Health and Version Endpoints

Last updated: 2026-02-12

## Overview

The v2 API provides endpoints for monitoring service health, retrieving version information, and inspecting a safe runtime configuration snapshot.

If API key auth is enabled for reads (COCO_REQUIRE_API_KEY_READS=true), these endpoints require the X-API-Key header with a key that has the v2:read scope (or a broader scope like v2:* or *).

## Endpoints

### GET /api/v2/health

**Purpose**: Quick health check for monitoring and load balancing.

**Response Time**: < 100ms (typically < 1ms)

**Status Codes**:
- `200 OK`: Service is healthy
- `401 UNAUTHORIZED`: Missing or invalid API key (when read auth is enabled)
- `403 FORBIDDEN`: API key present but missing required scope (when scoped identities are used)

**Response Body**:
```json
{
  "ok": true
}
```

**Example**:
```bash
curl http://127.0.0.1:8080/api/v2/health
```

**Use Cases**:
- Load balancer health checks
- Monitoring systems
- Service availability verification
- Uptime checks

---

### GET /api/v2/version

**Purpose**: Retrieve service version, API support, and database schema version.

**Status Codes**:
- `200 OK`: Successfully retrieved version information
- `401 UNAUTHORIZED`: Missing or invalid API key (when read auth is enabled)
- `403 FORBIDDEN`: API key present but missing required scope (when scoped identities are used)

**Response Body**:
```json
{
  "service": "cocopilot",
  "version": "1.0.0",
  "api": {
    "v1": true,
    "v2": true
  },
  "schema_version": 18,
  "retention": {
    "enabled": true,
    "interval_seconds": 3600,
    "max_rows": 0,
    "days": 30
  }
}
```

**Response Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `service` | string | Service name (always "cocopilot") |
| `version` | string | Service version in semantic versioning format |
| `api` | object | API version support flags |
| `api.v1` | boolean | Whether v1 API is supported |
| `api.v2` | boolean | Whether v2 API is supported |
| `schema_version` | integer | Current database schema version from `schema_migrations` table (0 if unavailable) |
| `retention` | object | Events retention configuration snapshot |
| `retention.enabled` | boolean | Whether retention pruning is enabled (`days > 0` or `max_rows > 0`) |
| `retention.interval_seconds` | integer | Prune interval in seconds |
| `retention.max_rows` | integer | Max events to retain (0 disables row-based pruning) |
| `retention.days` | integer | Retention window in days (0 disables age-based pruning) |

**Example**:
```bash
curl http://127.0.0.1:8080/api/v2/version
```

**Use Cases**:
- Verify API compatibility
- Check database migration status
- Service discovery
- Client compatibility checks
- Debugging and support

---

### GET /api/v2/config

**Purpose**: Retrieve a safe runtime configuration snapshot (no secrets).

**Status Codes**:
- `200 OK`: Successfully retrieved configuration
- `401 UNAUTHORIZED`: Missing or invalid API key (when read auth is enabled)
- `403 FORBIDDEN`: API key present but missing required scope (when scoped identities are used)

**Response Body**:
```json
{
  "db_path": "[redacted]",
  "auth": {
    "required": true,
    "require_reads": true,
    "identity_count": 2,
    "legacy_api_key_set": false
  },
  "retention": {
    "enabled": true,
    "interval_seconds": 3600,
    "max_rows": 0,
    "days": 30
  },
  "sse": {
    "heartbeat_seconds": 30,
    "replay_limit_max": 500
  }
}
```

**Response Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `db_path` | string | Always `"[redacted]"` (never returns the actual DB path) |
| `auth` | object | Auth configuration summary |
| `auth.required` | boolean | Whether API key auth is enabled for v2 requests |
| `auth.require_reads` | boolean | Whether read requests require auth |
| `auth.identity_count` | integer | Number of configured API identities |
| `auth.legacy_api_key_set` | boolean | Whether legacy COCO_API_KEY is set |
| `retention` | object | Events retention configuration snapshot |
| `retention.enabled` | boolean | Whether retention pruning is enabled (`days > 0` or `max_rows > 0`) |
| `retention.interval_seconds` | integer | Prune interval in seconds |
| `retention.max_rows` | integer | Max events to retain (0 disables row-based pruning) |
| `retention.days` | integer | Retention window in days (0 disables age-based pruning) |
| `sse` | object | SSE configuration summary |
| `sse.heartbeat_seconds` | integer | SSE heartbeat interval in seconds |
| `sse.replay_limit_max` | integer | Max events allowed for replay |

**Example**:
```bash
curl http://127.0.0.1:8080/api/v2/config
```

---

## Error Handling

Both endpoints use the canonical v2 error envelope:

```json
{
  "error": {
    "code": "METHOD_NOT_ALLOWED",
    "message": "Method not allowed",
    "details": {
      "method": "POST",
      "allowed_methods": ["GET"]
    }
  }
}
```

Typical codes for these endpoints:
- **405 METHOD_NOT_ALLOWED**: Unsupported method (only `GET` is allowed)
- **401 UNAUTHORIZED**: Missing or invalid API key (when read auth is enabled)
- **403 FORBIDDEN**: API key present but missing required scope (when scoped identities are used)
- **500 INTERNAL**: Internal failure (rare, e.g. DB/schema lookup issue on version endpoint)

Additional canonical examples used across v2 endpoints:

```json
{
  "error": {
    "code": "INVALID_ARGUMENT",
    "message": "name and workdir are required",
    "details": {
      "field": "workdir"
    }
  }
}
```

```json
{
  "error": {
    "code": "CONFLICT",
    "message": "Task is already leased by another agent",
    "details": {
      "task_id": 123,
      "agent_id": "agent-2"
    }
  }
}
```

## Implementation Notes

### Health Endpoint
- Minimal overhead (no database queries)
- Suitable for high-frequency polling
- Returns immediately with static response

### Version Endpoint
- Performs one lightweight database query to retrieve schema version
- Query: `SELECT MAX(version) FROM schema_migrations`
- Cached connection pool minimizes overhead
- Response time typically < 10ms

### Config Endpoint
- Returns a safe snapshot of runtime config
- Redacts `db_path` and does not expose API keys
- Includes auth, retention, and SSE summaries

## Integration Examples

### Shell Script Monitoring
```bash
#!/bin/bash
# Check if service is healthy
if curl -f -s http://127.0.0.1:8080/api/v2/health > /dev/null; then
  echo "Service is healthy"
else
  echo "Service is down"
  exit 1
fi
```

### Version Check in Client Code
```javascript
// Check API compatibility
const resp = await fetch('http://127.0.0.1:8080/api/v2/version');
const version = await resp.json();

if (!version.api.v2) {
  throw new Error('API v2 not supported by this server');
}

console.log(`Connected to ${version.service} v${version.version}`);
console.log(`Database schema version: ${version.schema_version}`);
```

### Python Client
```python
import requests

# Get version info
resp = requests.get('http://127.0.0.1:8080/api/v2/version')
version = resp.json()

print(f"Service: {version['service']}")
print(f"API v2 supported: {version['api']['v2']}")
print(f"Schema version: {version['schema_version']}")
```

## Testing

### Unit Tests
Health and version endpoints have coverage in `main_test.go`:
- `TestV2HealthEndpoint`: Validates response format and timing
- `TestV2HealthEndpointMethodNotAllowed`: Verifies method restrictions
- `TestV2VersionEndpoint`: Validates response structure and content
- `TestV2VersionEndpointSchemaVersion`: Verifies schema version accuracy
- `TestV2VersionEndpointMethodNotAllowed`: Verifies method restrictions

Config endpoint coverage lives in `v2_config_test.go`:
- `TestV2ConfigReturnsSafeRuntimeConfig`: Validates redactions and fields
- `TestV2ConfigAuthScopeEnforced`: Verifies read-scope enforcement when auth is enabled

### Manual Testing
```bash
# Test health endpoint
curl -v http://127.0.0.1:8080/api/v2/health

# Test version endpoint
curl -v http://127.0.0.1:8080/api/v2/version

# Test method restriction (should return 405)
curl -X POST http://127.0.0.1:8080/api/v2/health
```

## Performance Characteristics

| Endpoint | Avg Response Time | Database Queries | Suitable for Polling |
|----------|------------------|------------------|---------------------|
| `/api/v2/health` | < 1ms | 0 | Yes (high frequency) |
| `/api/v2/version` | < 10ms | 1 (lightweight) | Yes (moderate frequency) |
| `/api/v2/config` | < 10ms | 0 | Yes (moderate frequency) |

## Future Enhancements

Potential additions to the version endpoint:
- Git commit hash
- Build timestamp
- Uptime information
- Feature flags status
- Runtime configuration info
