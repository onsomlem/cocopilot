# Release Checklist

## Release Candidate Validation

Run this against a **fresh clone** using only the README. Every step must pass.

### Build & Launch

- [ ] `git clone` the repo
- [ ] `go build -o cocopilot ./cmd/cocopilot` succeeds
- [ ] `./cocopilot` starts without errors
- [ ] Browser opens to dashboard at `http://127.0.0.1:8080`

### Dashboard First Impression

- [ ] Dashboard loads — Kanban board is visible
- [ ] Default project exists (`proj_default`)
- [ ] "Seed Demo" button is visible and works
- [ ] After seeding: tasks, agents, and runs visible in UI

### Task Lifecycle (via UI or API)

- [ ] Create a task — appears on board in Pending column
- [ ] Task shows correct title, type, priority
- [ ] Claim task (via agent or built-in worker) — moves to In Progress
- [ ] Agent appears in Agents panel with heartbeat
- [ ] Run appears in Runs view
- [ ] Complete task — moves to Completed column
- [ ] Events feed shows: created → claimed → completed

### Real-Time Updates

- [ ] SSE events fire — status changes appear without page refresh
- [ ] Agent heartbeat changes over time in UI
- [ ] Board reflects task state changes immediately

### Cross-View Consistency

- [ ] Board, Runs, Agents, and Events views agree on task state
- [ ] Task detail shows correct run history
- [ ] Event timestamps are consistent and in order

### Release Packaging

- [ ] `make verify-repo` passes (no binaries, .vsix, node_modules, .db in git index)
- [ ] `make release` succeeds
- [ ] `make verify-release` passes (no .git, .db, .DS_Store, __MACOSX, node_modules, .vsix)
- [ ] Release zip can be unpacked and binary runs cleanly

### CI

- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] CI workflow passes on push to main

## Blocker Classification

After running the checklist, classify every issue:

| Bucket | Criteria |
|--------|----------|
| **Ship blocker** | Breaks the core workflow, crashes, data corruption, security issue |
| **Polish** | Cosmetic, UX friction, missing nice-to-have, documentation gap |

No middle bucket. Either it blocks shipping or it doesn't.

## Pre-Release

- [ ] All tests pass: `go test -race -timeout 180s ./...`
- [ ] Build succeeds on all targets: `make build-all`
- [ ] No lint errors: `make lint`
- [ ] VERSION file updated
- [ ] CHANGELOG.md updated

## Release Steps

1. Update VERSION: `echo "X.Y.Z" > VERSION`
2. Tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`
3. Push: `git push origin main && git push origin vX.Y.Z`
4. CI builds and publishes artifacts (triggered by tag)
5. Create GitHub release with changelog and binary artifacts

## Post-Release

- [ ] Verify binaries download and run correctly
- [ ] Merge release branch back to main (if used)
