// Msgdef demonstrates defining messages in Go with msgcat.MessageDef. Run from repo root:
//
//	msgcat extract -source resources/messages/en.yaml -out resources/messages/en.yaml .
//
// to sync these definitions into your source YAML (add/update entries by Key).
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/loopcontext/msgcat"
)

// Message definitions in Go; msgcat extract will find these and merge into source YAML.
var (
	personCats = msgcat.MessageDef{
		Key: "person.cats",
		ShortForms: map[string]string{
			"one":  "{{name}} has {{count}} cat.",
			"other": "{{name}} has {{count}} cats.",
		},
		LongForms: map[string]string{
			"one":  "{{name}} has one cat.",
			"other": "{{name}} has {{count}} cats.",
		},
	}
	itemsCount = msgcat.MessageDef{
		Key:   "items.count",
		Short: "{{count}} items",
		Long:  "Total: {{count}} items",
		Code:  msgcat.CodeInt(200),
	}
)

func main() {
	// Use a temp dir with en.yaml that includes the same keys (e.g. after running extract)
	dir, err := os.MkdirTemp("", "msgcat-msgdef-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	en := []byte(`default:
  short: Error
  long: Error
set:
  person.cats:
    short_forms:
      one: "{{name}} has {{count}} cat."
      other: "{{name}} has {{count}} cats."
    long_forms:
      one: "{{name}} has one cat."
      other: "{{name}} has {{count}} cats."
  items.count:
    short: "{{count}} items"
    long: "Total: {{count}} items"
    code: 200
`)
	if err := os.WriteFile(filepath.Join(dir, "en.yaml"), en, 0o600); err != nil {
		panic(err)
	}

	catalog, err := msgcat.NewMessageCatalog(msgcat.Config{ResourcePath: dir})
	if err != nil {
		panic(err)
	}

	ctx := context.WithValue(context.Background(), "language", "en")
	_ = personCats
	_ = itemsCount

	msg := catalog.GetMessageWithCtx(ctx, "person.cats", msgcat.Params{"name": "Alice", "count": 1})
	fmt.Println("person.cats (count=1):", msg.ShortText)
	msg = catalog.GetMessageWithCtx(ctx, "items.count", msgcat.Params{"count": 3})
	fmt.Println("items.count:", msg.ShortText)
}
