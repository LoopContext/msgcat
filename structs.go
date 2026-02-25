package msgcat

import "time"

type ContextKey string

type Messages struct {
	Group   int                `yaml:"group"`
	Default RawMessage         `yaml:"default"`
	Set     map[int]RawMessage `yaml:"set"`
}

type RawMessage struct {
	LongTpl  string `yaml:"long"`
	ShortTpl string `yaml:"short"`
	Code     int
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
	OnMessageMissing(lang string, msgCode int)
	OnTemplateIssue(lang string, msgCode int, issue string)
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
