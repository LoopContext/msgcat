// Load_messages demonstrates runtime loading via LoadMessages: keys must have the sys. prefix.
// Loaded messages survive Reload and are merged with YAML-loaded messages per language.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/loopcontext/msgcat"
)

func main() {
	dir, err := os.MkdirTemp("", "msgcat-load-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	en := []byte(`default:
  short: Unexpected error
  long: Not found in catalog
set:
  greeting.hello:
    short: Hello from YAML
`)
	if err := os.WriteFile(filepath.Join(dir, "en.yaml"), en, 0o600); err != nil {
		panic(err)
	}

	catalog, err := msgcat.NewMessageCatalog(msgcat.Config{ResourcePath: dir})
	if err != nil {
		panic(err)
	}

	// Load runtime-only message; Key must have prefix sys.
	err = catalog.LoadMessages("en", []msgcat.RawMessage{
		{
			Key:      "sys.maintenance",
			ShortTpl: "Under maintenance",
			LongTpl:  "Back in {{minutes}} minutes.",
			Code:     msgcat.CodeString("ERR_MAINTENANCE"),
		},
	})
	if err != nil {
		panic(err)
	}

	ctx := context.WithValue(context.Background(), "language", "en")

	// Use YAML-loaded key
	msg := catalog.GetMessageWithCtx(ctx, "greeting.hello", nil)
	fmt.Println("YAML key:", msg.ShortText)

	// Use runtime-loaded key with params
	msg = catalog.GetMessageWithCtx(ctx, "sys.maintenance", msgcat.Params{"minutes": 5})
	fmt.Println("sys. key:", msg.ShortText, "| long:", msg.LongText, "| code:", msg.Code)
}
