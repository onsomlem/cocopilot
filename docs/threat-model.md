# Threat Model — Cocopilot

## Trust Boundaries

```
┌──────────────────────────────────────────────────────────┐
│  Browser / Agent (Untrusted)                             │
│    ↕  HTTP (JSON / SSE / form-encoded)                   │
├──────────────────────────────────────────────────────────┤
│  Reverse Proxy (nginx / Caddy)         ← TLS terminates │
│    ↕  plain HTTP                                         │
├──────────────────────────────────────────────────────────┤
│  Cocopilot Process (Trusted)                             │
│    ↕  SQLite driver (in-process)                         │
├──────────────────────────────────────────────────────────┤
│  SQLite Database File (Trusted, local)                   │
└──────────────────────────────────────────────────────────┘
```

### Boundary 1 — Network ↔ Cocopilot

All HTTP input is untrusted. Agents, browsers, and automated systems send
requests over the network.

**Mitigations in place:**

| Control | Implementation |
|---------|---------------|
| API key auth | `COCO_REQUIRE_API_KEY=true` + `COCO_API_KEY` env var. Header `Authorization: Bearer <key>`. |
| Identity-scoped keys | `COCO_API_IDENTITIES` JSON for per-agent key isolation. |
| Policy enforcement | Per-project policies restrict which agents can perform which actions. |
| Auth on mutations | v1 mutation endpoints (`/create`, `/save`, `/delete`) gated by `withV1MutationAuth`. |
| Auth on v2 routes | All v2 routes wrapped with `withV2Auth`. |
| CORS | `withCORS` middleware echoes `Origin`, sets `Access-Control-Allow-*` headers, handles preflight. |

### Boundary 2 — Cocopilot ↔ SQLite

The SQLite database is a local file. Access is restricted to the process owner.

**Mitigations in place:**

| Control | Implementation |
|---------|---------------|
| Parameterized queries | All SQL uses `?` placeholders — no string interpolation. |
| WAL mode | `PRAGMA journal_mode=WAL` for concurrent read safety. |
| Foreign keys | `PRAGMA foreign_keys=ON` enforces referential integrity. |
| Backup/restore | `/api/v2/backup` streams a consistent snapshot; restore validates schema version. |

## Attack Vectors

### 1. Unauthenticated Access

**Risk:** Without `COCO_REQUIRE_API_KEY=true`, all endpoints are open.

**Mitigation:** Enable API key auth for production deployments. Documentation
recommends this default in `docs/security.md` and `docs/quickstart.md`.

**Residual risk:** Low — single-user/dev deployments on localhost are the
default use case.

### 2. SQL Injection

**Risk:** Malicious input in task titles, instructions, agent IDs, etc.

**Mitigation:** Every database query uses parameterized statements (`db.Exec`
/ `db.QueryRow` with `?` placeholders). No dynamic SQL construction.

**Residual risk:** Negligible.

### 3. Cross-Site Scripting (XSS)

**Risk:** The Kanban UI renders task content in HTML.

**Mitigation:** Go's `html/template` package auto-escapes all interpolated
values. The UI uses Alpine.js with `x-text` (not `x-html`).

**Residual risk:** Low — review any future `x-html` usage.

### 4. Server-Side Request Forgery (SSRF)

**Risk:** The server does not make outbound HTTP requests on behalf of users.

**Mitigation:** No outbound HTTP client code exists. Automation rules execute
in-process task creation only.

**Residual risk:** None currently.

### 5. Denial of Service

**Risk:** Resource exhaustion via bulk task creation, SSE connection flooding,
or large payloads.

**Mitigations:**
- Automation rules have rate limiting (`COCO_AUTOMATION_RATE_LIMIT`,
  `COCO_AUTOMATION_BURST_LIMIT`) and circuit breaker.
- Event pruning runs on a configurable interval.
- Lease expiry goroutine prevents zombie claims.
- Request body size is bounded by Go's default `http.MaxBytesReader` when
  behind a reverse proxy with body limits.

**Residual risk:** Medium — no built-in request rate limiting at the HTTP
layer. Recommend reverse proxy rate limits for production.

### 6. Data Exfiltration

**Risk:** Unauthorized read access to tasks, memory, or project data.

**Mitigations:**
- API key auth gates all v2 endpoints.
- `COCO_REQUIRE_API_KEY_READS=true` can gate read endpoints too.
- Policy engine can restrict per-agent access.

**Residual risk:** Low when auth is enabled.

### 7. Path Traversal

**Risk:** Embedded static file server could serve unintended files.

**Mitigation:** Static files use `go:embed` with a fixed `static/` directory.
`http.FS` on an embed.FS cannot escape the embedded tree.

**Residual risk:** None.

## Recommendations

1. **Always enable API key auth** for any non-localhost deployment.
2. **Deploy behind a reverse proxy** (nginx, Caddy) for TLS, rate limiting,
   and request size limits.
3. **Restrict SQLite file permissions** to the process owner (`chmod 600`).
4. **Monitor `/api/v2/metrics`** for anomalous task counts or agent activity.
5. **Use `COCO_API_IDENTITIES`** to give each agent a distinct key for
   audit traceability.
6. **Review automation rules** — misconfigured rules can create task loops
   (mitigated by depth/rate limits and circuit breaker).
