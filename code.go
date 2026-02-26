package msgcat

import (
	"fmt"
	"strconv"
)

// OptionalCode is the optional "code" field for catalog entries. Use it when your project
// already has error or message codes (HTTP statuses, legacy numbers, string ids like "ERR_NOT_FOUND")
// and you want to store that value in the catalog and return it from Message.Code and ErrorCode().
// It can be any value; uniqueness is not enforced. YAML accepts int or string (e.g. code: 404
// or code: "ERR_NOT_FOUND"). In Go use CodeInt or CodeString when building RawMessage.
type OptionalCode string

// UnmarshalYAML allows code to be given as int or string in YAML.
func (c *OptionalCode) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v interface{}
	if err := unmarshal(&v); err != nil {
		return err
	}
	if v == nil {
		*c = ""
		return nil
	}
	switch t := v.(type) {
	case string:
		*c = OptionalCode(t)
		return nil
	case int:
		*c = OptionalCode(strconv.Itoa(t))
		return nil
	case int64:
		*c = OptionalCode(strconv.FormatInt(t, 10))
		return nil
	default:
		return fmt.Errorf("code must be string or int, got %T", v)
	}
}

// CodeInt returns an OptionalCode from an int (e.g. HTTP status 503). Use when building RawMessage in code.
func CodeInt(i int) OptionalCode {
	return OptionalCode(strconv.Itoa(i))
}

// CodeString returns an OptionalCode from a string (e.g. "ERR_NOT_FOUND"). Use when building RawMessage in code.
func CodeString(s string) OptionalCode {
	return OptionalCode(s)
}
