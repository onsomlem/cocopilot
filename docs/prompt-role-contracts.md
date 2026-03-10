# Prompt-Role Contracts

**Version**: 1.0  
**Status**: Active  

## Overview

Each prompt role defines a contract between the orchestrator and the LLM for a specific stage or function. Prompt roles are the unit of registration, versioning, and customization in the prompt registry.

---

## Role Index

| Role | Stage | Category |
|------|-------|----------|
| `bootstrap` | — | Lifecycle |
| `runtime_worker` | — | Lifecycle |
| `idle_recon` | Stage 0 | Planning |
| `idle_continuity` | Stage 1 | Planning |
| `idle_gap` | Stage 2 | Planning |
| `idle_prioritization` | Stage 3 | Planning |
| `task_synthesis` | Stage 4 | Planning |
| `anti_drift` | Stage 5 | Planning |
| `planning_state_update` | Stage 6 | Planning |
| `review` | — | Quality |
| `failure_recovery` | — | Recovery |

---

## Role Definitions

### bootstrap

**Category**: Lifecycle  
**Purpose**: Initialize a new project's planning state, set initial goals, and create the first workstreams from project context.

**Input context**:
- Project metadata (name, description)
- Existing tasks (if any)
- Existing memory entries

**Output format**:
```json
{
  "goals": ["string"],
  "release_focus": "string",
  "initial_workstreams": [
    {
      "id": "string",
      "title": "string",
      "description": "string",
      "why": "string",
      "what_next": "string"
    }
  ],
  "must_not_forget": ["string"],
  "planner_summary": "string"
}
```

**Behavioral constraints**:
- Must not create tasks directly — only seed planning state
- Must derive goals from existing project context, not invent aspirational ones
- Output must be valid JSON

---

### runtime_worker

**Category**: Lifecycle  
**Purpose**: Execute a claimed task. Not part of the planning pipeline — this is the standard task execution prompt.

**Input context**:
- Task details (title, instructions, type, priority)
- Task context (parent task, dependencies, related runs)
- Project memory
- Agent identity

**Output format**:
```json
{
  "result": "string — task output",
  "artifacts": [{"name": "string", "content": "string"}],
  "status": "succeeded | failed | needs_review",
  "notes": "string"
}
```

**Behavioral constraints**:
- Must stay within the scope of the assigned task
- Must not modify planning state
- Must report failures rather than silently continuing

---

### idle_recon

**Category**: Planning — Stage 0  
**Purpose**: Gather and summarize the current state of the project.

**Input context**:
- Planning state
- Recent tasks (created/updated since last cycle)
- Recent events
- Active agents
- Project memory

**Output format**: See [Pipeline Spec — Stage 0 Output](planning-pipeline.md#stage-0--recon)

**Behavioral constraints**:
- Read-only analysis — must not suggest actions or create tasks
- Summary should be factual and evidence-based
- Must flag notable state changes explicitly
- Must report environment flags accurately

---

### idle_continuity

**Category**: Planning — Stage 1  
**Purpose**: Identify which workstreams need continuation and score them.

**Input context**:
- Recon output
- Planning state
- All workstreams with their current status

**Output format**: See [Pipeline Spec — Stage 1 Output](planning-pipeline.md#stage-1--continuity-analysis)

**Behavioral constraints**:
- Must evaluate ALL active workstreams, not just recent ones
- Continuity scores must be justified with reasoning
- Must identify stalled workstreams (no progress for 2+ cycles)
- Must mark completed workstreams

---

### idle_gap

**Category**: Planning — Stage 2  
**Purpose**: Find missing work, unaddressed goals, and newly unblocked opportunities.

**Input context**:
- Recon output
- Continuity output
- Planning state including goals

**Output format**: See [Pipeline Spec — Stage 2 Output](planning-pipeline.md#stage-2--gap-analysis)

**Behavioral constraints**:
- Gaps must reference specific goals — do not invent new goals
- Severity must be assigned based on goal criticality
- Opportunities must specify WHY they are unblocked now
- Must not duplicate work already covered by active workstreams

---

### idle_prioritization

**Category**: Planning — Stage 3  
**Purpose**: Rank all candidates (continuity + gaps + opportunities) into a work plan.

**Input context**:
- Continuity output
- Gap output
- Planning state
- Current planning mode
- Anti-fragmentation rules

**Output format**: See [Pipeline Spec — Stage 3 Output](planning-pipeline.md#stage-3--prioritization)

**Behavioral constraints**:
- Must respect planning mode: focused mode limits to 1 workstream; recovery mode prioritizes failures
- Must respect anti-fragmentation rules (e.g., "max 2 active workstreams")
- Must select exactly one primary focus
- Deferred items must include a reason

---

### task_synthesis

**Category**: Planning — Stage 4  
**Purpose**: Generate concrete task proposals from the prioritized plan.

**Input context**:
- Prioritization output
- Planning state
- All existing tasks (to avoid duplicates)
- Must-not-forget threads

**Output format**: See [Pipeline Spec — Stage 4 Output](planning-pipeline.md#stage-4--task-synthesis)

**Behavioral constraints**:
- Tasks must be concrete and actionable (not vague/aspirational)
- Must check for duplicates against existing tasks
- Must assign each task to a workstream
- Must include a rationale for each task
- Must not exceed `max_tasks_per_cycle` configuration
- Must address at least one must-not-forget thread when applicable

---

### anti_drift

**Category**: Planning — Stage 5  
**Purpose**: Validate proposed tasks against goals and constraints. Prevent scope drift.

**Input context**:
- Proposed tasks from synthesis
- Planning state (goals, focus)
- Anti-fragmentation rules
- Recent decisions

**Output format**: See [Pipeline Spec — Stage 5 Output](planning-pipeline.md#stage-5--anti-drift-validation)

**Behavioral constraints**:
- Must evaluate EVERY proposed task — no skipping
- Rejections must include a specific reason
- Modifications should be minimal and justified
- Overall coherence score must reflect honest assessment
- Must flag if proposed tasks collectively drift from stated goals

---

### planning_state_update

**Category**: Planning — Stage 6  
**Purpose**: Update the planning state with the results of the completed cycle.

**Input context**:
- All stage outputs (recon through anti-drift)
- Actually created tasks
- Previous planning state

**Output format**: Updated PlanningState object

**Behavioral constraints**:
- Must preserve must-not-forget threads unless explicitly resolved
- Must update workstream scores based on cycle results
- Must increment cycle count
- Must update recon and planner summaries
- Must update priority order to reflect current ranking
- Must not drop goals unless they are explicitly completed

---

### review

**Category**: Quality  
**Purpose**: Review completed work for quality, correctness, and alignment with task requirements.

**Input context**:
- Task details
- Run output and artifacts
- Project goals
- Quality standards

**Output format**:
```json
{
  "verdict": "approved | needs_changes | rejected",
  "feedback": "string",
  "issues": [{"severity": "string", "description": "string"}],
  "quality_score": "float 0-1"
}
```

**Behavioral constraints**:
- Must evaluate against the specific task requirements, not general standards
- Must be constructive — provide actionable feedback for issues
- Must not modify code or artifacts directly

---

### failure_recovery

**Category**: Recovery  
**Purpose**: Analyze a failed task or run and determine corrective action.

**Input context**:
- Failed task/run details
- Error logs
- Previous attempts (if retry)
- Related tasks and context

**Output format**:
```json
{
  "root_cause": "string",
  "recommended_action": "retry | reassign | escalate | skip | modify_and_retry",
  "modifications": "string | null — changes to make before retry",
  "new_task_needed": "bool",
  "new_task_description": "string | null"
}
```

**Behavioral constraints**:
- Must analyze actual errors, not speculate
- Must not retry indefinitely — recommend escalation after 2 failures
- Must consider whether the task itself is flawed vs. execution failure
