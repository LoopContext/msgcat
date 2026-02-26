# msgcat Retrieval Pack

Purpose: compact, chunk-friendly reference for LLM retrieval/indexing.

Runnable examples: `examples/basic`, `examples/load_messages`, `examples/reload`, `examples/strict`, `examples/stats`, `examples/http`, `examples/metrics`.

## C01_IDENTITY
- Module: `github.com/loopcontext/msgcat`
- Package: `msgcat`
- Domain: i18n message catalog for Go backends/APIs.

## C02_PRIMARY_CAPABILITIES
- Load localized messages from YAML by language.
- Resolve language from `context.Context`.
- Fallback chain for missing regional/language variants.
- Render templates with named parameters (plural/number/date tokens).
- Wrap errors with localized short/long text and code.
- Runtime reload + runtime-loaded messages (key prefix `sys.`).
- Observability hooks + in-memory counters.
- Concurrency-safe operations.

## C03_CONFIG
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
Defaults:
- `ResourcePath`: `./resources/messages`
- `CtxLanguageKey`: `"language"`
- `DefaultLanguage`: `"en"`
- `ObserverBuffer`: `1024`
- `StatsMaxKeys`: `512`
- `ReloadRetries`: `0`
- `ReloadRetryDelay`: `50ms`
- `NowFn`: `time.Now`

## C04_YAML_SCHEMA
```yaml
default:
  short: string
  long: string
set:
  <key>:   # e.g. greeting.hello, error.not_found
    code: int    # optional
    short: string
    long: string
```
Keys: `[a-zA-Z0-9_.-]+`. Templates use named params: `{{name}}`, `{{plural:count|a|b}}`, `{{num:amount}}`, `{{date:when}}`.
Validation:
- default short/long: at least one non-empty
- `set` omitted => initialized empty
- each key non-empty and valid format

## C05_LANGUAGE_RESOLUTION
Input language normalization:
- lowercase
- `_` replaced by `-`

Resolution order:
1. requested language (e.g. `es-ar`)
2. base language (e.g. `es`)
3. `FallbackLanguages` in order
4. `DefaultLanguage`
5. hard fallback `en`

If no language found:
- code: `CodeMissingLanguage`
- text: `MessageCatalogNotFound`

## C06_TEMPLATE_TOKENS
Supported (named):
- `{{name}}`
- `{{plural:count|singular|plural}}`
- `{{num:amount}}`
- `{{date:when}}`

Pass via `msgcat.Params{"name": value, ...}`. Processing order: plural, number, date, simple.

Limitation:
- plural branches are plain text (do not nest placeholders inside branches).

Strict mode (`StrictTemplates=true`):
- missing param => `<missing:paramName>`
- template issue recorded in stats/observer.

## C07_LOCALIZATION_RULES
`{{num:name}}`:
- default: `12,345.5`
- for base lang in `{es, pt, fr, de, it}`: `12.345,5`

`{{date:name}}`:
- default: `MM/DD/YYYY`
- for base lang in `{es, pt, fr, de, it}`: `DD/MM/YYYY`

Accepted date types:
- `time.Time`
- `*time.Time`

## C08_PUBLIC_API
```go
type Params map[string]interface{}

type MessageCatalog interface {
  LoadMessages(lang string, messages []RawMessage) error
  GetMessageWithCtx(ctx context.Context, msgKey string, params Params) *Message
  WrapErrorWithCtx(ctx context.Context, err error, msgKey string, params Params) error
  GetErrorWithCtx(ctx context.Context, msgKey string, params Params) error
}
```

LoadMessages: each RawMessage must have Key with prefix `sys.`.

```go
func NewMessageCatalog(cfg Config) (MessageCatalog, error)
func Reload(catalog MessageCatalog) error
func SnapshotStats(catalog MessageCatalog) (MessageCatalogStats, error)
func ResetStats(catalog MessageCatalog) error
func Close(catalog MessageCatalog) error
```

## C09_CODES_AND_CONSTANTS
```go
const (
  RuntimeKeyPrefix     = "sys."
  CodeMissingMessage  = "msgcat.missing_message"
  CodeMissingLanguage = "msgcat.missing_language"
)
```

## C09a_MESSAGE_AND_ERROR_CODES
Optional `code` field: for projects that already have error/message codes (HTTP status, legacy numbers, string ids like `ERR_NOT_FOUND`). Store that value in the catalog; it is returned in `Message.Code` and `ErrorCode()` for your API to expose unchanged.

- **Optional** — Omit when not needed; use `Message.Key` or `ErrorKey()` when empty.
- **Any value** — String. YAML: `code: 404` or `code: "ERR_NOT_FOUND"`. Go: `msgcat.CodeInt(503)`, `msgcat.CodeString("ERR_MAINT")`.
- **Not unique** — Same code can be used on multiple messages.
- **Opaque** — Catalog only stores and returns it; meaning is up to the caller.

Semantics:
- missing message => default message + `CodeMissingMessage`
- missing language => `CodeMissingLanguage`

## C10_RUNTIME_LOADING
`LoadMessages(lang, messages)`:
- each RawMessage.Key must have prefix `sys.`
- rejects duplicates per language
- messages persist across `Reload`

## C11_RELOAD
`Reload(catalog)`:
- re-reads YAML from `ResourcePath`
- re-validates and normalizes
- merges/preserves runtime-loaded messages
- retries according to `ReloadRetries` + `ReloadRetryDelay`

## C12_OBSERVABILITY
Observer hooks:
```go
type Observer interface {
  OnLanguageFallback(requestedLang, resolvedLang string)
  OnLanguageMissing(lang string)
  OnMessageMissing(lang string, msgKey string)
  OnTemplateIssue(lang string, msgKey string, issue string)
}
```

Snapshot counters:
```go
type MessageCatalogStats struct {
  LanguageFallbacks map[string]int // "requested->resolved"
  MissingLanguages  map[string]int // "lang"
  MissingMessages   map[string]int // "lang:msgKey"
  TemplateIssues    map[string]int // "lang:msgKey:issue"
  DroppedEvents     map[string]int // internal drop counters
  LastReloadAt      time.Time
}
```

## C13_CONCURRENCY
Concurrency safety via RW mutex:
- Safe concurrent reads (`GetMessageWithCtx`, error helpers)
- Safe concurrent writes (`LoadMessages`, `Reload`)
- Safe stat snapshots
- Observer callbacks run asynchronously and are panic-protected.
- `Reload` failure keeps last in-memory state intact.

Validated with `go test -race ./...`.

## C14_ERROR_MODEL
Returned catalog error supports:
- `Error()` => short localized text
- `Unwrap()` => wrapped original error
- `ErrorCode() string` => optional; empty when not set. Use `ErrorKey()` when empty.
- `ErrorKey()` => message key; use as API identifier when Code is empty
- `GetShortMessage()`
- `GetLongMessage()`

## C15_CANONICAL_SNIPPET
```go
catalog, _ := msgcat.NewMessageCatalog(msgcat.Config{
  ResourcePath:      "./resources/messages",
  CtxLanguageKey:    "language",
  DefaultLanguage:   "en",
  FallbackLanguages: []string{"es"},
  StrictTemplates:   true,
})

ctx := context.WithValue(context.Background(), "language", "es-MX")
params := msgcat.Params{"count": 3, "amount": 12345.5, "when": time.Now()}
msg := catalog.GetMessageWithCtx(ctx, "items.count", params)
_ = msg.ShortText
_ = msg.LongText

stats, _ := msgcat.SnapshotStats(catalog)
_ = stats
```

## C16_CI_COMMANDS
```bash
go test ./...
go test -race ./...
go test -run ^$ -bench . -benchmem ./...
```

## C17_EXAMPLE_NEW_CATALOG_MINIMAL
```go
catalog, err := msgcat.NewMessageCatalog(msgcat.Config{})
```

## C18_EXAMPLE_NEW_CATALOG_FULL
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

## C19_EXAMPLE_GET_MESSAGE_NIL_PARAMS
```go
msg := catalog.GetMessageWithCtx(ctx, "greeting.hello", nil)
fmt.Println(msg.ShortText, msg.Code)
```

## C20_EXAMPLE_GET_MESSAGE_WITH_PARAMS
```go
msg := catalog.GetMessageWithCtx(ctx, "greeting.template", msgcat.Params{
  "name": "juan", "detail": "admin",
})
```

## C21_EXAMPLE_TEMPLATE_SIMPLE
```go
// YAML: short: "Hello {{name}}"
msg := catalog.GetMessageWithCtx(ctx, "greeting.hello", msgcat.Params{"name": "juan"})
```

## C22_EXAMPLE_TEMPLATE_PLURAL
```go
// YAML: short: "You have {{count}} {{plural:count|item|items}}"
msg := catalog.GetMessageWithCtx(ctx, "items.count", msgcat.Params{"count": 3})
```

## C23_EXAMPLE_TEMPLATE_NUMBER
```go
// YAML: short: "Total: {{num:amount}}"
msg := catalog.GetMessageWithCtx(ctx, "report.total", msgcat.Params{"amount": 12345.67})
```

## C24_EXAMPLE_TEMPLATE_DATE
```go
// YAML: short: "At {{date:when}}"
msg := catalog.GetMessageWithCtx(ctx, "report.generated", msgcat.Params{"when": time.Now()})
```

## C25_EXAMPLE_GET_ERROR
```go
err := catalog.GetErrorWithCtx(ctx, "error.not_found", msgcat.Params{"resource": "order"})
```

## C26_EXAMPLE_WRAP_ERROR
```go
inner := errors.New("db timeout")
err := catalog.WrapErrorWithCtx(ctx, inner, "error.timeout", nil)
var catErr msgcat.Error
if errors.As(err, &catErr) {
  code := catErr.ErrorCode()   // "" if no code in catalog
  key := catErr.ErrorKey()     // use as API id when code is empty
  catErr.GetShortMessage()
  catErr.GetLongMessage()
  catErr.Unwrap()
}
```

## C27_EXAMPLE_LOAD_MESSAGES
```go
err := catalog.LoadMessages("en", []msgcat.RawMessage{{
  Key:      "sys.maintenance",
  ShortTpl: "Under maintenance",
  LongTpl:  "Back in {{minutes}} minutes.",
  Code:     503,
}})
msg := catalog.GetMessageWithCtx(ctx, "sys.maintenance", msgcat.Params{"minutes": 5})
```

## C28_EXAMPLE_RELOAD
```go
err := msgcat.Reload(catalog)
```

## C29_EXAMPLE_STATS
```go
stats, _ := msgcat.SnapshotStats(catalog)
_ = stats.MissingMessages
msgcat.ResetStats(catalog)
```

## C30_EXAMPLE_CLOSE
```go
defer func() { _ = msgcat.Close(catalog) }()
```

## C31_EXAMPLE_OBSERVER
```go
type obs struct{}
func (obs) OnLanguageFallback(req, res string) {}
func (obs) OnLanguageMissing(lang string)       {}
func (obs) OnMessageMissing(lang string, msgKey string) {}
func (obs) OnTemplateIssue(lang string, msgKey string, issue string) {}
catalog, _ := msgcat.NewMessageCatalog(msgcat.Config{Observer: obs{}, ObserverBuffer: 1024})
```

## C32_EXAMPLE_CONTEXT_LANGUAGE
```go
ctx = context.WithValue(ctx, msgcat.ContextKey("language"), "es-MX")
ctx = context.WithValue(ctx, "language", "es-MX")
```

## C33_EXAMPLE_MISSING_MESSAGE
```go
msg := catalog.GetMessageWithCtx(ctx, "nonexistent.key", nil)
// msg.Code == msgcat.CodeMissingMessage
```

## C34_EXAMPLE_MISSING_LANGUAGE
```go
ctx = context.WithValue(ctx, "language", "zz")
msg := catalog.GetMessageWithCtx(ctx, "greeting.hello", nil)
// msg.Code == msgcat.CodeMissingLanguage
```
