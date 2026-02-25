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

	en := []byte("group: 0\ndefault:\n  short: Unexpected error\n  long: Unexpected message code [{{0}}]\nset:\n  1:\n    short: Hello {{0}}\n    long: Number {{num:1}} at {{date:2}}\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "en.yaml"), en, 0o600); err != nil {
		b.Fatalf("failed to write fixture: %v", err)
	}

	catalog, err := msgcat.NewMessageCatalog(msgcat.Config{ResourcePath: tmpDir})
	if err != nil {
		b.Fatalf("failed to create catalog: %v", err)
	}

	return catalog
}

func BenchmarkGetMessageWithCtx(b *testing.B) {
	catalog := makeBenchCatalog(b)
	ctx := context.WithValue(context.Background(), "language", "en")
	date := time.Date(2026, time.January, 5, 12, 0, 0, 0, time.UTC)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = catalog.GetMessageWithCtx(ctx, 1, "world", 12345.67, date)
	}
}

func BenchmarkGetErrorWithCtx(b *testing.B) {
	catalog := makeBenchCatalog(b)
	ctx := context.WithValue(context.Background(), "language", "en")
	date := time.Date(2026, time.January, 5, 12, 0, 0, 0, time.UTC)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = catalog.GetErrorWithCtx(ctx, 1, "world", 12345.67, date)
	}
}
