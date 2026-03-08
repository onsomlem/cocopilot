# Release Checklist

## Pre-Release

- [ ] All tests pass: `go test -race -timeout 180s ./...`
- [ ] Build succeeds on all targets: `make build-all`
- [ ] No lint errors: `make lint`
- [ ] VERSION file updated to new version
- [ ] CHANGELOG.md updated with release notes
- [ ] Documentation reviewed and up to date
- [ ] Generated files are current: `go run tools/codegen/codegen.go && git diff --exit-code tools/shared/`

## Release Steps

1. **Create release branch** (for major/minor releases):
   ```bash
   git checkout -b release/vX.Y.Z
   ```

2. **Update VERSION**:
   ```bash
   echo "X.Y.Z" > VERSION
   git add VERSION
   git commit -m "Bump version to X.Y.Z"
   ```

3. **Tag the release**:
   ```bash
   git tag -a vX.Y.Z -m "Release vX.Y.Z"
   git push origin vX.Y.Z
   ```

4. **CI builds and publishes artifacts** (triggered by tag push):
   - Cross-compiled binaries (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64)
   - SHA256 checksums
   - VSIX extension package

5. **Create GitHub release**:
   - Use the tag
   - Paste changelog entries
   - Attach binary artifacts and checksums

## Post-Release

- [ ] Verify binaries download and run correctly
- [ ] Update Homebrew tap formula (if applicable)
- [ ] Announce release
- [ ] Merge release branch back to main (if used)

## Hotfix Process

For critical patches on a released version:

1. Branch from the release tag: `git checkout -b hotfix/vX.Y.Z+1 vX.Y.Z`
2. Apply fix, update VERSION to patch increment
3. Tag and push: `git tag -a vX.Y.Z+1 -m "Hotfix vX.Y.Z+1"`
4. Cherry-pick fix back to main
