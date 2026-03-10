# Definition of Done

A change is **not done** unless every item below is satisfied.

## Checklist

### 1. Tests Pass

- [ ] `go test -race ./...` passes with zero failures.
- [ ] New behaviour has at least one corresponding test.
- [ ] Golden-path test (`TestGoldenPath_HTTPFullLifecycle`) still passes.

### 2. Smoke Flow Passes

- [ ] `make test-smoke` — all UI pages, V1, and V2 routes return expected status codes.

### 3. Release Packaging Passes

- [ ] `make release` produces a clean zip.
- [ ] `make verify-release` finds no forbidden artifacts (.git, .db, node_modules, .vsix, __MACOSX, .DS_Store, coverage.out, *.exe~).

### 4. Repository Cleanliness

- [ ] `make verify-repo` finds no forbidden tracked files (.vsix, node_modules, compiled binaries, .db files).
- [ ] No unintended files are staged or committed.

### 5. Lint

- [ ] `go vet ./...` reports zero issues.

### 6. Documentation

- [ ] Docs affected by the change are updated (API docs, quickstart, migration notes, etc.).
- [ ] New environment variables are documented in AGENTS.md and README.md.
- [ ] New migration files follow the naming convention (`NNNN_description.sql`).

### 7. No Forbidden Artifacts

- [ ] No compiled binaries, database files, IDE config, or OS metadata are committed.
- [ ] `.gitignore` covers any new generated files.

## Quick Verification

Run the full release gate to verify all of the above in one command:

```bash
make gate
```

This runs, in order:

1. `verify-repo` — repository cleanliness
2. `lint` — `go vet`
3. `build` — compilation
4. `test -race` — all tests with race detector
5. `release` + `verify-release` — packaging validation

All five steps must pass for the gate to succeed.
