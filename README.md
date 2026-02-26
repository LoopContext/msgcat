# msgcat

`msgcat` is a lightweight i18n message catalog for Go focused on APIs and error handling.

It loads messages from YAML by language (string keys), resolves language from `context.Context`, supports runtime message loading with a reserved `sys.` key prefix, uses named template parameters, and can wrap domain errors with localized short/long messages.

**Maturity:** production-ready (`v1.x`) with SemVer and release/migration docs in `docs/`.

**Requirements:** Go 1.26 or later.

---

## Installation

```bash
go get github.com/loopcontext/msgcat
```

---

## Quick Start

### 1. Create message files

Default directory (when `ResourcePath` is empty):

```text
./resources/messages
```

One YAML file per language (e.g. `en.yaml`, `es.yaml`). Structure:

| Field     | Description |
|----------|-------------|
| `default`| Used when a message key is missing: `short` and `long` templates. |
| `set`    | Map of string message key → entry with optional `code`, `short`, `long`. Keys use `[a-zA-Z0-9_.-]+` (e.g. `greeting.hello`, `error.not_found`). |

Templates use **named parameters**: `{{name}}`, `{{plural:count\|singular\|plural}}`, `{{num:amount}}`, `{{date:when}}`.

Example `en.yaml`:

```yaml
default:
  short: Unexpected error
  long: Unexpected message was received and was not found in this catalog
set:
  greeting.hello:
    code: 1
    short: User created
    long: User {{name}} was created successfully
  items.count:
    short: You have {{count}} {{plural:count|item|items}}
    long: Total: {{num:amount}} generated at {{date:when}}
```

Example `es.yaml`:

```yaml
default:
  short: Error inesperado
  long: Se recibió un mensaje inesperado y no se encontró en el catálogo
set:
  greeting.hello:
    short: Usuario creado
    long: Usuario {{name}} fue creado correctamente
  items.count:
    short: Tienes {{count}} {{plural:count|elemento|elementos}}
    long: Total: {{num:amount}} generado el {{date:when}}
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

### 3. Resolve messages and errors from context

```go
ctx := context.WithValue(context.Background(), "language", "es-AR")

msg := catalog.GetMessageWithCtx(ctx, "greeting.hello", msgcat.Params{"name": "juan"})
fmt.Println(msg.ShortText) // "Usuario creado"
fmt.Println(msg.LongText)  // "Usuario juan fue creado correctamente"
fmt.Println(msg.Code)      // 1 (from YAML optional `code`)

params := msgcat.Params{"count": 3, "amount": 12345.5, "when": time.Now()}
err := catalog.WrapErrorWithCtx(ctx, errors.New("db timeout"), "items.count", params)
fmt.Println(err.Error()) // localized short message

if catErr, ok := err.(msgcat.Error); ok {
  fmt.Println(catErr.ErrorCode())
  fmt.Println(catErr.GetShortMessage())
  fmt.Println(catErr.GetLongMessage())
  fmt.Println(catErr.Unwrap()) // original "db timeout"
}
```

---

## Configuration

All fields of `msgcat.Config`:

| Field               | Type           | Description |
|---------------------|----------------|-------------|
| `ResourcePath`      | `string`       | Directory containing `*.yaml` message files. Default: `./resources/messages`. |
| `CtxLanguageKey`    | `ContextKey`   | Context key to read language (e.g. `"language"`). Supports typed key and string key lookup. |
| `DefaultLanguage`   | `string`       | Language used when context has no key or catalog has no match. Recommended: `"en"`. |
| `FallbackLanguages` | `[]string`     | Optional fallback list after requested/base (e.g. `[]string{"es"}`). |
| `StrictTemplates`   | `bool`         | If true, missing template params render as `<missing:N>`. Recommended `true` in production. |
| `Observer`          | `Observer`     | Optional; receives async events (fallback, missing lang, missing message, template issue). |
| `ObserverBuffer`    | `int`          | Size of observer event queue. Use ≥ 1 to avoid blocking the request path (e.g. 1024). |
| `StatsMaxKeys`      | `int`          | Max keys per stats map; overflow goes to `__overflow__`. Use to cap cardinality (e.g. 512). |
| `ReloadRetries`     | `int`          | Retries on reload parse/read failure (e.g. 2). |
| `ReloadRetryDelay`  | `time.Duration`| Delay between retries (e.g. 50ms). |
| `NowFn`             | `func() time.Time` | Optional; used for date formatting. Default: `time.Now`. |

---

## Features

- **Language from context**  
  Language is read from `context.Context` using `CtxLanguageKey` (typed or string key).

- **Fallback chain**  
  Order: requested language → base tag (`es-ar` → `es`) → `FallbackLanguages` → `DefaultLanguage` → `"en"`. First language that exists in the catalog is used.

- **YAML + runtime messages**  
  Messages from YAML plus runtime-loaded entries via `LoadMessages` for codes **9000–9999** (system range).

- **Template tokens**
  - `{{0}}`, `{{1}}`, … — positional parameters.
  - `{{plural:i|singular|plural}}` — plural form by parameter at index `i`.
  - `{{num:i}}` — localized number for parameter at index `i`.
  - `{{date:i}}` — localized date for parameter at index `i` (`time.Time` or `*time.Time`).

- **Strict template mode**  
  With `StrictTemplates: true`, missing or invalid params produce `<missing:N>` and observer events.

- **Error wrapping**  
  `WrapErrorWithCtx` and `GetErrorWithCtx` return errors implementing `msgcat.Error`: `ErrorCode()`, `GetShortMessage()`, `GetLongMessage()`, `Unwrap()`.

- **Concurrency**  
  Safe for concurrent reads; `LoadMessages` and `Reload` are safe to use concurrently with reads.

- **Reload**  
  `msgcat.Reload(catalog)` reloads YAML from disk with optional retries; runtime-loaded messages (9000–9999) are preserved. On failure, last in-memory state is kept.

- **Observability**  
  Optional `Observer` plus stats via `SnapshotStats` / `ResetStats`. Observer runs asynchronously and is panic-safe; queue overflow is counted in stats.

---

## API

### Core interface (`MessageCatalog`)

| Method | Description |
|--------|-------------|
| `LoadMessages(lang string, messages []RawMessage) error` | Add or replace messages for a language. Each `RawMessage` must have `Key` with prefix `sys.` (e.g. `sys.alert`). |
| `GetMessageWithCtx(ctx context.Context, msgKey string, params Params) *Message` | Resolve message for the context language; never nil. `params` can be nil. |
| `WrapErrorWithCtx(ctx context.Context, err error, msgKey string, params Params) error` | Wrap an error with localized short/long text and message code. |
| `GetErrorWithCtx(ctx context.Context, msgKey string, params Params) error` | Build an error with localized short/long text (no inner error). |

### Types

- **`Params`** — `map[string]interface{}` for named template parameters (e.g. `msgcat.Params{"name": "juan"}`).
- **`Message`** — `Code int`, `ShortText string`, `LongText string`.
- **`RawMessage`** — `Key` (required for `LoadMessages`), `ShortTpl`, `LongTpl` (YAML: `short`, `long`), optional `Code` (YAML: `code`).
- **`msgcat.Error`** — `Error() string`, `Unwrap() error`, `ErrorCode() int`, `GetShortMessage() string`, `GetLongMessage() string`.

### Package-level helpers

| Function | Description |
|----------|-------------|
| `msgcat.NewMessageCatalog(cfg Config) (MessageCatalog, error)` | Build catalog and load YAML from `ResourcePath`. |
| `msgcat.Reload(catalog MessageCatalog) error` | Reload YAML from disk (with retries if configured). |
| `msgcat.SnapshotStats(catalog MessageCatalog) (MessageCatalogStats, error)` | Copy of current stats. |
| `msgcat.ResetStats(catalog MessageCatalog) error` | Reset all stats counters. |
| `msgcat.Close(catalog MessageCatalog) error` | Stop observer worker and flush; call on shutdown if using an observer. |

### Constants

| Constant | Value | Description |
|----------|--------|-------------|
| `RuntimeKeyPrefix` | `"sys."` | Required prefix for message keys loaded via `LoadMessages`. |
| `CodeMissingMessage`  | 999999002 | Code used when a message key is missing in the catalog. |
| `CodeMissingLanguage` | 999999001 | Code used when the language is missing. |

## Observability

### Observer

Implement `msgcat.Observer` and pass it in `Config.Observer`:

```go
type Observer struct{}

func (Observer) OnLanguageFallback(requested, resolved string) {}
func (Observer) OnLanguageMissing(lang string)                 {}
func (Observer) OnMessageMissing(lang string, msgKey string)   {}
func (Observer) OnTemplateIssue(lang string, msgKey string, issue string) {}
```

Callbacks are invoked **asynchronously** and are panic-protected. If the observer queue is full, events are dropped and counted in `MessageCatalogStats.DroppedEvents`. Call `msgcat.Close(catalog)` on shutdown when using an observer.

### Stats (`MessageCatalogStats`)

| Field | Description |
|-------|-------------|
| `LanguageFallbacks` | Counts per `"requested->resolved"` language fallback. |
| `MissingLanguages`  | Counts per missing language. |
| `MissingMessages`   | Counts per `"lang:msgKey"` missing message. |
| `TemplateIssues`    | Counts per template issue key (e.g. `"lang:msgKey:issue"`). |
| `DroppedEvents`     | Counts per drop reason (e.g. `observer_queue_full`, `observer_closed`). |
| `LastReloadAt`      | Time of last successful reload. |

When `StatsMaxKeys` is set, each map is capped; extra keys are aggregated under `"__overflow__"`.

Example:

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

---

## Production notes

- Set `DefaultLanguage` explicitly (e.g. `"en"`).
- Set `FallbackLanguages` to match your traffic (e.g. regional defaults).
- Use `StrictTemplates: true` to catch bad template usage early.
- Set `ObserverBuffer` (e.g. 1024) so slow observers do not block the request path.
- Set `StatsMaxKeys` (e.g. 512) to avoid unbounded memory; watch `__overflow__` in dashboards.
- Run `go test -race ./...` in CI.
- For periodic YAML updates, call `msgcat.Reload(catalog)` (e.g. from a goroutine) and deploy files atomically (write to temp, then rename).
- Use `ReloadRetries` and `ReloadRetryDelay` to tolerate transient read/parse errors.
- If an observer is configured, call `msgcat.Close(catalog)` on service shutdown.

### Runtime contract

- `GetMessageWithCtx`, `GetErrorWithCtx`, `WrapErrorWithCtx` are safe for concurrent use.
- `LoadMessages` and `Reload` are safe concurrently with these reads.
- `Reload` keeps the previous in-memory state if the reload fails.
- Observer callbacks are async and panic-protected; overflow is reflected in `DroppedEvents`.

---

## Benchmarks

```bash
go test -run ^$ -bench . -benchmem ./...
```

---

## Examples

- HTTP language middleware: `examples/http/main.go`
- Metrics/observer (expvar-style): `examples/metrics/main.go`

---

## Docs and release

| Doc | Description |
|-----|-------------|
| [Changelog](docs/CHANGELOG.md) | Version history. |
| [Migration guide](docs/MIGRATION.md) | Upgrading and config changes. |
| [Release playbook](docs/RELEASE.md) | How to cut a release. |
| [Support policy](docs/SUPPORT.md) | Supported versions and compatibility. |
| [SECURITY.md](SECURITY.md) | How to report vulnerabilities. |
| [Context7](docs/CONTEXT7.md) | Machine-friendly API docs. |
| [Context7 retrieval](docs/CONTEXT7_RETRIEVAL.md) | Retrieval-oriented chunks. |
