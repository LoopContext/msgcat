// Stats demonstrates SnapshotStats, ResetStats, and stat keys (LanguageFallbacks,
// MissingLanguages, MissingMessages, TemplateIssues, DroppedEvents, LastReloadAt).
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/loopcontext/msgcat"
)

func main() {
	dir, err := os.MkdirTemp("", "msgcat-stats-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	en := []byte(`default:
  short: Unexpected error
  long: Not in catalog
set:
  greeting.hello:
    short: Hello
`)
	if err := os.WriteFile(filepath.Join(dir, "en.yaml"), en, 0o600); err != nil {
		panic(err)
	}

	catalog, err := msgcat.NewMessageCatalog(msgcat.Config{
		ResourcePath: dir,
		StatsMaxKeys: 16,
	})
	if err != nil {
		panic(err)
	}

	ctx := context.WithValue(context.Background(), "language", "en")

	_ = catalog.GetMessageWithCtx(ctx, "greeting.hello", nil)
	_ = catalog.GetMessageWithCtx(ctx, "missing.key", nil)

	stats, err := msgcat.SnapshotStats(catalog)
	if err != nil {
		panic(err)
	}
	fmt.Println("LanguageFallbacks:", stats.LanguageFallbacks)
	fmt.Println("MissingMessages:", stats.MissingMessages)
	fmt.Println("LastReloadAt:", stats.LastReloadAt.Round(time.Second))

	_ = msgcat.ResetStats(catalog)
	stats, _ = msgcat.SnapshotStats(catalog)
	fmt.Println("After ResetStats - MissingMessages:", stats.MissingMessages)
}
