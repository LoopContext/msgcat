package msgcat_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

	en := []byte("default:\n  short: Unexpected error\n  long: Unexpected code {{key}}\nset:\n  greeting.hello:\n    short: msg {{name}}\n    long: count {{num:amount}} at {{date:when}} {{plural:count|item|items}}\n")
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
	f.Add("en", "greeting.hello", "abc", float64(12.5), int(1))
	f.Add("es-MX", "missing.key", "xyz", float64(-1000.25), int(2))
	f.Add("", "greeting.template", "", float64(0), int(0))

	f.Fuzz(func(t *testing.T, lang string, msgKey string, name string, n float64, count int) {
		catalog := buildFuzzCatalog(t)
		ctx := context.WithValue(context.Background(), "language", lang)
		params := msgcat.Params{"name": name, "amount": n, "when": time.Now(), "count": count, "key": msgKey}
		_ = catalog.GetMessageWithCtx(ctx, msgKey, params)
	})
}

func FuzzLoadMessages(f *testing.F) {
	f.Add("en", "sys.fuzz_9001", "short", "long")
	f.Add("pt-BR", "sys.fuzz_9002", "a", "b")
	f.Add("  es  ", "sys.fuzz_9999", "x", "y")

	f.Fuzz(func(t *testing.T, lang string, key string, shortTpl string, longTpl string) {
		catalog := buildFuzzCatalog(t)
		if !strings.HasPrefix(key, msgcat.RuntimeKeyPrefix) {
			return
		}
		_ = catalog.LoadMessages(lang, []msgcat.RawMessage{{
			Key:      key,
			ShortTpl: shortTpl,
			LongTpl:  longTpl,
		}})
	})
}
