package msgcat

// Error is the catalog error type. When ErrorCode() is empty, use ErrorKey() as the API identifier.
type Error interface {
	Error() string
	Unwrap() error
	ErrorCode() string // Optional; user-defined. Empty when not set. Use ErrorKey() when empty.
	ErrorKey() string  // Message key (e.g. "error.not_found"); use as identifier when ErrorCode() is empty.
	GetShortMessage() string
	GetLongMessage() string
}

type DefaultError struct {
	err          error
	shortMessage string
	longMessage  string
	code         string
	key          string
}

func (ce DefaultError) Error() string {
	return ce.shortMessage
}

func (ce *DefaultError) Unwrap() error {
	return ce.err
}

func (ce *DefaultError) ErrorCode() string {
	return ce.code
}

func (ce *DefaultError) ErrorKey() string {
	return ce.key
}

func (ce *DefaultError) GetShortMessage() string {
	return ce.shortMessage
}

func (ce *DefaultError) GetLongMessage() string {
	return ce.longMessage
}

func newCatalogError(code string, key string, shortMessage string, longMessage string, err error) error {
	return &DefaultError{shortMessage: shortMessage, longMessage: longMessage, code: code, key: key, err: err}
}
