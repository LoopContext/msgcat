// Reload demonstrates: Reload(catalog) re-reads YAML from disk. Runtime-loaded
// messages (keys with sys. prefix) are preserved. On reload failure, previous state is kept.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/loopcontext/msgcat"
)

func main() {
	dir, err := os.MkdirTemp("", "msgcat-reload-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	enPath := filepath.Join(dir, "en.yaml")
	writeYAML := func(short string) {
		body := fmt.Sprintf(`default:
  short: Unexpected error
  long: Not in catalog
set:
  greeting.hello:
    short: %s
`, short)
		if err := os.WriteFile(enPath, []byte(body), 0o600); err != nil {
			panic(err)
		}
	}

	writeYAML("Hello before reload")
	catalog, err := msgcat.NewMessageCatalog(msgcat.Config{ResourcePath: dir})
	if err != nil {
		panic(err)
	}

	ctx := context.WithValue(context.Background(), "language", "en")
	msg := catalog.GetMessageWithCtx(ctx, "greeting.hello", nil)
	fmt.Println("Before reload:", msg.ShortText)

	// Change YAML on disk
	writeYAML("Hello after reload")

	// Reload re-reads from disk
	if err := msgcat.Reload(catalog); err != nil {
		panic(err)
	}

	msg = catalog.GetMessageWithCtx(ctx, "greeting.hello", nil)
	fmt.Println("After reload:", msg.ShortText)
}
