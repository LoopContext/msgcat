package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMerge_preservesTargetWhenOnlyForms(t *testing.T) {
	dir := t.TempDir()
	source := []byte(`default:
  short: Err
  long: Err
set:
  person.cats:
    short_forms:
      one: "{{name}} has {{count}} cat."
      other: "{{name}} has {{count}} cats."
    long_forms:
      one: "One cat."
      other: "{{count}} cats."
    plural_param: count
`)
	sourcePath := filepath.Join(dir, "en.yaml")
	if err := os.WriteFile(sourcePath, source, 0644); err != nil {
		t.Fatal(err)
	}
	// Target (es) has only short_forms/long_forms, no short/long
	target := []byte(`default:
  short: Error
  long: Error
set:
  person.cats:
    short_forms:
      one: "{{name}} tiene {{count}} gato."
      other: "{{name}} tiene {{count}} gatos."
    long_forms:
      one: "Un gato."
      other: "{{count}} gatos."
    plural_param: count
`)
	if err := os.WriteFile(filepath.Join(dir, "es.yaml"), target, 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &mergeConfig{
		source:      sourcePath,
		targetLangs: "es",
		outdir:      dir,
	}
	if err := runMerge(cfg); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(filepath.Join(dir, "translate.es.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(out)
	// Target's translated forms should be preserved (Spanish)
	if !strings.Contains(content, "tiene") {
		t.Errorf("merge should preserve target short_forms when target has only forms; got %s", content)
	}
	if !strings.Contains(content, "gato") {
		t.Errorf("merge should preserve target forms; got %s", content)
	}
}
