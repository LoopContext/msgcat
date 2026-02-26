package main

import (
	"fmt"
	"os"

	"github.com/loopcontext/msgcat"
	"gopkg.in/yaml.v2"
)

// runExtractSync reads the source YAML, merges in keys (empty short/long if missing) and
// defs (MessageDef content from Go), preserves group and default, writes to cfg.out.
func runExtractSync(cfg *extractConfig, keys []string, defs map[string]msgcat.RawMessage) error {
	src, err := os.ReadFile(cfg.source)
	if err != nil {
		return fmt.Errorf("read source %s: %w", cfg.source, err)
	}
	var m msgcat.Messages
	if err := yaml.Unmarshal(src, &m); err != nil {
		return fmt.Errorf("parse source YAML: %w", err)
	}
	if m.Set == nil {
		m.Set = make(map[string]msgcat.RawMessage)
	}
	added := 0
	// MessageDef from Go: add or overwrite by key
	for key, raw := range defs {
		if key == "" {
			continue
		}
		m.Set[key] = raw
		added++
	}
	// Keys from API calls: add if missing (empty short/long)
	for _, key := range keys {
		if key == "" {
			continue
		}
		if _, exists := m.Set[key]; exists {
			continue
		}
		m.Set[key] = msgcat.RawMessage{ShortTpl: "", LongTpl: ""}
		added++
	}
	outPath := cfg.out
	if outPath == "" {
		outPath = cfg.source
	}
	out, err := yaml.Marshal(&m)
	if err != nil {
		return fmt.Errorf("marshal YAML: %w", err)
	}
	if err := os.WriteFile(outPath, out, 0644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	if added > 0 {
		fmt.Fprintf(os.Stderr, "msgcat: added %d key(s) to %s\n", added, outPath)
	}
	return nil
}
