package msgcat

import "time"

type ContextKey string

// Params is the type for named template parameters. Use msgcat.Params{"name": value}.
type Params map[string]interface{}

type Messages struct {
	Default RawMessage              `yaml:"default"`
	Set     map[string]RawMessage   `yaml:"set"`
}

type RawMessage struct {
	LongTpl  string `yaml:"long"`
	ShortTpl string `yaml:"short"`
	Code     int    `yaml:"code"`
	// Key is set when loading via LoadMessages (runtime); YAML uses the map key as the message key.
	Key string `yaml:"-"`
}

type Message struct {
	LongText  string
	ShortText string
	Code      int
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
