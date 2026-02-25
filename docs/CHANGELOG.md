# Changelog

All notable changes to this project are documented in this file.

This project follows Semantic Versioning.

## [Unreleased]

### Added
- Async, panic-safe observer pipeline with bounded queue (`ObserverBuffer`).
- Bounded stats cardinality (`StatsMaxKeys`) with `__overflow__` bucket.
- Reload retries (`ReloadRetries`, `ReloadRetryDelay`).
- Extended stats fields: `DroppedEvents`, `LastReloadAt`.
- Helper functions: `ResetStats(catalog)`, `Close(catalog)`.
- Production-oriented docs for Context7 + retrieval-friendly docs.
- Additional tests: observer behavior, stats capping/reset, reload retry behavior.
- Benchmarks and fuzz test entrypoints.

### Changed
- `Reload` now supports transient read/parse retries.
- Observer callbacks are no longer executed inline on request path.
- Stats now enforce key caps to avoid unbounded memory growth.

### Fixed
- Updated docs links to `docs/` layout.

## [1.0.0] - 2026-02-25

### Added
- YAML-based message catalog loading by language.
- Context-based language resolution with fallback chain.
- Runtime message loading for system codes (`9000-9999`).
- Template support: positional, plural, localized number/date.
- Error wrapping with short/long localized messages.
- Concurrency-safe read/write behavior.
- Runtime reload and stats snapshot helpers.

### Security/Operational
- Race-safe implementation validated with `go test -race ./...`.
- Validations for YAML structure and message code constraints.
