package msgcat_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/loopcontext/msgcat"
)

func buildFuzzCatalog(t *testing.T) msgcat.MessageCatalog {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "msgcat-fuzz-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	en := []byte("group: 0\ndefault:\n  short: Unexpected error\n  long: Unexpected code {{0}}\nset:\n  1:\n    short: msg {{0}}\n    long: count {{num:1}} at {{date:2}} {{plural:3|item|items}}\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "en.yaml"), en, 0o600); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	catalog, err := msgcat.NewMessageCatalog(msgcat.Config{
		ResourcePath:    tmpDir,
		StrictTemplates: true,
		StatsMaxKeys:    64,
	})
	if err != nil {
		t.Fatalf("failed to create catalog: %v", err)
	}
	t.Cleanup(func() { _ = msgcat.Close(catalog) })
	return catalog
}

func FuzzGetMessageWithCtx(f *testing.F) {
	f.Add("en", 1, "abc", float64(12.5), int(1))
	f.Add("es-MX", 404, "xyz", float64(-1000.25), int(2))
	f.Add("", 2, "", float64(0), int(0))

	f.Fuzz(func(t *testing.T, lang string, code int, name string, n float64, count int) {
		catalog := buildFuzzCatalog(t)
		ctx := context.WithValue(context.Background(), "language", lang)
		_ = catalog.GetMessageWithCtx(ctx, code, name, n, time.Now(), count)
	})
}

func FuzzLoadMessages(f *testing.F) {
	f.Add("en", 9001, "short", "long")
	f.Add("pt-BR", 9002, "a", "b")
	f.Add("  es  ", 9999, "x", "y")

	f.Fuzz(func(t *testing.T, lang string, code int, shortTpl string, longTpl string) {
		catalog := buildFuzzCatalog(t)
		if code < 9000 || code > 9999 {
			return
		}
		_ = catalog.LoadMessages(lang, []msgcat.RawMessage{{
			Code:     code,
			ShortTpl: shortTpl,
			LongTpl:  longTpl,
		}})
	})
}
