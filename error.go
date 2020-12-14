package msgcat

type Error interface {
	Error() string
	Unwrap() error
	ErrorCode() int
	GetShortMessage() string
	GetLongMessage() string
}

type DefaultError struct {
	err          error
	shortMessage string
	longMessage  string
	code         int
}

func (ce DefaultError) Error() string {
	return ce.shortMessage
}

func (ce *DefaultError) Unwrap() error {
	return ce.err
}

func (ce *DefaultError) ErrorCode() int {
	return ce.code
}

func (ce *DefaultError) GetShortMessage() string {
	return ce.shortMessage
}

func (ce *DefaultError) GetLongMessage() string {
	return ce.longMessage
}

func newCatalogError(code int, shortMessage string, longMessage string, err error) error {
	return &DefaultError{shortMessage: shortMessage, longMessage: longMessage, code: code, err: err}
}
