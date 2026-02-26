package msgcat_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/loopcontext/msgcat"
)

func makeBenchCatalog(b *testing.B) msgcat.MessageCatalog {
	b.Helper()
	tmpDir, err := os.MkdirTemp("", "msgcat-bench-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	b.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	en := []byte("default:\n  short: Unexpected error\n  long: Unexpected message code [{{key}}]\nset:\n  greeting.hello:\n    short: Hello {{name}}\n    long: Number {{num:amount}} at {{date:when}}\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "en.yaml"), en, 0o600); err != nil {
		b.Fatalf("failed to write fixture: %v", err)
	}

	catalog, err := msgcat.NewMessageCatalog(msgcat.Config{ResourcePath: tmpDir})
	if err != nil {
		b.Fatalf("failed to create catalog: %v", err)
	}
	b.Cleanup(func() { _ = msgcat.Close(catalog) })

	return catalog
}

type noopObserver struct{}

func (noopObserver) OnLanguageFallback(requestedLang string, resolvedLang string) {}
func (noopObserver) OnLanguageMissing(lang string)                                {}
func (noopObserver) OnMessageMissing(lang string, msgKey string)                  {}
func (noopObserver) OnTemplateIssue(lang string, msgKey string, issue string)   {}

func BenchmarkGetMessageWithCtx(b *testing.B) {
	catalog := makeBenchCatalog(b)
	ctx := context.WithValue(context.Background(), "language", "en")
	date := time.Date(2026, time.January, 5, 12, 0, 0, 0, time.UTC)
	params := msgcat.Params{"name": "world", "amount": 12345.67, "when": date}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = catalog.GetMessageWithCtx(ctx, "greeting.hello", params)
	}
}

func BenchmarkGetErrorWithCtx(b *testing.B) {
	catalog := makeBenchCatalog(b)
	ctx := context.WithValue(context.Background(), "language", "en")
	date := time.Date(2026, time.January, 5, 12, 0, 0, 0, time.UTC)
	params := msgcat.Params{"name": "world", "amount": 12345.67, "when": date}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = catalog.GetErrorWithCtx(ctx, "greeting.hello", params)
	}
}

func BenchmarkGetMessageWithCtxStrictOff(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "msgcat-bench-strict-off-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	b.Cleanup(func() { _ = os.RemoveAll(tmpDir) })
	en := []byte("default:\n  short: Unexpected error\n  long: Unexpected message code [{{key}}]\nset:\n  greeting.hello:\n    short: Hello {{name}}\n    long: Number {{num:amount}} at {{date:when}}\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "en.yaml"), en, 0o600); err != nil {
		b.Fatalf("failed to write fixture: %v", err)
	}
	catalog, err := msgcat.NewMessageCatalog(msgcat.Config{ResourcePath: tmpDir, StrictTemplates: false})
	if err != nil {
		b.Fatalf("failed to create catalog: %v", err)
	}
	b.Cleanup(func() { _ = msgcat.Close(catalog) })

	ctx := context.WithValue(context.Background(), "language", "en")
	date := time.Date(2026, time.January, 5, 12, 0, 0, 0, time.UTC)
	params := msgcat.Params{"name": "world", "amount": 12345.67, "when": date}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = catalog.GetMessageWithCtx(ctx, "greeting.hello", params)
	}
}

func BenchmarkGetMessageWithCtxStrictOn(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "msgcat-bench-strict-on-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	b.Cleanup(func() { _ = os.RemoveAll(tmpDir) })
	en := []byte("default:\n  short: Unexpected error\n  long: Unexpected message code [{{key}}]\nset:\n  greeting.hello:\n    short: Hello {{name}}\n    long: Number {{num:amount}} at {{date:when}}\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "en.yaml"), en, 0o600); err != nil {
		b.Fatalf("failed to write fixture: %v", err)
	}
	catalog, err := msgcat.NewMessageCatalog(msgcat.Config{ResourcePath: tmpDir, StrictTemplates: true})
	if err != nil {
		b.Fatalf("failed to create catalog: %v", err)
	}
	b.Cleanup(func() { _ = msgcat.Close(catalog) })

	ctx := context.WithValue(context.Background(), "language", "en")
	date := time.Date(2026, time.January, 5, 12, 0, 0, 0, time.UTC)
	params := msgcat.Params{"name": "world", "amount": 12345.67, "when": date}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = catalog.GetMessageWithCtx(ctx, "greeting.hello", params)
	}
}

func BenchmarkGetMessageWithCtxFallbackChain(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "msgcat-bench-fallback-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	b.Cleanup(func() { _ = os.RemoveAll(tmpDir) })
	es := []byte("default:\n  short: Error inesperado\n  long: Error {{key}}\nset:\n  greeting.hello:\n    short: Hola {{name}}\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "es.yaml"), es, 0o600); err != nil {
		b.Fatalf("failed to write fixture: %v", err)
	}
	catalog, err := msgcat.NewMessageCatalog(msgcat.Config{
		ResourcePath:      tmpDir,
		DefaultLanguage:   "en",
		FallbackLanguages: []string{"es"},
	})
	if err != nil {
		b.Fatalf("failed to create catalog: %v", err)
	}
	b.Cleanup(func() { _ = msgcat.Close(catalog) })

	ctx := context.WithValue(context.Background(), "language", "es-MX")
	params := msgcat.Params{"name": "world"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = catalog.GetMessageWithCtx(ctx, "greeting.hello", params)
	}
}

func BenchmarkGetMessageWithCtxObserverEnabled(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "msgcat-bench-observer-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	b.Cleanup(func() { _ = os.RemoveAll(tmpDir) })
	en := []byte("default:\n  short: Unexpected error\n  long: Unexpected message code [{{key}}]\nset:\n  greeting.hello:\n    short: Hello {{name}}\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "en.yaml"), en, 0o600); err != nil {
		b.Fatalf("failed to write fixture: %v", err)
	}
	catalog, err := msgcat.NewMessageCatalog(msgcat.Config{
		ResourcePath:   tmpDir,
		Observer:       noopObserver{},
		ObserverBuffer: 1024,
	})
	if err != nil {
		b.Fatalf("failed to create catalog: %v", err)
	}
	b.Cleanup(func() { _ = msgcat.Close(catalog) })

	ctx := context.WithValue(context.Background(), "language", "en")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = catalog.GetMessageWithCtx(ctx, "missing.key", nil)
	}
}
