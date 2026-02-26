package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractMessageDef(t *testing.T) {
	dir := t.TempDir()
	src := []byte(`
package main
import "github.com/loopcontext/msgcat"
var personCats = msgcat.MessageDef{
	Key: "person.cats",
	ShortForms: map[string]string{
		"one": "{{name}} has {{count}} cat.",
		"other": "{{name}} has {{count}} cats.",
	},
	LongForms: map[string]string{
		"one": "{{name}} has one cat.",
		"other": "{{name}} has {{count}} cats.",
	},
}
`)
	if err := os.WriteFile(filepath.Join(dir, "msgdef.go"), src, 0644); err != nil {
		t.Fatal(err)
	}
	ext := newKeyExtractor("github.com/loopcontext/msgcat")
	if err := ext.extractFromFile(filepath.Join(dir, "msgdef.go"), src); err != nil {
		t.Fatal(err)
	}
	if len(ext.defs) != 1 {
		t.Fatalf("expected 1 def, got %d", len(ext.defs))
	}
	raw, ok := ext.defs["person.cats"]
	if !ok {
		t.Fatal("expected person.cats in defs")
	}
	if len(raw.ShortForms) != 2 {
		t.Errorf("ShortForms: got %d entries", len(raw.ShortForms))
	}
	if raw.ShortForms["one"] != "{{name}} has {{count}} cat." {
		t.Errorf("ShortForms[one]: got %q", raw.ShortForms["one"])
	}
	if raw.ShortForms["other"] != "{{name}} has {{count}} cats." {
		t.Errorf("ShortForms[other]: got %q", raw.ShortForms["other"])
	}
	if _, ok := ext.keys["person.cats"]; !ok {
		t.Error("expected person.cats in keys")
	}
}

func TestExtractMessageDef_syncToYAML(t *testing.T) {
	dir := t.TempDir()
	enYaml := []byte(`default:
  short: Err
  long: Err
set:
  greeting.hello:
    short: Hello
    long: Hello there
`)
	enPath := filepath.Join(dir, "en.yaml")
	if err := os.WriteFile(enPath, enYaml, 0644); err != nil {
		t.Fatal(err)
	}
	goSrc := []byte(`
package p
import "github.com/loopcontext/msgcat"
var _ = msgcat.MessageDef{Key: "person.cats", Short: "Cats", Long: "Cats count"}
`)
	goPath := filepath.Join(dir, "p.go")
	if err := os.WriteFile(goPath, goSrc, 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &extractConfig{source: enPath, out: filepath.Join(dir, "en_out.yaml")}
	ext := newKeyExtractor("github.com/loopcontext/msgcat")
	if err := ext.extractFromFile(goPath, goSrc); err != nil {
		t.Fatal(err)
	}
	keys := ext.sortedKeys()
	if err := runExtractSync(cfg, keys, ext.defs); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(cfg.out)
	if err != nil {
		t.Fatal(err)
	}
	content := string(out)
	if !strings.Contains(content, "person.cats") {
		t.Errorf("expected person.cats in output: %s", content)
	}
	if !strings.Contains(content, "Cats") {
		t.Errorf("expected Short/Long in output: %s", content)
	}
}
