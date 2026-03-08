# Decisions Log

**Last Updated**: 2026-02-12

This document records all significant decisions made during the project's evolution from a simple task queue to a comprehensive agentic system. Each decision includes context, rationale, tradeoffs, and consequences to maintain transparency and enable future decision-making.

## Decision Format

Each decision follows this structure:
- **Date**: When the decision was made
- **Status**: Locked (unchangeable) or Revisitable (can be reconsidered)
- **Context**: Situation or problem that prompted the decision
- **Decision**: What was decided
- **Rationale**: Why this choice was made
- **Tradeoffs**: Advantages and disadvantages considered
- **Consequences**: Expected outcomes and impacts
- **References**: Related documents, discussions, or code

---

## Core Architectural Decisions

### DEC-001: Go + SQLite Technology Stack
- **Date**: Project inception
- **Status**: Locked
- **Context**: Need for a simple, deployable task queue with minimal dependencies
- **Decision**: Use Go for the server and SQLite for data persistence
- **Rationale**: 
  - Go provides excellent HTTP server capabilities and concurrency
  - SQLite eliminates external database dependencies
  - Both technologies support single-binary deployment
- **Tradeoffs**:
  - Pro: Simple deployment, no external dependencies, excellent performance for moderate scale
  - Con: SQLite limits horizontal scaling, Go may be unfamiliar to some developers
- **Consequences**: 
  - System can be deployed as a single binary with embedded database
  - Scaling beyond single-machine limits will require architectural changes
- **References**: `main.go`, `go.mod`

### DEC-002: v1 API Stability Commitment
- **Date**: Pre-roadmap planning
- **Status**: Locked
- **Context**: Need to evolve system while maintaining compatibility with existing agents
- **Decision**: v1 endpoints (`/task`, `/create`, `/save`, `/events`, etc.) must remain functionally stable throughout evolution
- **Rationale**: Existing agents and integrations depend on current behavior
- **Tradeoffs**:
  - Pro: Backward compatibility ensures no disruption to existing workflows
  - Con: May limit architectural flexibility and require maintaining legacy code paths
- **Consequences**: 
  - All new features must be additive or implemented under `/api/v2/*`
  - v1 endpoints become a compatibility layer over enhanced backend
- **References**: `docs/epics/00-poc-regression.md`, `ROADMAP.md`

### DEC-003: Event-First Architecture for v2
- **Date**: Automation engine design phase
- **Status**: Revisitable
- **Context**: Need for observable, reproducible, and traceable system behavior
- **Decision**: All state changes in v2 will be captured in an append-only event log
- **Rationale**: 
  - Enables complete audit trails and debugging capabilities
  - Supports event-driven automation and real-time updates
  - Allows for replay and recovery scenarios
- **Tradeoffs**:
  - Pro: Complete observability, deterministic automation, debugging capabilities
  - Con: Additional complexity, storage overhead, potential performance impact
- **Consequences**:
  - Events table becomes source of truth for all system state changes
  - Automation engine can process events deterministically
  - UI updates can be driven by event stream
- **References**: `docs/epics/05-automation-engine.md`, `migrations/0009_events.sql`

---

## Development Process Decisions

### DEC-021: Track Plan Completion Percentage and Drift
- **Date**: 2026-02-12
- **Status**: Locked
- **Context**: ROADMAP.md and COMPLETION_SUMMARY.md needed consistent signals on progress and divergence from plan.
- **Decision**: Record plan completion percentage and drift metrics in ROADMAP.md and COMPLETION_SUMMARY.md.
- **Rationale**:
  - Ensures progress reporting is consistent across planning and status documents
  - Highlights variance from the plan early to guide course correction
- **Tradeoffs**:
  - Pro: Clear progress visibility, better alignment between planning and delivery
  - Con: Requires regular updates and discipline to keep metrics current
- **Consequences**:
  - ROADMAP.md and COMPLETION_SUMMARY.md must be updated when scope or completion changes
  - Status reviews will reference these metrics for decision-making
- **References**: `ROADMAP.md`, `COMPLETION_SUMMARY.md`

### DEC-004: Phase-Based Development Approach
- **Date**: Roadmap creation
- **Status**: Locked
- **Context**: Need to manage complexity while maintaining system stability
- **Decision**: Implement features in strict phases with dependencies between phases
- **Rationale**:
  - Reduces risk of introducing breaking changes
  - Ensures each phase delivers complete, testable functionality
  - Allows for learning and course correction between phases
- **Tradeoffs**:
  - Pro: Lower risk, incremental value delivery, easier testing and validation
  - Con: Slower overall development, potential for over-engineering early phases
- **Consequences**:
  - Each phase must pass regression tests before proceeding
  - Features may be delayed if they span multiple phases
  - Clear milestones and deliverables for project tracking
- **References**: `ROADMAP.md`, phases 0-9

### DEC-005: Feature Flags for Gradual Rollout
- **Date**: Implementation planning
- **Status**: Revisitable
- **Context**: Need to deploy new features safely without breaking existing functionality
- **Decision**: Use feature flags to control access to v2 features during development
- **Rationale**:
  - Allows testing in production without full exposure
  - Enables quick rollback if issues are discovered
  - Supports gradual user adoption
- **Tradeoffs**:
  - Pro: Risk reduction, flexible deployment, A/B testing capability
  - Con: Code complexity, configuration management overhead
- **Consequences**:
  - All risky new features must be behind flags initially
  - Flag management system needed for configuration
  - Testing must cover both flag states
- **References**: Implementation guidelines in `ROADMAP.md`

### DEC-006: SQLite Migration System
- **Date**: Schema evolution planning  
- **Status**: Revisitable
- **Context**: Need for versioned, repeatable database schema changes
- **Decision**: Implement ordered SQL migration files with version tracking
- **Rationale**:
  - Ensures consistent database state across environments
  - Supports rollback and recovery scenarios
  - Enables collaborative development with schema changes
- **Tradeoffs**:
  - Pro: Reliable schema evolution, version control integration, rollback capability
  - Con: Migration complexity, potential for migration failures
- **Consequences**:
  - All schema changes must be implemented as migrations
  - Database initialization includes migration runner
  - Migration failures must be handled gracefully
- **References**: `docs/epics/01-schema-migrations.md`, `migrations/` directory

---

## Feature Design Decisions

### DEC-007: Context Packs for Task Context
- **Date**: Memory and context design phase
- **Status**: Revisitable
- **Context**: Need for durable, immutable context bundles for tasks
- **Decision**: Implement "context packs" as immutable bundles of files, snippets, and related context
- **Rationale**:
  - Immutability ensures audit trail and reproducibility
  - Bundling reduces context fragmentation
  - Budget controls prevent context explosion
- **Tradeoffs**:
  - Pro: Reliable context preservation, clear audit trail, controlled resource usage
  - Con: Storage overhead, complexity in context selection algorithms
- **Consequences**:
 - **Consequences**:
  - Context pack creation becomes part of task claiming process
  - Storage requirements increase with task volume
  - Context selection algorithm becomes critical for usefulness
  - Context pack storage and endpoints are now implemented in v2
- **References**: `docs/epics/02-api-v2-contract.md`, `migrations/0009_memory.sql`, `migrations/0010_context_packs.sql`, v2 context pack endpoints

### DEC-020: Runs as Task Sub-Resources
- **Date**: 2026-02-11
- **Status**: Locked
- **Context**: Need durable, queryable execution history tied to tasks without inventing a separate top-level entity
- **Decision**: Represent runs as sub-resources under tasks with list and detail endpoints in v2
- **Rationale**:
  - Keeps run history scoped to a task and easy to discover
  - Aligns with task lifecycle and auditing requirements
  - Simplifies permission checks and routing
- **Tradeoffs**:
  - Pro: Clear API shape, minimal discovery overhead, consistent task ownership
  - Con: Requires joins for cross-task run analytics, more nested routes
- **Consequences**:
  - Run creation and retrieval are now part of the v2 API surface
  - Migrations and data access must enforce task ownership for runs
- **References**: `migrations/0006_runs.sql`, `docs/api/openapi-v2.yaml`, v2 runs endpoints

### DEC-008: Lease-Based Task Claiming
- **Date**: Multi-agent coordination design
- **Status**: Locked
- **Context**: Need for safe task claiming with multiple concurrent agents
- **Decision**: Implement exclusive leases with heartbeat requirements for task claiming
- **Rationale**:
  - Prevents race conditions and task duplication
  - Enables detection of failed/disconnected agents
  - Supports automatic task recovery
- **Tradeoffs**:
  - Pro: Safe concurrency, fault tolerance, abandoned task recovery
  - Con: Additional complexity, heartbeat overhead, lease management
- **Consequences**:
  - Agents must implement heartbeat protocol
  - Lease expiration handling becomes critical system component
  - Task claiming latency increases due to lease management
- **References**: `docs/epics/02-api-v2-contract.md`, `ROADMAP.md`

### DEC-018: v2 Auth Guardrails with Scoped Identities
- **Date**: 2026-02-11
- **Status**: Locked
- **Context**: v2 endpoints needed consistent access control without breaking v1 compatibility
- **Decision**: Implement API key auth for v2 mutating endpoints with optional read protection and scoped identities
- **Rationale**:
  - Enables incremental rollout without disrupting existing v1 agents
  - Allows fine-grained scope enforcement per endpoint
  - Keeps v2 error responses consistent and auditable
- **Tradeoffs**:
  - Pro: Safer rollout, clear authorization boundaries, improved observability
  - Con: Additional configuration surface, risk of misconfigured scopes
- **Consequences**:
  - v2 mutating routes enforce API keys; read routes can be gated via config
  - Unauthorized responses use the standard v2 error envelope
  - Auth decisions are logged and persisted to events for audit
- **References**: `ROADMAP.md`, v2 auth implementation snapshot

### DEC-019: Events Retention Pruning as Background Maintenance
- **Date**: 2026-02-11
- **Status**: Revisitable
- **Context**: Events volume grows quickly with leases, tasks, and audit logging
- **Decision**: Prune events on a configurable interval when retention limits are enabled
- **Rationale**:
  - Keeps storage bounded while preserving recent history
  - Avoids long-running queries over unbounded event tables
- **Tradeoffs**:
  - Pro: Predictable storage growth, better query performance
  - Con: Older audit data may be discarded if limits are too aggressive
- **Consequences**:
  - Retention settings must be communicated to operators
  - Prune jobs must report outcomes for observability
- **References**: `ROADMAP.md`, events retention notes

### DEC-009: Automation Engine with Deterministic Rules
- **Date**: Automation design phase
- **Status**: Revisitable
- **Context**: Need for automatic task creation based on events and conditions
- **Decision**: Build rule-based automation engine that processes events deterministically
- **Rationale**:
  - Event-driven approach ensures responsiveness
  - Deterministic rules enable testing and debugging
  - Bounded execution prevents automation runaway
- **Tradeoffs**:
  - Pro: Predictable behavior, testable automation, responsive task creation
  - Con: Rule complexity, potential for automation loops, debugging challenges
- **Consequences**:
  - All automation rules must be deterministic and testable
  - Idempotency mechanisms required to prevent duplicate task creation
  - Loop prevention and escalation logic becomes critical
- **References**: `docs/epics/05-automation-engine.md`

---

## Integration Decisions

### DEC-010: MCP Server for Tool Standardization
- **Date**: External integration planning
- **Status**: Revisitable
- **Context**: Need for standardized interface between cocopilot and various agent frameworks
- **Decision**: Implement Model Context Protocol (MCP) server to expose cocopilot capabilities
- **Rationale**:
  - MCP provides standard tools/resources/prompts interface
  - Reduces integration complexity for different agent frameworks
  - Enables VS Code integration without custom protocols
- **Tradeoffs**:
  - Pro: Standard interface, broad compatibility, reduced integration work
  - Con: Additional protocol layer, MCP dependency, potential limitations
- **Consequences**:
  - All major cocopilot capabilities must be exposed via MCP
  - MCP server becomes critical integration point
  - Agent frameworks need MCP client support
- **References**: `docs/epics/04-mcp-vsix.md`

### DEC-011: VS Code Extension for IDE Integration
- **Date**: Developer experience planning
- **Status**: Revisitable
- **Context**: Need for seamless integration with developer workflows
- **Decision**: Build VS Code extension ("CocoBridge") for IDE signal capture and task integration
- **Rationale**:
  - VS Code is primary development environment for many users
  - IDE signals provide valuable context for task execution
  - Seamless integration improves developer adoption
- **Tradeoffs**:
  - Pro: Enhanced context, better developer experience, workflow integration
  - Con: Platform limitation, extension maintenance overhead, privacy concerns
- **Consequences**:
  - Extension development and maintenance becomes ongoing responsibility
  - IDE signal handling must preserve user privacy
  - Context pack building can leverage IDE signals for better relevance
- **References**: `docs/epics/04-mcp-vsix.md`

---

## Security and Governance Decisions

### DEC-012: Policy-Based Governance
- **Date**: Security architecture planning
- **Status**: Revisitable
- **Context**: Need for controlled tool execution and resource access in autonomous agent operations
- **Decision**: Implement configurable policy engine for governing agent actions
- **Rationale**:
  - Autonomous agents require guardrails for safe operation
  - Policies enable organizational compliance and risk management
  - Configurable rules support different deployment environments
- **Tradeoffs**:
  - Pro: Risk reduction, compliance support, configurable security
  - Con: Performance overhead, complexity in policy definition, potential for over-restriction
- **Consequences**:
  - All potentially risky operations must go through policy evaluation
  - Policy configuration becomes part of deployment process
  - Policy violations require clear error messages and escalation paths
- **References**: `docs/epics/05-automation-engine.md`, policy enforcement sections

### DEC-013: Audit Trail for All Actions
- **Date**: Security and compliance planning
- **Status**: Locked
- **Context**: Need for complete traceability of agent actions for debugging and compliance
- **Decision**: All agent actions, tool executions, and state changes must be logged in audit trail
- **Rationale**:
  - Autonomous systems require complete observability
  - Compliance requirements often mandate audit trails
  - Debugging complex agent behaviors requires detailed logs
- **Tradeoffs**:
  - Pro: Complete traceability, compliance support, debugging capability
  - Con: Storage overhead, performance impact, privacy considerations
- **Consequences**:
  - Event log becomes permanent record of all system activity
  - Log retention and privacy policies must be implemented
  - Audit trail access controls required for sensitive deployments
- **References**: Events table schema, audit logging requirements

---

## Performance and Scalability Decisions

### DEC-014: SQLite with WAL Mode for Concurrency
- **Date**: Database performance optimization
- **Status**: Revisitable
- **Context**: Need for better concurrent read/write performance with SQLite
- **Decision**: Configure SQLite with WAL (Write-Ahead Logging) mode for improved concurrency
- **Rationale**:
  - WAL mode allows concurrent readers during writes
  - Significantly improves performance under concurrent load
  - Maintains SQLite simplicity while improving scalability
- **Tradeoffs**:
  - Pro: Better concurrency, improved performance, maintains simplicity
  - Con: Additional WAL files, slightly more complex backup procedures
- **Consequences**:
  - Concurrent agent performance improves significantly
  - Backup procedures must account for WAL files
  - Database file management becomes slightly more complex
- **References**: SQLite performance settings in `ROADMAP.md`

### DEC-015: Bounded Resource Usage
- **Date**: System reliability planning
- **Status**: Locked
- **Context**: Need to prevent resource exhaustion in autonomous agent operations
- **Decision**: Implement hard limits on context pack size, automation task creation, and memory usage
- **Rationale**:
  - Autonomous systems can consume unbounded resources without limits
  - Hard limits prevent system degradation and failures
  - Resource budgets enable predictable system behavior
- **Tradeoffs**:
  - Pro: Predictable resource usage, system stability, performance consistency
  - Con: Potential for legitimate use cases to hit limits, tuning complexity
- **Consequences**:
  - All resource-intensive operations must have configurable limits
  - Limit exceeded scenarios require graceful handling and user feedback
  - Resource monitoring becomes critical for system health
- **References**: Context pack budgets, automation quotas in `ROADMAP.md`

---

## Future Consideration Decisions

### DEC-016: Database Migration Path Reserved
- **Date**: Long-term architecture planning
- **Status**: Locked
- **Context**: Recognition that SQLite may not scale indefinitely
- **Decision**: Reserve migration path to distributed database while maintaining SQLite for initial deployments
- **Rationale**:
  - SQLite provides excellent developer experience and simple deployment
  - Large-scale deployments will eventually require distributed databases
  - Migration path planning prevents architectural dead-ends
- **Tradeoffs**:
  - Pro: Smooth scaling path, maintains current simplicity, future flexibility
  - Con: Architectural complexity to support multiple backends
- **Consequences**:
  - Database abstraction layer must be designed for multiple backends
  - Migration tools and procedures will be required for scaling
  - Feature development must consider multi-backend compatibility
- **References**: Scalability considerations in architecture documentation

### DEC-017: Microservice Decomposition Not Yet Needed
- **Date**: Scalability architecture review
- **Status**: Revisitable
- **Context**: Evaluation of whether to decompose into microservices for scalability
- **Decision**: Maintain monolithic architecture until clear scalability bottlenecks are identified
- **Rationale**:
  - Premature optimization adds complexity without proven benefit
  - Current architecture can handle expected load
  - Microservices introduce operational complexity
- **Tradeoffs**:
  - Pro: Maintains simplicity, easier development and deployment, better performance
  - Con: May require significant refactoring if scaling becomes necessary
- **Consequences**:
  - Single deployment artifact and operational model
  - Scaling initially focuses on vertical scaling and database optimization
  - Service boundaries must be designed to enable future decomposition
- **References**: Deployment architecture in system documentation; COMPLETION_SUMMARY.md for current progress tracking

---

## Decision Review Process

Decisions marked as "Revisitable" should be reviewed when:
- New requirements emerge that conflict with current decisions
- Performance or scalability issues arise that weren't anticipated
- New technologies or approaches become available that offer significant advantages
- User feedback indicates problems with current approach

All decision revisions must:
1. Reference the original decision and explain why revision is needed
2. Document new context and requirements
3. Analyze impact on existing implementations
4. Provide migration path if breaking changes are required
5. Update all affected documentation and code

This decision log serves as the authoritative record of project reasoning and must be consulted before making any architectural or significant feature changes.