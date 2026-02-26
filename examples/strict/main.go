// Strict demonstrates StrictTemplates: when a template parameter is missing,
// the placeholder is replaced with <missing:paramName> and observer/stats record the issue.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/loopcontext/msgcat"
)

type logObserver struct {
	mu sync.Mutex
}

func (o *logObserver) OnLanguageFallback(_, _ string) {}
func (o *logObserver) OnLanguageMissing(_ string)     {}
func (o *logObserver) OnMessageMissing(lang string, msgKey string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	fmt.Printf("[observer] missing message %s:%s\n", lang, msgKey)
}
func (o *logObserver) OnTemplateIssue(lang string, msgKey string, issue string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	fmt.Printf("[observer] template issue %s:%s: %s\n", lang, msgKey, issue)
}

func main() {
	dir, err := os.MkdirTemp("", "msgcat-strict-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	en := []byte(`default:
  short: Unexpected error
  long: Not in catalog
set:
  greeting.template:
    short: "Hello {{name}}, role {{role}}"
    long: "Hello {{name}}, you are {{role}}."
`)
	if err := os.WriteFile(filepath.Join(dir, "en.yaml"), en, 0o600); err != nil {
		panic(err)
	}

	observer := &logObserver{}
	catalog, err := msgcat.NewMessageCatalog(msgcat.Config{
		ResourcePath:    dir,
		StrictTemplates: true,
		Observer:        observer,
		ObserverBuffer:  64,
	})
	if err != nil {
		panic(err)
	}
	defer func() { _ = msgcat.Close(catalog) }()

	ctx := context.WithValue(context.Background(), "language", "en")

	// Missing param "role" => <missing:role>
	msg := catalog.GetMessageWithCtx(ctx, "greeting.template", msgcat.Params{"name": "juan"})
	fmt.Println("Missing role:", msg.ShortText)

	// All params provided
	msg = catalog.GetMessageWithCtx(ctx, "greeting.template", msgcat.Params{"name": "juan", "role": "admin"})
	fmt.Println("All params:", msg.ShortText)

	stats, _ := msgcat.SnapshotStats(catalog)
	fmt.Println("Template issues count:", len(stats.TemplateIssues))
}
