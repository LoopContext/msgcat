# msgcat - Context7 Reference

This document is the canonical LLM-oriented reference for `github.com/loopcontext/msgcat`.

It is optimized for retrieval systems (Context7-style ingestion): stable sections, explicit behavior, and implementation-aligned examples.

## 1. What It Is

`msgcat` is a Go i18n message catalog designed for backend/API use cases.

Primary goals:
- Localized messages and errors by language.
- Context-based language resolution.
- Runtime-safe use in concurrent services.
- Predictable fallback and observability.

## 2. Package and Module

- Module: `github.com/loopcontext/msgcat`
- Package import: `github.com/loopcontext/msgcat`

## 3. Main Concepts

- **Message catalog**: in-memory map of language -> message set (keyed by string) loaded from YAML.
- **Message key**: string key (e.g. `greeting.hello`) resolved per language.
- **Default message**: fallback template when key is missing for a language.
- **Language fallback chain**: requested language falls back through deterministic candidates.
- **Runtime system messages**: messages with key prefix `sys.` can be injected via `LoadMessages`.

## 4. YAML Format

Each language file is named `<lang>.yaml`, for example `en.yaml`, `es.yaml`.

### Schema

```yaml
default:
  short: string
  long: string
set:
  <key>:   # string key, e.g. greeting.hello, error.not_found
    code: int | string   # optional (YAML accepts either; stored as string)
    short: string
    long: string
```

Keys use `[a-zA-Z0-9_.-]+`. Templates use **named parameters**: `{{name}}`, `{{plural:count|singular|plural}}`, `{{num:amount}}`, `{{date:when}}`.

### Validation rules

- `default.short` or `default.long` must be non-empty.
- `set` can be omitted; it will be initialized empty.
- each key in `set` must be non-empty and match the key format.

## 5. Public Types

### `type Config struct`

```go
type Config struct {
  ResourcePath      string
  CtxLanguageKey    ContextKey
  DefaultLanguage   string
  FallbackLanguages []string
  StrictTemplates   bool
  Observer          Observer
  ObserverBuffer    int
  StatsMaxKeys      int
  ReloadRetries     int
  ReloadRetryDelay  time.Duration
  NowFn             func() time.Time
}
```

Field behavior:
- `ResourcePath`: directory with language YAML files. Default: `./resources/messages`.
- `CtxLanguageKey`: context key for language. Default: `"language"`.
- `DefaultLanguage`: default language when context does not provide one. Default: `"en"`.
- `FallbackLanguages`: extra ordered fallback list after requested/base language.
- `StrictTemplates`: if true, missing placeholder params are replaced by `<missing:n>` and counted as issues.
- `Observer`: optional hook receiver for fallback/miss/template events.
- `ObserverBuffer`: async observer queue size. Overflow is dropped and counted.
- `StatsMaxKeys`: max keys per stats map, overflow grouped under `__overflow__`.
- `ReloadRetries` / `ReloadRetryDelay`: retry strategy for transient reload parse/read errors.
- `NowFn`: injectable clock function. Default: `time.Now`.

### `type Message struct`

```go
type Message struct {
  LongText  string
  ShortText string
  Code      string // Optional; user-defined (e.g. "404", "ERR_001"). Empty when not set. Use Key when empty.
  Key       string // Message key (e.g. "greeting.hello"); always set.
}
```

### `type Params`

```go
type Params map[string]interface{}
```

Named template parameters. Use `msgcat.Params{"name": "juan", "count": 3}`.

### `type RawMessage struct`

```go
type RawMessage struct {
  LongTpl  string       `yaml:"long"`
  ShortTpl string       `yaml:"short"`
  Code     OptionalCode `yaml:"code"` // Optional; any value. YAML: code: 404 or code: "ERR_001". Not unique.
  Key      string       `yaml:"-"`    // required when using LoadMessages; must have prefix sys.
}
```

`OptionalCode` unmarshals from YAML as int or string. In Go use `msgcat.CodeInt(503)` or `msgcat.CodeString("ERR_MAINT")`.

### `type MessageCatalogStats struct`

```go
type MessageCatalogStats struct {
  LanguageFallbacks map[string]int
  MissingLanguages  map[string]int
  MissingMessages   map[string]int
  TemplateIssues    map[string]int
  DroppedEvents     map[string]int
  LastReloadAt      time.Time
}
```

### `type Observer interface`

```go
type Observer interface {
  OnLanguageFallback(requestedLang string, resolvedLang string)
  OnLanguageMissing(lang string)
  OnMessageMissing(lang string, msgKey string)
  OnTemplateIssue(lang string, msgKey string, issue string)
}
```

## 6. Public API

### Constructor

```go
func NewMessageCatalog(cfg Config) (MessageCatalog, error)
```

### Interface

```go
type MessageCatalog interface {
  LoadMessages(lang string, messages []RawMessage) error
  GetMessageWithCtx(ctx context.Context, msgKey string, params Params) *Message
  WrapErrorWithCtx(ctx context.Context, err error, msgKey string, params Params) error
  GetErrorWithCtx(ctx context.Context, msgKey string, params Params) error
}
```

For `LoadMessages`, each `RawMessage` must have `Key` set with prefix `RuntimeKeyPrefix` (`"sys."`).

### Helper functions

```go
func Reload(catalog MessageCatalog) error
func SnapshotStats(catalog MessageCatalog) (MessageCatalogStats, error)
func ResetStats(catalog MessageCatalog) error
func Close(catalog MessageCatalog) error
```

Notes:
- helpers require concrete catalog support; otherwise return error.
- call `Close(catalog)` on shutdown when observer is enabled.

## 7. Constants

```go
const (
  RuntimeKeyPrefix     = "sys."
  CodeMissingMessage  = "msgcat.missing_message"
  CodeMissingLanguage = "msgcat.missing_language"
)
```

## 7.1 Message and error codes (optional `code` field)

Many projects already have **error or message codes** (HTTP statuses, legacy numbers, string ids like `ERR_NOT_FOUND`). The optional **`code`** field lets you **store that value** in the catalog and have it returned in `Message.Code` and `ErrorCode()` so your API can expose it as-is.

- **Optional** — Omit `code` when you don’t need it; use `Message.Key` or `ErrorKey()` as the identifier when `Code` / `ErrorCode()` is empty.
- **Any value** — Codes are strings. YAML accepts `code: 404` (becomes `"404"`) or `code: "ERR_NOT_FOUND"`. In Go: `msgcat.CodeInt(503)` or `msgcat.CodeString("ERR_MAINT")`.
- **Not unique** — Uniqueness is not enforced. The same code can appear on multiple messages if that matches your design.
- **Opaque to the catalog** — The library only stores and returns the value; meaning and uniqueness are entirely up to the caller.

Use `code` when you need a stable, project-specific value for clients (e.g. status or error enums). Otherwise, rely on the message **key** as the identifier.

## 8. Language Resolution Algorithm

Given requested language from context (normalized lower-case, `_` -> `-`):

1. requested language (for example `es-ar`)
2. base language (`es`)
3. each `Config.FallbackLanguages` entry in order
4. `Config.DefaultLanguage`
5. final hard fallback: `en`

First language present in catalog is used.

If a fallback language was used, observer/stats records a fallback event.

If none found, response uses `CodeMissingLanguage` and `MessageCatalogNotFound`.

## 9. Template Engine

Supported tokens (all **named**):

- Simple: `{{name}}`
- Plural: `{{plural:count|singular|plural}}`
- Number: `{{num:amount}}`
- Date: `{{date:when}}`

Parameter names use `[a-zA-Z_][a-zA-Z0-9_.]*`. Pass values via `Params` (e.g. `msgcat.Params{"name": "juan", "count": 3}`).

Processing order:
1. plural
2. number
3. date
4. simple

### Important limitation

Plural branches are plain text. Do not nest other placeholders inside `plural` branches.

Good:
- `"You have {{count}} {{plural:count|item|items}}"`

Avoid:
- `"You have {{plural:count|1 item|{{count}} items}}"`

### Strict template behavior

When `StrictTemplates=true` and a parameter is missing:
- token is replaced with `<missing:paramName>`
- observer/stats receives a template issue event

Example strict placeholder: `"Hello {{name}}"` with params `nil` or missing `name` => `"Hello <missing:name>"`.

When strict mode is off:
- unresolved token is left as-is
- issue is still recorded

## 10. Number/Date Localization

`{{num:name}}` (e.g. `{{num:amount}}`):
- default style: `12,345.5`
- for base languages `es`, `pt`, `fr`, `de`, `it`: `12.345,5`

`{{date:name}}` (e.g. `{{date:when}}`):
- default: `MM/DD/YYYY`
- for base languages `es`, `pt`, `fr`, `de`, `it`: `DD/MM/YYYY`

Accepted date params:
- `time.Time`
- `*time.Time`

## 11. Error Model

`WrapErrorWithCtx` and `GetErrorWithCtx` return a concrete error implementing `msgcat.Error`:
- `Error()` -> short localized message
- `ErrorCode() string` -> optional; user-defined. Empty when not set. Use `ErrorKey()` when empty.
- `ErrorKey()` -> message key; use as API identifier when `ErrorCode()` is empty
- `GetShortMessage()` and `GetLongMessage()`
- `Unwrap()` support for wrapped error chaining

Code is optional and can be any value (string); uniqueness is not enforced. When empty, return `ErrorKey()` as the API value.

## 12. Runtime Loading and Reload

### Runtime loading

`LoadMessages(lang, messages)`:
- each `RawMessage` must have `Key` with prefix `sys.` (e.g. `sys.alert`)
- rejects duplicate key per language
- stores messages in runtime set so they survive YAML reload

### Reload

`Reload(catalog)`:
- re-reads YAML files from `ResourcePath`
- validates and normalizes files
- preserves runtime-loaded messages
- retries based on `ReloadRetries` and `ReloadRetryDelay`

## 13. Concurrency Guarantees

The catalog uses `sync.RWMutex` for internal maps.

Safe operations in concurrent services:
- read path: `GetMessageWithCtx`, `GetErrorWithCtx`, `WrapErrorWithCtx`
- write path: `LoadMessages`, `Reload`
- statistics snapshot: `SnapshotStats`

Race tests pass with `go test -race ./...`.

## 13.1 Runtime Contract

- Read path methods are safe under concurrent use.
- `LoadMessages` and `Reload` can run concurrently with reads.
- If `Reload` fails, previously loaded in-memory messages remain available.
- Observer callback failures do not fail request-path calls.

## 14. Observability Semantics

`SnapshotStats` returns cumulative counters since catalog creation:
- `LanguageFallbacks`: keyed as `"requested->resolved"`
- `MissingLanguages`: keyed by requested language
- `MissingMessages`: keyed as `"lang:msgKey"`
- `TemplateIssues`: keyed as `"lang:msgKey:issue"`
- `DroppedEvents`: internal drop counters (for example observer queue overflow)
- `LastReloadAt`: timestamp set using `Config.NowFn`

Observer hooks are dispatched asynchronously through a bounded queue.
Panics inside observer callbacks are recovered to protect request path.

## 15. Performance Notes

Current benchmark command:

```bash
go test -run ^$ -bench . -benchmem ./...
```

Recent example on Apple Silicon (M4 Pro):
- `BenchmarkGetMessageWithCtx`: ~1226 ns/op, ~1033 B/op, 44 allocs/op
- `BenchmarkGetErrorWithCtx`: ~1252 ns/op, ~1097 B/op, 45 allocs/op

Treat values as machine/runtime-dependent.

## 16. Canonical Usage Example

```go
package main

import (
  "context"
  "fmt"
  "time"

  "github.com/loopcontext/msgcat"
)

func main() {
  catalog, err := msgcat.NewMessageCatalog(msgcat.Config{
    ResourcePath:      "./resources/messages",
    CtxLanguageKey:    "language",
    DefaultLanguage:   "en",
    FallbackLanguages: []string{"es"},
    StrictTemplates:   true,
  })
  if err != nil {
    panic(err)
  }

  ctx := context.WithValue(context.Background(), "language", "es-MX")
  params := msgcat.Params{"count": 3, "amount": 12345.5, "when": time.Now()}
  msg := catalog.GetMessageWithCtx(ctx, "items.count", params)
  fmt.Println(msg.ShortText)
  fmt.Println(msg.LongText)

  stats, err := msgcat.SnapshotStats(catalog)
  if err == nil {
    fmt.Println(stats.LanguageFallbacks)
  }
}
```

## 17. Compatibility and Caveats

- Context key compatibility supports both typed key and plain string key.
- Missing message key uses language default message and `CodeMissingMessage`.
- Missing language uses `MessageCatalogNotFound` and `CodeMissingLanguage`.
- `NowFn` exists for future deterministic time-driven extensions; date formatting currently uses params directly.

## 18. Recommended CI Checks

```bash
go test ./...
go test -race ./...
go test -run ^$ -bench . -benchmem ./...
```

## 19. Examples by API surface (for retrieval)

Each snippet is self-contained for copy-paste or chunk retrieval. For runnable programs see the repository `examples/` directory: `basic`, `load_messages`, `reload`, `strict`, `stats`, `http`, `metrics`.

### Example: NewMessageCatalog minimal

```go
catalog, err := msgcat.NewMessageCatalog(msgcat.Config{})
// Uses ./resources/messages, DefaultLanguage "en", CtxLanguageKey "language"
```

### Example: NewMessageCatalog full config

```go
catalog, err := msgcat.NewMessageCatalog(msgcat.Config{
  ResourcePath:      "./resources/messages",
  CtxLanguageKey:    msgcat.ContextKey("language"),
  DefaultLanguage:   "en",
  FallbackLanguages: []string{"es"},
  StrictTemplates:   true,
  Observer:          myObserver,
  ObserverBuffer:    1024,
  StatsMaxKeys:      512,
  ReloadRetries:     2,
  ReloadRetryDelay:  50 * time.Millisecond,
  NowFn:             time.Now,
})
```

### Example: GetMessageWithCtx with nil params

```go
msg := catalog.GetMessageWithCtx(ctx, "greeting.hello", nil)
// Use when message has no placeholders or default text is enough
fmt.Println(msg.ShortText, msg.Code)
```

### Example: GetMessageWithCtx with Params

```go
msg := catalog.GetMessageWithCtx(ctx, "greeting.template", msgcat.Params{
  "name": "juan", "detail": "admin",
})
```

### Example: Simple placeholder {{name}}

```go
// YAML: short: "Hello {{name}}"
msg := catalog.GetMessageWithCtx(ctx, "greeting.hello", msgcat.Params{"name": "juan"})
```

### Example: Plural placeholder {{plural:count|singular|plural}}

```go
// YAML: short: "You have {{count}} {{plural:count|item|items}}"
msg := catalog.GetMessageWithCtx(ctx, "items.count", msgcat.Params{"count": 3})
// => "You have 3 items"
msg := catalog.GetMessageWithCtx(ctx, "items.count", msgcat.Params{"count": 1})
// => "You have 1 item"
```

### Example: Number placeholder {{num:amount}}

```go
// YAML: short: "Total: {{num:amount}}"
msg := catalog.GetMessageWithCtx(ctx, "report.total", msgcat.Params{"amount": 12345.67})
// en => "Total: 12,345.67"; es => "Total: 12.345,67"
```

### Example: Date placeholder {{date:when}}

```go
// YAML: short: "Generated at {{date:when}}"
msg := catalog.GetMessageWithCtx(ctx, "report.generated", msgcat.Params{"when": time.Now()})
// en => MM/DD/YYYY; es/pt/fr/de/it => DD/MM/YYYY
```

### Example: GetErrorWithCtx

```go
err := catalog.GetErrorWithCtx(ctx, "error.not_found", msgcat.Params{"resource": "order"})
fmt.Println(err.Error()) // short localized message
```

### Example: WrapErrorWithCtx and msgcat.Error

```go
inner := errors.New("db timeout")
err := catalog.WrapErrorWithCtx(ctx, inner, "error.timeout", nil)
var catErr msgcat.Error
if errors.As(err, &catErr) {
  catErr.ErrorCode()
  catErr.GetShortMessage()
  catErr.GetLongMessage()
  catErr.Unwrap() // original inner
}
```

### Example: LoadMessages with sys. prefix

```go
err := catalog.LoadMessages("en", []msgcat.RawMessage{{
  Key:      "sys.maintenance",
  ShortTpl: "Under maintenance",
  LongTpl:  "Back in {{minutes}} minutes.",
  Code:     503,
}})
msg := catalog.GetMessageWithCtx(ctx, "sys.maintenance", msgcat.Params{"minutes": 5})
```

### Example: Reload

```go
err := msgcat.Reload(catalog)
// Re-reads YAML; runtime sys.* messages are preserved; on failure previous state kept
```

### Example: SnapshotStats and ResetStats

```go
stats, err := msgcat.SnapshotStats(catalog)
_ = stats.LanguageFallbacks
_ = stats.MissingMessages
_ = stats.TemplateIssues
err = msgcat.ResetStats(catalog)
```

### Example: Close (with observer)

```go
defer func() { _ = msgcat.Close(catalog) }()
// Call on shutdown when Config.Observer is set; stops worker and flushes queue
```

### Example: Observer implementation

```go
type obs struct{}
func (obs) OnLanguageFallback(req, res string) {}
func (obs) OnLanguageMissing(lang string)       {}
func (obs) OnMessageMissing(lang string, msgKey string) {}
func (obs) OnTemplateIssue(lang string, msgKey string, issue string) {}
catalog, _ := msgcat.NewMessageCatalog(msgcat.Config{Observer: obs{}, ObserverBuffer: 1024})
```

### Example: Context language (typed vs string key)

```go
ctx = context.WithValue(ctx, msgcat.ContextKey("language"), "es-MX")
ctx = context.WithValue(ctx, "language", "es-MX")
// Both are supported for CtxLanguageKey lookup
```

### Example: Missing message key

```go
msg := catalog.GetMessageWithCtx(ctx, "nonexistent.key", nil)
// msg.Code == msgcat.CodeMissingMessage; short/long = default message for language
```

### Example: Missing language

```go
ctx = context.WithValue(ctx, "language", "zz")
msg := catalog.GetMessageWithCtx(ctx, "greeting.hello", nil)
// msg.Code == msgcat.CodeMissingLanguage; text = MessageCatalogNotFound
```
