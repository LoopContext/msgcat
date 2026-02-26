package msgcat

import "time"

type ContextKey string

// Params is the type for named template parameters. Use msgcat.Params{"name": value}.
type Params map[string]interface{}

type Messages struct {
	Default RawMessage              `yaml:"default"`
	Set     map[string]RawMessage   `yaml:"set"`
}

// RawMessage is one catalog entry. Code is optional and can be any value the user wants (e.g. "404", "ERR_NOT_FOUND");
// it is for projects that map their own error/message codes into the catalog. Uniqueness is not enforced.
type RawMessage struct {
	LongTpl  string       `yaml:"long"`
	ShortTpl string       `yaml:"short"`
	Code     OptionalCode `yaml:"code"` // Optional. In YAML: code: 404 or code: "ERR_NOT_FOUND". Use Key when empty.
	// Key is set when loading via LoadMessages (runtime); YAML uses the map key as the message key.
	Key string `yaml:"-"`
}

// Message is the resolved message for a request. Key is always the message key used for lookup.
// Code is optional (from catalog); when empty, use Key as the API identifier (e.g. in JSON responses).
type Message struct {
	LongText  string
	ShortText string
	Code      string // Optional; user-defined (e.g. "404", "ERR_001"). Empty when not set. Use Key when empty.
	Key       string // Message key (e.g. "greeting.hello"); set when found or when missing (requested key).
}

type MessageCatalogStats struct {
	LanguageFallbacks map[string]int
	MissingLanguages  map[string]int
	MissingMessages   map[string]int
	TemplateIssues    map[string]int
	DroppedEvents     map[string]int
	LastReloadAt      time.Time
}

type Observer interface {
	OnLanguageFallback(requestedLang string, resolvedLang string)
	OnLanguageMissing(lang string)
	OnMessageMissing(lang string, msgKey string)
	OnTemplateIssue(lang string, msgKey string, issue string)
}

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
