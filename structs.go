package msgcat

import "time"

type ContextKey string

// Params is the type for named template parameters. Use msgcat.Params{"name": value}.
type Params map[string]interface{}

type Messages struct {
	Group   OptionalGroup         `yaml:"group,omitempty"` // Optional; int or string (e.g. group: 0 or group: "api"). Catalog does not interpret it.
	Default RawMessage            `yaml:"default"`
	Set     map[string]RawMessage `yaml:"set"`
}

// RawMessage is one catalog entry. Code is optional and can be any value the user wants (e.g. "404", "ERR_NOT_FOUND");
// it is for projects that map their own error/message codes into the catalog. Uniqueness is not enforced.
// Optional ShortForms/LongForms enable CLDR plural forms (zero, one, two, few, many, other); when set,
// the plural_param (default "count") is used to select the form. See docs/CLDR_AND_GO_MESSAGES_PLAN.md.
type RawMessage struct {
	LongTpl     string            `yaml:"long"`
	ShortTpl    string            `yaml:"short"`
	Code        OptionalCode      `yaml:"code"` // Optional. In YAML: code: 404 or code: "ERR_NOT_FOUND". Use Key when empty.
	ShortForms  map[string]string  `yaml:"short_forms,omitempty"` // Optional CLDR forms: zero, one, two, few, many, other.
	LongForms   map[string]string  `yaml:"long_forms,omitempty"`
	PluralParam string            `yaml:"plural_param,omitempty"` // Param name for plural selection (default "count").
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

// MessageDef defines a message that can be extracted to YAML via the msgcat CLI (extract -source).
// Use in Go for "messages in Go" workflow; at runtime the catalog loads from YAML. Key is required.
type MessageDef struct {
	Key         string            // Message key (e.g. "person.cats"). Required.
	Short       string            // Short template (or use ShortForms for CLDR).
	Long        string            // Long template (or use LongForms for CLDR).
	ShortForms  map[string]string  `yaml:"short_forms,omitempty"` // Optional CLDR forms: zero, one, two, few, many, other.
	LongForms   map[string]string  `yaml:"long_forms,omitempty"`
	PluralParam string            `yaml:"plural_param,omitempty"` // Param name for plural selection (default "count").
	Code        OptionalCode      `yaml:"code,omitempty"`
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
