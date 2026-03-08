Cocopilot — Living Build Plan (Auto UX + Canonical Runtime)

Purpose: This is the single living checklist/spec of everything still required to reach the target system: automatic operation + simple UX.
Rule: When we ship/fix something, we move it to ✅ Done and remove/merge duplicates. When we discover new needed work, we add it.

Current Baseline (what we assume is already done)

Based on your “10 commits” summary:
	•	✅ FailTask v1 status bug fixed (StatusFailed + tests)
	•	✅ v1 /task and v2 claim-by-id already route to ClaimTaskByID
	•	✅ v2 claim-next now routes to ClaimTaskByID (removed duplicate logic)
	•	✅ AssignmentEnvelope now includes Project + CompletionContract
	•	✅ Context includes last 5 runs (as RecentRuns)
	•	✅ Memory retrieval is filtered (type + failures + keyword; capped)
	•	✅ repo.changed emitted after completion when files touched
	•	✅ repo.changed automation stores changed files and injects into context
	•	✅ task.failed triggers follow-up “Investigate failure: {title}”

If any of the above is wrong, we’ll correct it next update.

⸻

North Star Definition (what “done” looks like)

One-sentence UX

User adds a task → agent executes automatically → system learns + proposes next tasks → dashboard shows exactly what happened and what to do next.

Non‑negotiable invariants
	•	One canonical lifecycle for all clients (VSIX/MCP/CLI/agents): claim → lease+run → envelope → finalize → events → automation.
	•	One canonical envelope returned on every claim, with ranked context + completion contract.
	•	Repo → context → memory → automation is a default loop (not optional).
	•	No drift between server, MCP tools, VSIX client (contract generated + CI check).
	•	Safe-by-default (localhost bind, auth optional but consistent, workdir guardrails, secret hygiene).

⸻

Verification Snapshot (where we REALLY are)

Updated using the latest cocopilot.zip you uploaded.

✅ Verified implemented (0–8)
	•	CI workflow exists (.github/workflows/ci.yml) with Go + MCP + VSIX jobs, artifact guard, and contract-drift job.
	•	VSIX core task flow is v2-first; updateStatus targets /api/v2/tasks/:id (no /update-status usage).
	•	AssignmentEnvelope includes PolicySnapshot and backend supports ClaimNext.
	•	Repo automation pipeline is wired: repo.changed → repo.scanned → context.invalidated.
	•	Dashboard projection exists and includes recommendations.
	•	Multi-project UI is supported; proj_default hardcoding removed from pages.
	•	Worker backoff/failure guards exist; tests + build pass.

⚠️ What’s still worth doing (not “broken,” but missing for true maximum automation)
	•	Repo.scanned “real work”: add filesystem scan/sync + diff summary + richer repo metadata.
	•	Context quality vNext: ranking/budgeting across all context categories (memories, repo changes, run summaries, artifacts), not just repo files.
	•	Contract integrity vNext: drift CI is heuristic; add generated TS client/types and validate generated outputs.
	•	One-click experience: quickstart + notifications + default worker for “it just runs.”

⸻

Work Plan (ALL future steps)

(ALL future steps)
(ALL future steps)
(ALL future steps)
(ALL future steps)
(ALL future steps)

0–8) ✅ Implemented (Archived)

Sections 0–8 have been verified implemented in the latest code drop. They are archived to keep this canvas focused on what’s left.

If something regresses in a future upload, we’ll un-archive the relevant checklist.

⸻

9) One‑Trick Pony Expansion (make it feel “automatic” and “shiny”) (make it feel “automatic” and “shiny”)

Goal: reduce friction to near-zero and add a few high-signal ‘wow’ capabilities that compound usefulness.

9.1 One‑click Quickstart
	•	cocopilot quickstart (CLI) or VSIX command: start server + open dashboard + start default worker
	•	Health check + self-diagnosis output (ports, DB migrations, auth, workdir validity)

9.2 Built‑in default worker (optional)
	•	Provide a reference worker that:
	•	polls claim-next
	•	executes a pluggable “executor” (human-in-the-loop, script runner, or external LLM adapter)
	•	posts structured finalization payload
	•	Safety: rate limits + loop guards + max concurrency per project

9.3 Notifications that matter
	•	Notification sinks: VSIX notifications + webhook endpoint (generic)
	•	Notify on: failures, stalled runs, policy denials, project idle, new recommendations

9.4 Human-in-the-loop approvals
	•	Optional approval gate for high-impact tasks (e.g., repo writes, policy changes)
	•	“Approve/Reject” UX in dashboard and VSIX

9.5 Task templates + ‘smart create’
	•	Template library (per project): bugfix, refactor, test, review, release, index repo
	•	Smart task creation that auto-fills completion contract + initial context hints

9.6 Export/import + backups
	•	Export project state (tasks/runs/events/memory/repo metadata) to a single archive
	•	Import to restore a project (with id remapping)

9.7 Multi-agent routing (later)
	•	Agent capability profiles and task-type routing
	•	Basic scoring: which agent succeeds on what