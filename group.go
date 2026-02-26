package msgcat

import (
	"fmt"
	"strconv"
)

// OptionalGroup is the optional "group" field for message files. Use it to tag a file (or later, an entry)
// with a group that can be an integer or a string (e.g. group: 0 or group: "api") for organization or tooling.
// The catalog does not interpret group; it is only stored. YAML accepts int or string.
type OptionalGroup string

// UnmarshalYAML allows group to be given as int or string in YAML.
func (g *OptionalGroup) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v interface{}
	if err := unmarshal(&v); err != nil {
		return err
	}
	if v == nil {
		*g = ""
		return nil
	}
	switch t := v.(type) {
	case string:
		*g = OptionalGroup(t)
		return nil
	case int:
		*g = OptionalGroup(strconv.Itoa(t))
		return nil
	case int64:
		*g = OptionalGroup(strconv.FormatInt(t, 10))
		return nil
	default:
		return fmt.Errorf("group must be string or int, got %T", v)
	}
}

// MarshalYAML emits int when the value is numeric, otherwise string, for readable round-trip (group: 0 vs group: "api").
func (g OptionalGroup) MarshalYAML() (interface{}, error) {
	s := string(g)
	if s == "" {
		return nil, nil
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n, nil
	}
	return s, nil
}
