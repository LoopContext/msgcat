# msgcat Retrieval Pack

Purpose: compact, chunk-friendly reference for LLM retrieval/indexing.

## C01_IDENTITY
- Module: `github.com/loopcontext/msgcat`
- Package: `msgcat`
- Domain: i18n message catalog for Go backends/APIs.

## C02_PRIMARY_CAPABILITIES
- Load localized messages from YAML by language.
- Resolve language from `context.Context`.
- Fallback chain for missing regional/language variants.
- Render templates with positional/plural/number/date tokens.
- Wrap errors with localized short/long text and code.
- Runtime reload + runtime-loaded system messages.
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
group: 0
default:
  short: string
  long: string
set:
  <code>:
    short: string
    long: string
```
Validation:
- `group >= 0`
- default short/long: at least one non-empty
- `set` omitted => initialized empty
- each code in `set` must be `> 0`

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
Supported:
- `{{0}}`, `{{1}}`, ... positional
- `{{plural:i|singular|plural}}`
- `{{num:i}}`
- `{{date:i}}`

Processing order:
1. plural
2. number
3. date
4. positional

Limitation:
- plural branches are plain text (do not nest placeholders inside branches).

Strict mode (`StrictTemplates=true`):
- missing param index => `<missing:n>`
- template issue recorded in stats/observer.

## C07_LOCALIZATION_RULES
`{{num:i}}`:
- default: `12,345.5`
- for base lang in `{es, pt, fr, de, it}`: `12.345,5`

`{{date:i}}`:
- default: `MM/DD/YYYY`
- for base lang in `{es, pt, fr, de, it}`: `DD/MM/YYYY`

Accepted date types:
- `time.Time`
- `*time.Time`

## C08_PUBLIC_API
```go
type MessageCatalog interface {
  LoadMessages(lang string, messages []RawMessage) error
  GetMessageWithCtx(ctx context.Context, msgCode int, msgParams ...interface{}) *Message
  WrapErrorWithCtx(ctx context.Context, err error, msgCode int, msgParams ...interface{}) error
  GetErrorWithCtx(ctx context.Context, msgCode int, msgParams ...interface{}) error
}

func NewMessageCatalog(cfg Config) (MessageCatalog, error)
func Reload(catalog MessageCatalog) error
func SnapshotStats(catalog MessageCatalog) (MessageCatalogStats, error)
func ResetStats(catalog MessageCatalog) error
func Close(catalog MessageCatalog) error
```

## C09_CODES_AND_CONSTANTS
```go
const (
  SystemMessageMinCode = 9000
  SystemMessageMaxCode = 9999
  CodeMissingMessage   = 999999002
  CodeMissingLanguage  = 999999001
)
```

Semantics:
- missing message in resolved language => default language message + `CodeMissingMessage`
- missing language after full chain => `CodeMissingLanguage`

## C10_RUNTIME_LOADING
`LoadMessages(lang, messages)`:
- only allows codes in `[9000, 9999]`
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
  OnMessageMissing(lang string, msgCode int)
  OnTemplateIssue(lang string, msgCode int, issue string)
}
```

Snapshot counters:
```go
type MessageCatalogStats struct {
  LanguageFallbacks map[string]int // "requested->resolved"
  MissingLanguages  map[string]int // "lang"
  MissingMessages   map[string]int // "lang:code"
  TemplateIssues    map[string]int // "lang:code:issue"
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
- `ErrorCode()`
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
msg := catalog.GetMessageWithCtx(ctx, 2, 3, 12345.5, time.Now())
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
