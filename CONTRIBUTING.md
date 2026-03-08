# Contributing to Cocopilot

## Development Setup

```bash
# Clone and build
git clone https://github.com/onsomlem/cocopilot.git
cd cocopilot
go build -o cocopilot ./cmd/cocopilot

# Run the server
./cocopilot

# Run tests
go test -race -timeout 180s ./...
```

**Requirements:** Go 1.21+, Node 20+ (for VSIX/MCP tools only)

## Running Tests

```bash
# All tests
go test ./...

# With race detection
go test -race ./...

# Specific test
go test -v -run TestClaimTaskByID_Success

# With coverage
go test -cover ./...
```

## Project Structure

| File/Dir | Purpose |
|----------|---------|
| `cmd/cocopilot/main.go` | Thin entry point (imports `server` package) |
| `server/main.go` | Server startup, DB init, background goroutines |
| `server/routes.go` | HTTP route registration |
| `server/handlers_v2_*.go` | v2 API handlers |
| `server/handlers_v1.go` | v1 legacy handlers |
| `server/models_v2.go` | Data models |
| `server/db_v2.go` | Database operations |
| `server/automation.go` | Automation rules engine |
| `server/assignment.go` | Task claiming service |
| `server/finalization.go` | Task completion service |
| `internal/` | Internal packages (models, dbstore, config, etc.) |
| `tools/` | MCP server + VS Code extension |
| `migrations/` | SQLite schema migrations |

## Pull Request Guidelines

1. **One concern per PR** — don't mix features with refactors
2. **Tests required** — new features need tests, bug fixes need regression tests
3. **No behavior changes in refactors** — refactor PRs must not change behavior
4. **Run before submitting:**
   ```bash
   go build ./...
   go test -race -timeout 180s ./...
   go vet ./...
   ```
5. **Commit messages** — use imperative mood ("Add feature" not "Added feature")

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Use `writeV2Error()` for v2 API error responses
- Use `nowISO()` for timestamps
- Use `sql.NullString`/`sql.NullInt64` for nullable DB columns
- Keep handlers focused — delegate to service functions

## Reporting Issues

- Use the issue templates in `.github/ISSUE_TEMPLATE/`
- Include reproduction steps for bugs
- Include expected vs actual behavior
