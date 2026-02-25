# msgcat

`msgcat` is a lightweight i18n message catalog for Go focused on APIs and error handling.

It loads messages from YAML by language, resolves language from `context.Context`, supports runtime message loading for system codes, and can wrap domain errors with localized short/long messages.

Maturity: production-ready (`v1.x`) with SemVer and release/migration docs in `docs/`.

## Installation

```bash
go get github.com/loopcontext/msgcat
```

## Quick Start

### 1. Create message files

Default path:

```text
./resources/messages
```

Example `en.yaml`:

```yaml
group: 0
default:
  short: Unexpected error
  long: Unexpected message code [{{0}}] was received and was not found in this catalog
set:
  1:
    short: User created
    long: User {{0}} was created successfully
  2:
    short: You have {{0}} {{plural:0|item|items}}
    long: Total: {{num:1}} generated at {{date:2}}
```

Example `es.yaml`:

```yaml
group: 0
default:
  short: Error inesperado
  long: Se recibió un código de mensaje inesperado [{{0}}] y no se encontró en el catálogo
set:
  1:
    short: Usuario creado
    long: Usuario {{0}} fue creado correctamente
  2:
    short: Tienes {{0}} {{plural:0|elemento|elementos}}
    long: Total: {{num:1}} generado el {{date:2}}
```

### 2. Initialize catalog

```go
catalog, err := msgcat.NewMessageCatalog(msgcat.Config{
  ResourcePath:      "./resources/messages",
  CtxLanguageKey:    "language",
  DefaultLanguage:   "en",
  FallbackLanguages: []string{"es"},
  StrictTemplates:   true,
  ObserverBuffer:    1024,
  StatsMaxKeys:      512,
  ReloadRetries:     2,
  ReloadRetryDelay:  50 * time.Millisecond,
})
if err != nil {
  panic(err)
}
```

### 3. Resolve messages/errors from context

```go
ctx := context.WithValue(context.Background(), "language", "es-AR")

msg := catalog.GetMessageWithCtx(ctx, 1, "juan")
fmt.Println(msg.ShortText) // "Usuario creado"

err := catalog.WrapErrorWithCtx(ctx, errors.New("db timeout"), 2, 3, 12345.5, time.Now())
fmt.Println(err.Error()) // localized short message
```

## Features

- Language resolution from context (typed key and string key compatibility).
- Language fallback chain: requested -> base (`es-ar` -> `es`) -> configured fallbacks -> default -> `en`.
- YAML + runtime-loaded system messages (`9000-9999`).
- Template tokens:
  - `{{0}}`, `{{1}}`, ... positional
  - `{{plural:i|singular|plural}}`
  - `{{num:i}}` localized number format
  - `{{date:i}}` localized date format
- Strict template mode (`StrictTemplates`) for missing parameters.
- Error wrapping with localized short/long messages and error code.
- Concurrency-safe reads/writes.
- Runtime reload (`msgcat.Reload`) preserving runtime-loaded messages.
- Observability hooks and counters (`SnapshotStats`).

## API

### Core interface

- `LoadMessages(lang string, messages []RawMessage) error`
- `GetMessageWithCtx(ctx context.Context, msgCode int, msgParams ...interface{}) *Message`
- `WrapErrorWithCtx(ctx context.Context, err error, msgCode int, msgParams ...interface{}) error`
- `GetErrorWithCtx(ctx context.Context, msgCode int, msgParams ...interface{}) error`

### Helpers

- `msgcat.Reload(catalog MessageCatalog) error`
- `msgcat.SnapshotStats(catalog MessageCatalog) (MessageCatalogStats, error)`
- `msgcat.ResetStats(catalog MessageCatalog) error`
- `msgcat.Close(catalog MessageCatalog) error`

### Constants

- `SystemMessageMinCode = 9000`
- `SystemMessageMaxCode = 9999`
- `CodeMissingMessage = 999999998`
- `CodeMissingLanguage = 99999999`

## Observability

Provide an observer in config:

```go
type Observer struct{}

func (Observer) OnLanguageFallback(requested, resolved string) {}
func (Observer) OnLanguageMissing(lang string) {}
func (Observer) OnMessageMissing(lang string, msgCode int) {}
func (Observer) OnTemplateIssue(lang string, msgCode int, issue string) {}
```

Snapshot counters at runtime:

```go
stats, err := msgcat.SnapshotStats(catalog)
if err == nil {
  _ = stats.LanguageFallbacks
  _ = stats.MissingLanguages
  _ = stats.MissingMessages
  _ = stats.TemplateIssues
  _ = stats.DroppedEvents
  _ = stats.LastReloadAt
}
```

## Production Notes

- Keep `DefaultLanguage` explicit (`en` recommended).
- Define `FallbackLanguages` intentionally (for example for regional traffic).
- Use `StrictTemplates: true` in production to detect bad template usage early.
- Set `ObserverBuffer` to avoid request-path pressure from slow observers.
- Set `StatsMaxKeys` to cap cardinality (`__overflow__` key holds overflow counts).
- Use `go test -race ./...` in CI.
- For periodic YAML refresh, call `msgcat.Reload(catalog)` in a controlled goroutine and prefer atomic file replacement (`write temp + rename`).
- Use `ReloadRetries` and `ReloadRetryDelay` to reduce transient parse/read errors during rollout windows.
- If observer is configured, call `msgcat.Close(catalog)` on service shutdown.

### Runtime Contract

- `GetMessageWithCtx` / `GetErrorWithCtx` / `WrapErrorWithCtx` are safe for concurrent use.
- `LoadMessages` and `Reload` are safe concurrently with reads.
- `Reload` keeps the last in-memory state if reload fails.
- Observer callbacks are async and panic-protected; overflow is counted in `DroppedEvents`.

## Benchmarks

Run:

```bash
go test -run ^$ -bench . -benchmem ./...
```

## Integration Examples

- HTTP language middleware sample: `examples/http/main.go`
- Metrics/observer sample (expvar style): `examples/metrics/main.go`

## Context7 / LLM Docs

For full machine-friendly docs, see `docs/CONTEXT7.md`.
For retrieval-optimized chunks, see `docs/CONTEXT7_RETRIEVAL.md`.

## Release + Migration

- Changelog: `docs/CHANGELOG.md`
- Migration guide: `docs/MIGRATION.md`
- Release playbook: `docs/RELEASE.md`
- Support policy: `docs/SUPPORT.md`
