# msgcat

`msgcat` is a lightweight i18n message catalog for Go focused on APIs and error handling.

It loads messages from YAML by language, resolves language from `context.Context`, supports runtime message loading for system codes (9000–9999), and can wrap domain errors with localized short/long messages.

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
| `group`  | Optional numeric group (e.g. `0`). |
| `default`| Used when a message code is missing: `short` and `long` templates. |
| `set`    | Map of message code → `short` / `long` template strings. |

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

### 3. Resolve messages and errors from context

```go
ctx := context.WithValue(context.Background(), "language", "es-AR")

msg := catalog.GetMessageWithCtx(ctx, 1, "juan")
fmt.Println(msg.ShortText) // "Usuario creado"
fmt.Println(msg.LongText)  // "Usuario juan fue creado correctamente"
fmt.Println(msg.Code)     // 1

err := catalog.WrapErrorWithCtx(ctx, errors.New("db timeout"), 2, 3, 12345.5, time.Now())
fmt.Println(err.Error()) // localized short message

if catErr, ok := err.(msgcat.Error); ok {
  fmt.Println(catErr.ErrorCode())      // 2
  fmt.Println(catErr.GetShortMessage())
  fmt.Println(catErr.GetLongMessage())
  fmt.Println(catErr.Unwrap())         // original "db timeout"
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
| `LoadMessages(lang string, messages []RawMessage) error` | Add or replace messages for a language. Only codes in 9000–9999 are allowed. |
| `GetMessageWithCtx(ctx context.Context, msgCode int, msgParams ...interface{}) *Message` | Resolve message for the context language; never nil. |
| `WrapErrorWithCtx(ctx context.Context, err error, msgCode int, msgParams ...interface{}) error` | Wrap an error with localized short/long text and message code. |
| `GetErrorWithCtx(ctx context.Context, msgCode int, msgParams ...interface{}) error` | Build an error with localized short/long text (no inner error). |

### Types

- **`Message`** — `Code int`, `ShortText string`, `LongText string`.
- **`RawMessage`** — `ShortTpl`, `LongTpl` (YAML: `short`, `long`); used in YAML and `LoadMessages`.
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
| `SystemMessageMinCode` | 9000 | Min code for runtime-loaded system messages. |
| `SystemMessageMaxCode` | 9999 | Max code for runtime-loaded system messages. |
| `CodeMissingMessage`  | 999999002 | Code used when a message is missing in the catalog. |
| `CodeMissingLanguage` | 999999001 | Code used when the language is missing. |

## Observability

### Observer

Implement `msgcat.Observer` and pass it in `Config.Observer`:

```go
type Observer struct{}

func (Observer) OnLanguageFallback(requested, resolved string) {}
func (Observer) OnLanguageMissing(lang string)                 {}
func (Observer) OnMessageMissing(lang string, msgCode int)      {}
func (Observer) OnTemplateIssue(lang string, msgCode int, issue string) {}
```

Callbacks are invoked **asynchronously** and are panic-protected. If the observer queue is full, events are dropped and counted in `MessageCatalogStats.DroppedEvents`. Call `msgcat.Close(catalog)` on shutdown when using an observer.

### Stats (`MessageCatalogStats`)

| Field | Description |
|-------|-------------|
| `LanguageFallbacks` | Counts per `"requested->resolved"` language fallback. |
| `MissingLanguages`  | Counts per missing language. |
| `MissingMessages`   | Counts per `"lang:code"` missing message. |
| `TemplateIssues`    | Counts per template issue key (e.g. `"lang:code:issue"`). |
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
