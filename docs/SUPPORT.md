# Support Policy

## Stability Target

`msgcat` aims to provide stable behavior for backend/API i18n message resolution and wrapping.

## Compatibility

- Public API changes follow SemVer.
- Breaking changes are introduced only in major releases.
- Migration notes are published in `docs/MIGRATION.md`.

## Production Contract

- Core request-path methods are concurrency-safe.
- Observer callback failures are isolated from request path.
- Stats cardinality is bounded when `StatsMaxKeys` is configured.

## Reporting Issues

Please include:
- Go version and OS.
- `msgcat` version/tag.
- YAML samples (minimal reproducible).
- `Config` used.
- Steps to reproduce.
