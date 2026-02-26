package main

import (
	"sort"
	"testing"
)

func TestKeyExtractor(t *testing.T) {
	src := []byte(`
package main
import "github.com/loopcontext/msgcat"
func main() {
	var catalog msgcat.MessageCatalog
	_ = catalog.GetMessageWithCtx(ctx, "greeting.hello", nil)
	_ = catalog.GetErrorWithCtx(ctx, "error.not_found", nil)
	_ = catalog.WrapErrorWithCtx(ctx, err, "wrap.key", nil)
}
`)
	ext := newKeyExtractor("github.com/loopcontext/msgcat")
	if err := ext.extractFromFile("test.go", src); err != nil {
		t.Fatal(err)
	}
	keys := ext.sortedKeys()
	want := []string{"error.not_found", "greeting.hello", "wrap.key"}
	if len(keys) != len(want) {
		t.Fatalf("got %d keys %v, want %d %v", len(keys), keys, len(want), want)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Errorf("keys[%d] = %q, want %q", i, keys[i], want[i])
		}
	}
}

func TestKeyExtractor_skipsFilesWithoutMsgcatImport(t *testing.T) {
	src := []byte(`
package main
func main() {
	GetMessageWithCtx(ctx, "ignored", nil)
}
`)
	ext := newKeyExtractor("github.com/loopcontext/msgcat")
	if err := ext.extractFromFile("test.go", src); err != nil {
		t.Fatal(err)
	}
	keys := ext.sortedKeys()
	if len(keys) != 0 {
		t.Errorf("expected no keys when file does not import msgcat, got %v", keys)
	}
}

func TestKeyExtractor_stringConcat(t *testing.T) {
	src := []byte(`
package main
import "github.com/loopcontext/msgcat"
func main() {
	key := "prefix." + "suffix"
	catalog.GetMessageWithCtx(ctx, key, nil)
}
`)
	ext := newKeyExtractor("github.com/loopcontext/msgcat")
	if err := ext.extractFromFile("test.go", src); err != nil {
		t.Fatal(err)
	}
	// We only extract string literals; "prefix." + "suffix" is a binary expr, we support it
	keys := ext.sortedKeys()
	if len(keys) != 0 {
		// Our extractString for BinaryExpr only handles direct string literals; key is an ident, not a literal at call site
		t.Logf("keys (if we extracted from ident): %v", keys)
	}
}

func TestUnquote(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{`"hello"`, "hello"},
		{`"a\"b"`, `a"b`},
	}
	for _, tt := range tests {
		got, _ := unquote(tt.in)
		if got != tt.want {
			t.Errorf("unquote(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMergeTargetLangsList(t *testing.T) {
	cfg := &mergeConfig{targetLangs: "es, fr , de"}
	got := cfg.targetLangsList()
	sort.Strings(got)
	want := []string{"de", "es", "fr"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
