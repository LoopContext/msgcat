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
| `group`  | Optional. Int or string (e.g. `group: 0` or `group: "api"`) for organization; catalog does not interpret it. See [Optional group](#optional-group). |
| `default`| Used when a message key is missing: `short` and `long` templates. |
| `set`    | Map of string message key → entry with optional `code`, `short`, `long`; optional **`short_forms`** / **`long_forms`** (CLDR: zero, one, two, few, many, other), **`plural_param`** (default `count`). Keys use `[a-zA-Z0-9_.-]+`. |

Templates use **named parameters**: `{{name}}`, `{{plural:count\|singular\|plural}}`, `{{num:amount}}`, `{{date:when}}`.

Example `en.yaml`:

```yaml
default:
  short: Unexpected error
  long: Unexpected message was received and was not found in this catalog
set:
  greeting.hello:
    code: GREETING_HELLO
    short: User created
    long: User {{name}} was created successfully
  items.count:
    short: You have {{count}} {{plural:count|item|items}}
    long: Total: {{num:amount}} generated at {{date:when}}
  person.cats:
    short: "{{plural:count|zero:No cats|one:One cat|other:{{count}} cats}}"
```

Example `es.yaml`:

```yaml
default:
  short: Error inesperado
  long: Se recibió un mensaje inesperado y no se encontró en el catálogo
set:
  greeting.hello:
    code: GREETING_HELLO
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
fmt.Println(msg.Code)      // "GREETING_HELLO" (from YAML optional code; Code is string)

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
  Messages from YAML plus runtime-loaded entries via `LoadMessages`; keys must use the **`sys.`** prefix (e.g. `sys.alert`).

- **Template tokens (named parameters)**
  - `{{name}}` — simple substitution.
  - `{{plural:count|singular|plural}}` — binary plural by named count parameter.
  - `{{plural:count|one:item|few:items|many:items|other:items}}` — multi-form plural by named count parameter using CLDR rules (supports 0, 1, 2, few, many, other depending on language).
  - **CLDR plural forms** — optional `short_forms` / `long_forms` per entry (keys: `zero`, `one`, `two`, `few`, `many`, `other`) for full locale rules; see [CLDR and messages in Go](docs/CLDR_AND_GO_MESSAGES_PLAN.md).
  - `{{num:amount}}` — localized number for named parameter.
  - `{{date:when}}` — localized date for named parameter (`time.Time` or `*time.Time`).

- **Messages in Go**  
  Define content with **`msgcat.MessageDef`** (Key, Short, Long, or ShortForms/LongForms, Code). Run **`msgcat extract -source en.yaml -out en.yaml .`** to merge those definitions into your source YAML.

- **Strict template mode**  
  With `StrictTemplates: true`, missing or invalid params produce `<missing:paramName>` and observer events.

- **Error wrapping**  
  `WrapErrorWithCtx` and `GetErrorWithCtx` return errors implementing `msgcat.Error`: `ErrorCode() string` (optional), `ErrorKey()`, `GetShortMessage()`, `GetLongMessage()`, `Unwrap()`. See [Message and error codes](#message-and-error-codes).

- **Concurrency**  
  Safe for concurrent reads; `LoadMessages` and `Reload` are safe to use concurrently with reads.

- **Reload**  
  `msgcat.Reload(catalog)` reloads YAML from disk with optional retries; runtime-loaded messages (keys with `sys.` prefix) are preserved. On failure, last in-memory state is kept.

- **Observability**  
  Optional `Observer` plus stats via `SnapshotStats` / `ResetStats`. Observer runs asynchronously and is panic-safe; queue overflow is counted in stats.

### Optional group

Message files can include an optional top-level **`group`** with an integer or string value (e.g. `group: 0` or `group: "api"`). Use it to tag files for organization or tooling. The catalog does not interpret group; it only stores it. The CLI preserves `group` when running `extract` (sync) and `merge`.

```yaml
group: api
default:
  short: Unexpected error
  long: ...
set:
  error.not_found:
    short: Not found
    long: ...
```

---

## CLI workflow (extract & merge)

The **msgcat** CLI helps discover message keys from Go code and prepare translation files.

**Install:**

```bash
go install github.com/loopcontext/msgcat/cmd/msgcat@latest
```

**Extract (keys only)** — list message keys used in Go (e.g. in `GetMessageWithCtx`, `WrapErrorWithCtx`, `GetErrorWithCtx`):

```bash
msgcat extract [paths]              # print keys to stdout (default: current dir)
msgcat extract -out keys.txt .      # write keys to file
msgcat extract -include-tests .     # include _test.go files
```

**Extract (sync to source YAML)** — add keys from API calls (empty `short`/`long`) and **merge `msgcat.MessageDef`** struct literals from Go (full content) into your source file:

```bash
msgcat extract -source resources/messages/en.yaml -out resources/messages/en.yaml .
```

**Merge** — produce `translate.<lang>.yaml` files from a source file. For each target language, missing or empty entries use source text as placeholder; existing translations are kept. Copies `group` and `default` from source.

```bash
msgcat merge -source resources/messages/en.yaml -targetLangs es,fr -outdir resources/messages
# Creates translate.es.yaml, translate.fr.yaml

msgcat merge -source resources/messages/en.yaml -targetDir resources/messages -outdir resources/messages
# Infers target languages from existing *.yaml in targetDir (excluding source and translate.*)
```

After translators fill `translate.es.yaml`, rename or copy it to `es.yaml` for runtime.

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
- **`Message`** — `ShortText`, `LongText`, `Code string` (optional; see [Message and error codes](#message-and-error-codes)), `Key string` (message key; use when `Code` is empty).
- **`RawMessage`** — `Key` (required for `LoadMessages`), `ShortTpl`, `LongTpl`, optional `Code`; optional **`ShortForms`** / **`LongForms`** (CLDR plural maps), **`PluralParam`** (default `"count"`).
- **`MessageDef`** — For “messages in Go”: `Key`, `Short`, `Long`, optional `ShortForms` / `LongForms`, `PluralParam`, `Code`. Use with **msgcat extract -source** to merge into YAML.
- **`msgcat.Error`** — `Error()`, `Unwrap()`, `ErrorCode() string` (optional), `ErrorKey() string` (use when `ErrorCode()` is empty), `GetShortMessage()`, `GetLongMessage()`.

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
| `CodeMissingMessage`  | `"msgcat.missing_message"` | Code used when a message key is missing in the catalog. |
| `CodeMissingLanguage` | `"msgcat.missing_language"` | Code used when the language is missing. |

## Message and error codes

Many projects already use **error or message codes** (HTTP statuses, legacy numeric codes, string identifiers like `ERR_NOT_FOUND`). The optional **`code`** field in the catalog lets you **store that value** with each message and have it returned in `Message.Code` and `ErrorCode()` so your API can expose it unchanged.

- **Optional** — You can omit `code` entirely. When empty, use `Message.Key` or `ErrorKey()` as the stable identifier for clients (e.g. in JSON: `"error_code": msg.Code or msg.Key`).
- **Any value** — Codes are strings. In YAML you can write `code: ERR_NOT_FOUND` or `code: "404"`. In Go use `msgcat.CodeString("ERR_MAINT")` or `msgcat.CodeInt(503)`.
- **Not unique** — The catalog does not require codes to be unique. If your design uses the same code for several messages (e.g. same HTTP status for different keys), you can repeat the same `code` value.
- **Your identifier** — The catalog never interprets the code; it only stores and returns it. You decide what values to use and how to expose them in your API.

**When to set a code:** Use it when you need a stable, project-specific value to return to clients (status codes, error enums, etc.). Strings are generally preferred over integers as they are more descriptive (e.g., `code: ERR_ACCESS_DENIED` instead of `code: 403`). When you don’t need a separate code, leave it unset and use the message **key** as the identifier.

Helpers for building `RawMessage.Code` in code: `msgcat.CodeInt(503)`, `msgcat.CodeString("ERR_NOT_FOUND")`.

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

## API examples (every nook and cranny)

All of the following assume a catalog and context are set up; use your own YAML keys and params as needed.

### Create catalog (minimal vs full config)

```go
// Minimal: uses ./resources/messages, language "en", no observer
catalog, err := msgcat.NewMessageCatalog(msgcat.Config{})

// Full: custom path, fallbacks, strict templates, observer, reload retries
catalog, err := msgcat.NewMessageCatalog(msgcat.Config{
  ResourcePath:      "./resources/messages",
  CtxLanguageKey:    msgcat.ContextKey("language"), // typed key
  DefaultLanguage:   "en",
  FallbackLanguages: []string{"es", "pt"},
  StrictTemplates:   true,
  Observer:          myObserver,
  ObserverBuffer:    1024,
  StatsMaxKeys:      512,
  ReloadRetries:     2,
  ReloadRetryDelay:  50 * time.Millisecond,
  NowFn:             time.Now,
})
```

### GetMessageWithCtx: no params vs named params

```go
// No template params: pass nil
msg := catalog.GetMessageWithCtx(ctx, "greeting.hello", nil)
fmt.Println(msg.ShortText, msg.LongText, msg.Code)

// Named params: use Params map
msg := catalog.GetMessageWithCtx(ctx, "greeting.template", msgcat.Params{
  "name":   "juan",
  "detail": "admin",
})
```

### Template placeholders: simple, plural, number, date

```go
// Simple: {{name}}, {{detail}}, etc.
msg := catalog.GetMessageWithCtx(ctx, "greeting.template", msgcat.Params{
  "name": "juan", "detail": "nice",
})

// Plural: {{count}} and {{plural:count|item|items}}
msg := catalog.GetMessageWithCtx(ctx, "items.count", msgcat.Params{
  "count": 3,
})

// Number: {{num:amount}} (localized thousands/decimal)
msg := catalog.GetMessageWithCtx(ctx, "report.total", msgcat.Params{
  "amount": 12345.67,
})

// Date: {{date:when}} (localized format)
msg := catalog.GetMessageWithCtx(ctx, "report.generated", msgcat.Params{
  "when": time.Now(),
})

// All together
msg := catalog.GetMessageWithCtx(ctx, "items.count", msgcat.Params{
  "count": 3, "amount": 12345.5, "generatedAt": time.Now(),
})
```

### GetErrorWithCtx and WrapErrorWithCtx

```go
// Error without wrapping an underlying error
err := catalog.GetErrorWithCtx(ctx, "error.not_found", msgcat.Params{"resource": "order"})
fmt.Println(err.Error()) // short message

// Wrap a domain error with localized message
inner := errors.New("db: connection timeout")
err := catalog.WrapErrorWithCtx(ctx, inner, "error.timeout", nil)
if catErr, ok := err.(msgcat.Error); ok {
  fmt.Println(catErr.Error())           // short message
  fmt.Println(catErr.ErrorCode())       // optional; empty when not set in catalog
  fmt.Println(catErr.ErrorKey())        // message key; use as API id when ErrorCode() is empty
  fmt.Println(catErr.GetShortMessage())
  fmt.Println(catErr.GetLongMessage())
  fmt.Println(catErr.Unwrap() == inner) // true
}
```

**Code** is optional: use it to store your own error/message codes (e.g. HTTP status, `"ERR_001"`) and return them from the API. When empty, use `Message.Key` or `ErrorKey()`. See [Message and error codes](#message-and-error-codes).

### LoadMessages (runtime messages with sys. prefix)

```go
err := catalog.LoadMessages("en", []msgcat.RawMessage{
  {
    Key:      "sys.maintenance",
    ShortTpl: "Service under maintenance",
    LongTpl:  "The service is temporarily unavailable. Try again in {{minutes}} minutes.",
    Code:     msgcat.CodeInt(503),
  },
})
// Then use the key like any other
msg := catalog.GetMessageWithCtx(ctx, "sys.maintenance", msgcat.Params{"minutes": 5})
```

### Reload, stats, close

```go
// Reload YAML from disk (keeps runtime-loaded sys.* messages)
err := msgcat.Reload(catalog)

// Snapshot current stats (safe concurrent read)
stats, err := msgcat.SnapshotStats(catalog)
if err == nil {
  for k, n := range stats.MissingMessages { fmt.Println(k, n) }
}

// Reset all counters to zero
err = msgcat.ResetStats(catalog)

// On shutdown when using an observer: stop worker and flush queue
err = msgcat.Close(catalog)
```

### Language from context (typed key vs string key)

```go
// Both work: typed ContextKey or plain string
ctx = context.WithValue(ctx, msgcat.ContextKey("language"), "es-MX")
ctx = context.WithValue(ctx, "language", "es-MX")
msg := catalog.GetMessageWithCtx(ctx, "greeting.hello", nil)
```

### Observer implementation

```go
type myObserver struct{}

func (myObserver) OnLanguageFallback(requested, resolved string) {
  log.Printf("fallback %s -> %s", requested, resolved)
}
func (myObserver) OnLanguageMissing(lang string) {
  log.Printf("missing language: %s", lang)
}
func (myObserver) OnMessageMissing(lang string, msgKey string) {
  log.Printf("missing message %s:%s", lang, msgKey)
}
func (myObserver) OnTemplateIssue(lang string, msgKey string, issue string) {
  log.Printf("template issue %s:%s: %s", lang, msgKey, issue)
}

catalog, _ := msgcat.NewMessageCatalog(msgcat.Config{
  Observer:       myObserver{},
  ObserverBuffer: 1024,
})
```

### Missing message / missing language

```go
// Unknown key: returns default message for that language, Code = CodeMissingMessage (string)
msg := catalog.GetMessageWithCtx(ctx, "unknown.key", nil)
if msg.Code == msgcat.CodeMissingMessage {
  // key was not in catalog
}

// Requested language not in catalog: uses MessageCatalogNotFound text, Code = CodeMissingLanguage (string)
ctx = context.WithValue(ctx, "language", "xx")
msg := catalog.GetMessageWithCtx(ctx, "greeting.hello", nil)
if msg.Code == msgcat.CodeMissingLanguage {
  // no language match in catalog
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

Runnable programs (each uses a temp dir and minimal YAML so you can run from any directory):

| Example | What it demonstrates |
|---------|----------------------|
| `examples/basic` | NewMessageCatalog, GetMessageWithCtx (nil and with Params), GetErrorWithCtx, WrapErrorWithCtx, msgcat.Error |
| `examples/cldr_plural` | CLDR plural forms (short_forms/long_forms) with one/other and plural_param |
| `examples/msgdef` | MessageDef in Go and extract workflow |
| `examples/load_messages` | LoadMessages with `sys.` prefix, using runtime-loaded keys |
| `examples/reload` | Reload(catalog) to re-read YAML from disk |
| `examples/strict` | StrictTemplates and observer for missing template params |
| `examples/stats` | SnapshotStats, ResetStats, stat keys |
| `examples/http` | HTTP server with language from Accept-Language and GetMessageWithCtx |
| `examples/metrics` | Observer (expvar-style) and Close on shutdown |

Run from repo root: `go run ./examples/basic`, `go run ./examples/load_messages`, etc.

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
| [CLI workflow plan](docs/CLI_WORKFLOW_PLAN.md) | Extract and merge workflow; optional group. |
| [CLDR and messages in Go](docs/CLDR_AND_GO_MESSAGES_PLAN.md) | CLDR plurals and MessageDef + extract (roadmap). |
