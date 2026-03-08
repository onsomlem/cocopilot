# System Architecture

**Last Updated**: 2026-03-08

## Current Architecture (v1 + v2)

### Components

**1. Go HTTP Server** (`main.go`)
- Port: 8080 (default)
- Serves static Kanban UI plus v1 and v2 API endpoints
- Handles concurrent requests with built-in Go HTTP server
- Background jobs for expired lease cleanup and events retention pruning
- Single-process deployment model

**2. SQLite Database** (`tasks.db`)
- File-based storage in working directory
- Schema migrations for tasks, projects, runs, agents, leases, events, and related tables
- Atomic operations for task and lease state transitions
- No external database dependencies

**3. Web Frontend** (embedded HTML/CSS/JS)
- Alpine.js for reactive UI components
- Kanban board with drag-drop functionality
- Server-Sent Events for real-time updates
- VS Code-themed styling

**4. Agent Interface** (HTTP API)
- RESTful endpoints for v1 and v2 task lifecycle management
- Lease-backed task claiming (`GET /task`, `POST /api/v2/tasks/:id/claim`)
- Context inheritance from parent tasks
- SSE for v1 (`GET /events`) and v2 (`GET /api/v2/events/stream`) events
- Runtime config and version endpoints (`GET /api/v2/config`, `GET /api/v2/version`)
- Memory endpoints for project knowledge (`GET/PUT /api/v2/projects/:projectId/memory`)
- Context pack generation (`POST /api/v2/projects/:projectId/context-packs`)
- Run details and sub-resources (`GET /api/v2/runs/:runId`, `GET /api/v2/runs/:runId/steps`, `GET /api/v2/runs/:runId/logs`, `GET /api/v2/runs/:runId/artifacts`)

**5. MCP Server** (optional, separate process)
- Located in `tools/cocopilot-mcp`
- Exposes tools/resources/prompts for MCP-capable clients
- Currently runs alongside the Go server; no direct in-process coupling

**6. VS Code Extension (VSIX)** (optional, preview)
- Located in `tools/cocopilot-vsix`
- Provides IDE-side signals and helper actions for Cocopilot workflows
- Not required for server operation; integration is early-stage

### Data Flow

```
┌─────────────────┐     ┌─────────────────┐
│   Web Browser   │◀───▶│   Go Server     │
│  (Kanban UI)    │     │   :8000         │
└─────────────────┘     └────────┬────────┘
                                 │
┌─────────────────┐              │
│   LLM Agent     │──────────────┤
│  (polls /task)  │              │
└─────────────────┘              ▼
                        ┌─────────────────┐
                        │   SQLite DB     │
                        │   tasks.db      │
                        └─────────────────┘

┌─────────────────┐     ┌─────────────────┐
│  MCP Clients    │◀───▶│  MCP Server     │
│  (optional)     │     │  (sidecar)      │
└─────────────────┘     └────────┬────────┘
                                 │
                                 ▼
                        ┌─────────────────┐
                        │   Go Server     │
                        │   HTTP API      │
                        └─────────────────┘

┌─────────────────┐     ┌─────────────────┐
│ VS Code (VSIX)  │◀───▶│   Go Server     │
│ (optional)      │     │   HTTP API      │
└─────────────────┘     └─────────────────┘
```

### Request Flow

1. **Task Creation**: UI/Agent → POST /create or POST /api/v2/tasks → DB INSERT → SSE event
2. **Task Claiming**: Agent → GET /task or POST /api/v2/tasks/:id/claim → Lease acquisition → Context building
3. **Task Completion**: Agent → POST /save or POST /api/v2/tasks/:id/complete → DB UPDATE + lease release → SSE event
4. **Real-time Updates**: DB change → SSE broadcast → UI update
5. **Memory Read/Write**: UI/Agent → GET/PUT /api/v2/projects/:projectId/memory → DB query/upsert → SSE event
6. **Context Pack Build**: Agent → POST /api/v2/projects/:projectId/context-packs → DB INSERT → Context pack response
7. **Run Observability**: UI/Agent → GET /api/v2/runs/:runId/{steps,logs,artifacts} → DB SELECT → UI update
8. **Retention Pruning**: Background job → events table cleanup → retention metrics in logs

## Target Architecture (Project Brain)

### Expanded Components

**1. Core Server** (Go)
- Migration runner for schema evolution
- Feature flag system for gradual rollout
- Event-driven architecture with append-only log
- Policy engine for governance and security

**2. Database Layer** (SQLite → Multi-table)
- Projects: Multi-project support with scoped settings
- Runs: Execution ledger with steps and artifacts
- Events: Append-only log of all state changes
- Memory: Durable context storage across sessions
- Leases: Safe multi-agent task claiming
- Dependencies: DAG support for task orchestration

**3. Agent Protocol** (Enhanced HTTP API)
- Structured task completion with metadata
- Lease-based claiming with heartbeats
- Context pack integration for rich context
- Tool execution logging and artifact management

**4. Automation Engine** (Internal Module)
- Event processing for automatic task creation
- Blocker classification and resolution workflows
- Deterministic rule execution with idempotency
- Loop prevention and escalation mechanisms

**5. Frontend Evolution** (Progressive Enhancement)
- Project selector and multi-project support
- Run viewer for execution observability
- Memory panel for context management
- Task dependency visualization
- Agent dashboard for coordination monitoring

**6. External Integrations**
- MCP server for tool/resource/prompt exposure
- VS Code extension for IDE signal capture
- Webhook system for external notifications

### Enhanced Data Flow

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   VS Code       │────▶│  Cocopilot      │◀───▶│   Web UI        │
│   Extension     │     │  Server         │     │  (Multi-view)   │
└─────────────────┘     └─────────┬───────┘     └─────────────────┘
                                  │
┌─────────────────┐     ┌─────────▼───────┐     ┌─────────────────┐
│   MCP Client    │◀───▶│  Event Bus      │◀───▶│  Automation     │
│   (Agents)      │     │  (SQLite)       │     │  Engine         │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

## Database Schema Evolution

### Phase 0-1: Foundation
- schema_migrations: Version tracking
- projects: Multi-project support  
- tasks: Enhanced with project_id, type, priority

### Phase 2-3: Execution & Coordination
- agents: Registration and capabilities
- runs: Execution attempts with metadata
- run_steps: Granular progress tracking
- artifacts: Generated files/diffs/reports
- leases: Exclusive task claiming

### Phase 4-5: Intelligence & Memory
- events: Append-only state change log
- memory_items: Durable context storage
- context_packs: Immutable context bundles
- automation_emissions: Idempotency tracking

### Phase 6+: Advanced Features
- task_dependencies: DAG relationships
- policies: Governance rules
- repo_files: File system monitoring

## Security Architecture

### Authentication & Authorization
- Optional API key guardrails for v2 endpoints
- Read protection toggle for v2 endpoints (including SSE stream)
- Scoped identities for endpoint-level permissions
- Auth decision logging with denial events persisted to the events stream

### Data Protection
- Input validation and sanitization
- No sensitive data in logs
- Secure defaults for all configurations
- Privacy-preserving context handling

## Scalability Considerations

### Current Limitations
- Single SQLite file limits concurrent writes
- In-memory SSE client management
- Single-process deployment model
- No horizontal scaling support

### Future Scaling Paths
- WAL mode for better SQLite concurrency
- Redis for SSE client coordination
- Database clustering/replication
- Microservice decomposition for high load

## Deployment Architecture

### Development
```bash
# Local development
go run main.go
# Database: ./tasks.db
# Config: Environment variables
```

### Production
```bash
# Binary deployment
./cocopilot
# Database: Persistent volume mount
# Config: Environment variables or config file
# Monitoring: Prometheus metrics (future)
# Logs: Structured JSON logging
```

### Container Deployment
```dockerfile
FROM golang:1.21-alpine AS builder
COPY . /app
WORKDIR /app
RUN go build -o cocopilot ./cmd/cocopilot

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/cocopilot /usr/local/bin/
VOLUME ["/data"]
ENV COCO_DB_PATH=/data/tasks.db
EXPOSE 8080
CMD ["cocopilot"]
```

## Performance Characteristics

### Current Benchmarks
- Task creation: <10ms p95
- Task claiming: <20ms p95  
- SSE broadcast: <50ms p95
- Concurrent agents: ~50 without degradation

### Optimization Strategies
- Database connection pooling
- Prepared statement caching
- Background task processing
- Efficient SSE client management

## Monitoring & Observability

### Metrics (Planned)
- Task throughput and latency
- Agent connection counts and health
- Database performance metrics
- Memory usage and garbage collection

### Logging Strategy
- Structured JSON logs with correlation IDs
- Different log levels for development vs production
- Agent action audit trail
- Performance and error tracking
- Events retention prune outcomes logged with deleted counts and durations

This architecture evolves incrementally, preserving the simplicity of the current PoC while building toward a comprehensive agentic system.