# Release Playbook

## Versioning

- SemVer is used.
- `MAJOR`: breaking public API or behavior contract.
- `MINOR`: backward-compatible features.
- `PATCH`: backward-compatible fixes/docs/perf.

## Support Policy

- Active support: latest minor in current major.
- Security/critical fixes: latest two minors in current major (when practical).

## Pre-release checklist

1. Update `docs/CHANGELOG.md` (`[Unreleased]` -> target version/date).
2. Update `docs/MIGRATION.md` if behavior changed.
3. Run:
   - `go test ./...`
   - `go test -race ./...`
   - `go vet ./...`
   - `go test -run ^$ -bench . -benchmem ./...`
4. Validate docs links and examples compile.
5. Create release commit.

## Tagging

```bash
git tag -a v1.0.0 -m "v1.0.0"
git push origin v1.0.0
```

## GitHub release notes template

- Summary: What changed and why.
- Breaking changes: explicit list, if any.
- Migration notes: link `docs/MIGRATION.md`.
- Performance notes: benchmark deltas.
- Operational notes: observer/stats/reload caveats.

### Example (v1.0.8)

```markdown
## Summary
Production hardening: async observer, bounded stats, reload retries, CI, SECURITY.md, Go 1.26, and lint config.

## Breaking changes
None. Go 1.26+ required (see migration guide).

## Migration
[Migration guide](docs/MIGRATION.md) — sections 3–6 and 8 (Go version).

## Operational
Observer is async and panic-safe; call `msgcat.Close(catalog)` on shutdown if using an observer.
```

## Post-release checks

- Validate tag visible in remote.
- Validate docs render correctly.
- Sanity test a downstream sample app.
