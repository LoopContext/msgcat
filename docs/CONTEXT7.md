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

- **Message catalog**: in-memory map of language -> message set loaded from YAML.
- **Message code**: integer key resolved per language.
- **Default message**: fallback template when code is missing for a language.
- **Language fallback chain**: requested language falls back through deterministic candidates.
- **Runtime system messages**: codes `9000-9999` can be injected from code.

## 4. YAML Format

Each language file is named `<lang>.yaml`, for example `en.yaml`, `es.yaml`.

### Schema

```yaml
group: 0
default:
  short: string
  long: string
set:
  <code>:
    short: string
    long: string
```

### Validation rules

- `group >= 0`
- `default.short` or `default.long` must be non-empty.
- `set` can be omitted; it will be initialized empty.
- each code in `set` must be `> 0`.

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
  Code      int
}
```

### `type RawMessage struct`

```go
type RawMessage struct {
  LongTpl  string `yaml:"long"`
  ShortTpl string `yaml:"short"`
  Code     int
}
```

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
  OnMessageMissing(lang string, msgCode int)
  OnTemplateIssue(lang string, msgCode int, issue string)
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
  GetMessageWithCtx(ctx context.Context, msgCode int, msgParams ...interface{}) *Message
  WrapErrorWithCtx(ctx context.Context, err error, msgCode int, msgParams ...interface{}) error
  GetErrorWithCtx(ctx context.Context, msgCode int, msgParams ...interface{}) error
}
```

### Helper functions

```go
func Reload(catalog MessageCatalog) error
func SnapshotStats(catalog MessageCatalog) (MessageCatalogStats, error)
func ResetStats(catalog MessageCatalog) error
func Close(catalog MessageCatalog) error
```

Notes:
- helpers require concrete catalog support; otherwise return error.

## 7. Constants

```go
const (
  SystemMessageMinCode = 9000
  SystemMessageMaxCode = 9999
  CodeMissingMessage   = 999999998
  CodeMissingLanguage  = 99999999
)
```

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

Supported tokens:

- Positional: `{{0}}`, `{{1}}`, ...
- Plural: `{{plural:i|singular|plural}}`
- Number: `{{num:i}}`
- Date: `{{date:i}}`

Processing order:
1. plural
2. number
3. date
4. simple positional

### Important limitation

Plural branches are plain text. Do not nest other placeholders inside `plural` branches.

Good:
- `"You have {{0}} {{plural:0|item|items}}"`

Avoid:
- `"You have {{plural:0|1 item|{{0}} items}}"`

### Strict template behavior

When `StrictTemplates=true` and parameter index is missing:
- token is replaced with `<missing:n>`
- observer/stats receives a template issue event

When strict mode is off:
- unresolved token is left as-is
- issue is still recorded

## 10. Number/Date Localization

`{{num:i}}`:
- default style: `12,345.5`
- for base languages `es`, `pt`, `fr`, `de`, `it`: `12.345,5`

`{{date:i}}`:
- default: `MM/DD/YYYY`
- for base languages `es`, `pt`, `fr`, `de`, `it`: `DD/MM/YYYY`

Accepted date params:
- `time.Time`
- `*time.Time`

## 11. Error Model

`WrapErrorWithCtx` and `GetErrorWithCtx` return a concrete error with:
- `Error()` -> short localized message
- `ErrorCode()` -> resolved code
- `GetShortMessage()` and `GetLongMessage()`
- `Unwrap()` support for wrapped error chaining

## 12. Runtime Loading and Reload

### Runtime loading

`LoadMessages(lang, messages)`:
- only accepts codes in `[9000, 9999]`
- rejects duplicate code per language
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

## 14. Observability Semantics

`SnapshotStats` returns cumulative counters since catalog creation:
- `LanguageFallbacks`: keyed as `"requested->resolved"`
- `MissingLanguages`: keyed by requested language
- `MissingMessages`: keyed as `"lang:code"`
- `TemplateIssues`: keyed as `"lang:code:issue"`
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
  msg := catalog.GetMessageWithCtx(ctx, 2, 3, 12345.5, time.Now())
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
- Missing code uses language default message and `CodeMissingMessage`.
- Missing language uses `MessageCatalogNotFound` and `CodeMissingLanguage`.
- `NowFn` exists for future deterministic time-driven extensions; date formatting currently uses params directly.

## 18. Recommended CI Checks

```bash
go test ./...
go test -race ./...
go test -run ^$ -bench . -benchmem ./...
```
