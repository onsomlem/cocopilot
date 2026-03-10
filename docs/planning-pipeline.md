# Staged Idle-Planning Pipeline Specification

**Version**: 1.0  
**Status**: Active  
**Last updated**: 2025-01-01

## Overview

This document defines the canonical multi-stage planning pipeline for autonomous idle planning in Cocopilot. It replaces the previous single-pass approach with an explicit, ordered sequence of analysis and generation stages.

The pipeline runs when no actionable work exists and the planner activates autonomously.

## Pipeline Stages

```
┌─────────┐   ┌──────────────┐   ┌──────────────┐   ┌────────────────┐
│  Recon  │──▶│ Continuity   │──▶│ Gap Analysis │──▶│ Prioritization │
│ Stage 0 │   │   Stage 1    │   │   Stage 2    │   │    Stage 3     │
└─────────┘   └──────────────┘   └──────────────┘   └────────────────┘
                                                            │
                  ┌──────────────────┐   ┌──────────────┐   │
                  │ Planning-State   │◀──│  Anti-Drift  │◀──┘
                  │   Update (6)     │   │ Validation(5)│   
                  └──────────────────┘   └──────────────┘   
                                                ▲
                                                │
                                         ┌──────────────┐
                                         │    Task      │
                                         │ Synthesis(4) │
                                         └──────────────┘
```

---

## Stage 0 — Recon

**Purpose**: Gather and summarize the current state of the project, tasks, agents, and environment.

**Prompt role**: `recon_prompt`

### Inputs

| Field | Type | Description |
|-------|------|-------------|
| project_id | string | Target project |
| planning_state | PlanningState | Current persisted planning state (may be empty on first run) |
| recent_tasks | []Task | Tasks created/updated in the last cycle |
| recent_events | []Event | Events since last planning run |
| active_agents | []Agent | Currently connected agents |
| project_memory | []Memory | Project memory entries |

### Output Schema

```json
{
  "summary": "string — high-level project status sentence",
  "active_task_count": "int",
  "queued_task_count": "int",
  "failed_task_count": "int",
  "blocked_task_count": "int",
  "agent_count": "int",
  "notable_changes": ["string — important changes since last recon"],
  "environment_flags": {
    "has_failures": "bool",
    "has_blocked": "bool",
    "has_idle_agents": "bool",
    "stale_workstreams_detected": "bool"
  }
}
```

### Failure behavior
If recon fails, the pipeline halts. No tasks are generated. The failure is logged as an event.

### Fallback behavior
If the recon prompt times out or produces unparseable output, use the previous recon summary from planning state and proceed with a staleness warning flag.

---

## Stage 1 — Continuity Analysis

**Purpose**: Identify in-progress workstreams and assess which ones need continuation.

**Prompt role**: `continuity_prompt`

### Inputs

| Field | Type | Description |
|-------|------|-------------|
| recon_output | ReconOutput | Output from Stage 0 |
| planning_state | PlanningState | Current planning state |
| workstreams | []Workstream | All known workstreams |

### Output Schema

```json
{
  "continuation_candidates": [
    {
      "workstream_id": "string",
      "continuity_score": "float 0-1",
      "reason": "string — why this should continue",
      "suggested_next_action": "string",
      "risk_if_abandoned": "string"
    }
  ],
  "completed_workstreams": ["string — workstream IDs that appear done"],
  "stalled_workstreams": ["string — workstream IDs with no progress"]
}
```

### Failure behavior
If continuity analysis fails, proceed to gap analysis with an empty continuation candidates list. Log the failure.

### Fallback behavior
If the prompt returns partial results, use whatever candidates were parsed. Set a `partial_continuity` flag for downstream stages.

---

## Stage 2 — Gap Analysis

**Purpose**: Identify missing work, unaddressed goals, and unblocked opportunities.

**Prompt role**: `gap_prompt`

### Inputs

| Field | Type | Description |
|-------|------|-------------|
| recon_output | ReconOutput | Output from Stage 0 |
| continuity_output | ContinuityOutput | Output from Stage 1 |
| planning_state | PlanningState | Current planning state |
| project_goals | []string | Stated project goals |

### Output Schema

```json
{
  "identified_gaps": [
    {
      "description": "string",
      "severity": "string — critical | important | minor",
      "related_goal": "string — which project goal this gap blocks",
      "suggested_workstream": "string — new or existing workstream ID"
    }
  ],
  "unblocked_opportunities": [
    {
      "description": "string",
      "reason_unblocked": "string",
      "estimated_effort": "string — small | medium | large"
    }
  ]
}
```

### Failure behavior
If gap analysis fails, proceed to prioritization using only continuity candidates. Log the failure.

### Fallback behavior
If partial output is returned, use parsed gaps. New workstream creation is skipped if gap analysis produced no valid output.

---

## Stage 3 — Prioritization

**Purpose**: Rank work candidates across continuity and gap outputs to produce a prioritized work plan.

**Prompt role**: `prioritization_prompt`

### Inputs

| Field | Type | Description |
|-------|------|-------------|
| continuity_output | ContinuityOutput | Output from Stage 1 |
| gap_output | GapOutput | Output from Stage 2 |
| planning_state | PlanningState | Current planning state |
| planning_mode | string | Current planning mode (standard / focused / recovery / maintenance) |
| anti_fragmentation_rules | []Rule | Structural rules constraining prioritization |

### Output Schema

```json
{
  "ranked_items": [
    {
      "rank": "int",
      "source": "string — continuity | gap | opportunity",
      "workstream_id": "string | null",
      "description": "string",
      "priority_score": "float 0-1",
      "reasoning": "string"
    }
  ],
  "selected_focus": "string — the primary workstream or goal for this cycle",
  "deferred_items": [
    {
      "description": "string",
      "defer_reason": "string"
    }
  ]
}
```

### Failure behavior
If prioritization fails, fall back to continuing the highest-scored workstream from the last cycle. Log the failure.

### Fallback behavior
Use the previous cycle's priority ordering with a `stale_priorities` flag.

---

## Stage 4 — Task Synthesis

**Purpose**: Generate concrete task proposals from the prioritized work plan.

**Prompt role**: `task_synthesis_prompt`

### Inputs

| Field | Type | Description |
|-------|------|-------------|
| prioritization_output | PrioritizationOutput | Output from Stage 3 |
| planning_state | PlanningState | Current planning state |
| existing_tasks | []Task | All current tasks (to avoid duplicates) |
| must_not_forget | []string | Threads that must not be dropped |

### Output Schema

```json
{
  "proposed_tasks": [
    {
      "title": "string",
      "instructions": "string",
      "type": "string",
      "priority": "int 0-100",
      "parent_task_id": "int | null",
      "tags": ["string"],
      "workstream_id": "string",
      "rationale": "string — why this task is needed now",
      "dependencies": ["int — task IDs"]
    }
  ],
  "skipped_items": [
    {
      "description": "string",
      "reason": "string — why no task was generated"
    }
  ]
}
```

### Failure behavior
If task synthesis fails, no tasks are created. The pipeline still proceeds to anti-drift validation and planning-state update. Log the failure.

### Fallback behavior
If partial output is returned, create only the tasks that parsed successfully. Flag partial synthesis in the planning state.

---

## Stage 5 — Anti-Drift Validation

**Purpose**: Review proposed tasks against project goals, recent history, and anti-fragmentation rules to prevent drift.

**Prompt role**: `anti_drift_prompt`

### Inputs

| Field | Type | Description |
|-------|------|-------------|
| proposed_tasks | []TaskProposal | Output from Stage 4 |
| planning_state | PlanningState | Current planning state |
| project_goals | []string | Stated project goals |
| anti_fragmentation_rules | []Rule | Structural constraints |
| recent_decisions | []Decision | Recent planner decisions for context |

### Output Schema

```json
{
  "approved_tasks": [
    {
      "task_index": "int — index in proposed_tasks",
      "approved": "bool",
      "modification": "string | null — suggested change",
      "rejection_reason": "string | null"
    }
  ],
  "drift_warnings": [
    {
      "description": "string",
      "severity": "string — low | medium | high"
    }
  ],
  "overall_coherence_score": "float 0-1"
}
```

### Failure behavior
If anti-drift validation fails, default to approving all proposed tasks (fail-open for productivity). Log the validation failure prominently.

### Fallback behavior
Skip validation and approve all tasks. Set `drift_check_skipped: true` in planning state.

---

## Stage 6 — Planning-State Update

**Purpose**: Persist the results of the planning cycle into the planning state for future continuity.

**Prompt role**: `planning_state_update_prompt`

### Inputs

| Field | Type | Description |
|-------|------|-------------|
| recon_output | ReconOutput | From Stage 0 |
| continuity_output | ContinuityOutput | From Stage 1 |
| gap_output | GapOutput | From Stage 2 |
| prioritization_output | PrioritizationOutput | From Stage 3 |
| synthesis_output | SynthesisOutput | From Stage 4 |
| anti_drift_output | AntiDriftOutput | From Stage 5 |
| approved_tasks | []Task | Actually created tasks |
| planning_state | PlanningState | Previous state |

### Output Schema

The output is the updated `PlanningState` object (see planning-state schema).

### Failure behavior
If the update fails, the old planning state is preserved. The failure is logged but does not block the created tasks.

### Fallback behavior
Perform a minimal mechanical update: increment cycle count, update timestamps, record created task IDs. Skip LLM-driven state summarization.

---

## Pipeline Execution Rules

1. Stages execute sequentially in order 0 → 6.
2. Each stage must complete (or fail) before the next begins.
3. Stage failures do NOT halt the pipeline unless specified (only Stage 0 is a hard halt).
4. All stage outputs are persisted as part of the planning cycle record.
5. The pipeline produces a `PlanningCycleResult` at the end containing all stage outputs.
6. Only approved tasks from Stage 5 are actually created.
7. The pipeline is idempotent for the same cycle — re-running produces the same decisions given the same state.

## Pipeline Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `max_tasks_per_cycle` | 3 | Maximum tasks to create per planning cycle |
| `min_cycle_interval` | 5m | Minimum time between planning cycles |
| `continuity_threshold` | 0.3 | Minimum continuity score to consider a workstream active |
| `coherence_threshold` | 0.5 | Minimum coherence score from anti-drift to proceed |
| `planning_mode` | standard | Current planning mode |
| `enable_anti_drift` | true | Whether anti-drift validation runs |

## Analysis vs Generation Stages

| Stage | Type | Modifies State? | Creates Tasks? |
|-------|------|-----------------|----------------|
| 0 — Recon | Analysis | No | No |
| 1 — Continuity | Analysis | No | No |
| 2 — Gap Analysis | Analysis | No | No |
| 3 — Prioritization | Analysis | No | No |
| 4 — Task Synthesis | Generation | No | Yes (proposes) |
| 5 — Anti-Drift | Validation | No | No (filters) |
| 6 — State Update | Mutation | Yes | No |
