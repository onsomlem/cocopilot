# Versioning Policy

## API Versioning

- **`/api/v2`** is the stable API. Breaking changes require a major version bump.
- The OpenAPI spec (`docs/api/openapi-v2.yaml`) is the canonical contract.
- Clients (MCP tools, VSIX extension) are generated from or validated against the OpenAPI spec.

## Semantic Versioning

Releases follow [semver](https://semver.org/): `vMAJOR.MINOR.PATCH`

- **MAJOR** — Breaking API changes (removed endpoints, changed response shapes, renamed fields)
- **MINOR** — New endpoints, new optional fields, new features (backward-compatible)
- **PATCH** — Bug fixes, performance improvements, documentation updates

## What Counts as a Breaking Change

| Breaking | Not Breaking |
|----------|-------------|
| Removing an endpoint | Adding a new endpoint |
| Removing a response field | Adding a new optional response field |
| Changing a field's type | Adding a new query parameter |
| Changing error code semantics | Adding a new error code |
| Requiring a previously optional field | Making a required field optional |

## Release Tags

- Tags: `v2.0.0`, `v2.1.0`, `v2.1.1`, etc.
- Pre-release: `v2.1.0-rc.1`
- The `VERSION` file at repo root tracks the current version.

## v1 API

The v1 API (`/task`, `/create`, `/save`, `/events`) is legacy and frozen. No new features will be added. It will be maintained for backward compatibility but may be removed in v3.
