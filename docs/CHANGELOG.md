# Changelog

All notable changes to this project are documented in this file.

This project follows Semantic Versioning.

## [Unreleased]

### Added
- **CLDR plural forms:** optional `short_forms` / `long_forms` on `RawMessage` (keys: zero, one, two, few, many, other) and `plural_param` (default `count`). `internal/plural` selects form by language and count. Binary `{{plural:count|singular|plural}}` unchanged.
- **MessageDef:** type for defining messages in Go (Key, Short, Long, ShortForms, LongForms, PluralParam, Code). **msgcat extract -source** finds MessageDef struct literals and merges their content into source YAML.
- **Optional group:** `Messages.Group` and `OptionalGroup` (int or string in YAML, e.g. `group: 0` or `group: "api"`). CLI extract/merge preserve group.
- CLI **extract** (keys from GetMessageWithCtx/WrapErrorWithCtx/GetErrorWithCtx; sync to YAML with MessageDef merge) and **merge** (translate.\<lang\>.yaml with group and plural fields copied).
- Examples: `cldr_plural`, `msgdef`. Docs: CLI_WORKFLOW_PLAN, CLDR_AND_GO_MESSAGES_PLAN.
- String message keys (e.g. `"greeting.hello"`) instead of numeric codes for lookup.
- Named template parameters: `{{name}}`, `{{plural:count|...}}`, `{{num:amount}}`, `{{date:when}}` with `msgcat.Params`.
- Optional string `code` field: any value (e.g. `"404"`, `"ERR_NOT_FOUND"`); not unique. YAML accepts `code: 404` or `code: "ERR_001"`. Helpers `CodeInt()`, `CodeString()`.
- `Message.Key` and `ErrorKey()` for API identifier when code is empty.
- Runnable examples: `basic`, `load_messages`, `reload`, `strict`, `stats`; HTTP and metrics examples get message resources.
- Documentation: "Message and error codes" section; API examples (README, CONTEXT7); CONVERSION_PLAN final state; MIGRATION section 9 for string keys.

### Changed (breaking)
- `GetMessageWithCtx(ctx, msgKey string, params Params)`; `WrapErrorWithCtx` / `GetErrorWithCtx` take `msgKey string` and `params Params`. Params can be nil.
- `Message.Code` and `ErrorCode()` are `string` (empty when not set). Use `Key` / `ErrorKey()` when empty.
- `LoadMessages`: each `RawMessage` must have `Key` with prefix `sys.`; no numeric code range. Use `Code: msgcat.CodeInt(503)` or `msgcat.CodeString("ERR_X")`.
- Observer: `OnMessageMissing(lang, msgKey string)`, `OnTemplateIssue(lang, msgKey string, issue string)`.
- YAML: `set` uses string keys; optional `code` per entry; `group` removed. Template placeholders are named only.
- Constants `CodeMissingMessage` and `CodeMissingLanguage` are strings (`"msgcat.missing_message"`, `"msgcat.missing_language"`).

## [1.0.8] - 2026-02-25

### Added
- Async, panic-safe observer pipeline with bounded queue (`ObserverBuffer`).
- Bounded stats cardinality (`StatsMaxKeys`) with `__overflow__` bucket.
- Reload retries (`ReloadRetries`, `ReloadRetryDelay`).
- Extended stats fields: `DroppedEvents`, `LastReloadAt`.
- Helper functions: `ResetStats(catalog)`, `Close(catalog)`.
- Production-oriented docs for Context7 + retrieval-friendly docs.
- Additional tests: observer behavior, stats capping/reset, reload retry behavior.
- Benchmarks and fuzz test entrypoints.
- GitHub Actions CI workflow (test, race, vet, examples build).
- `SECURITY.md` for vulnerability reporting.
- `.golangci.yml` for lint configuration (replaces misspelled `.golanci.yml`).

### Changed
- `Reload` now supports transient read/parse retries.
- Observer callbacks are no longer executed inline on request path.
- Stats now enforce key caps to avoid unbounded memory growth.
- Go module requires Go 1.26.
- Replaced deprecated `io/ioutil` with `os` (`ReadDir`, `ReadFile`).

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
