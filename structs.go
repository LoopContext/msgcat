package msgcat

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

type Config struct {
	ResourcePath   string
	CtxLanguageKey string
}
