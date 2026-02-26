// Cldr_plural demonstrates CLDR plural forms (short_forms/long_forms) in the message catalog.
// Use short_forms and long_forms with keys "zero", "one", "two", "few", "many", "other" for
// languages that need more than binary singular/plural (e.g. Arabic, Russian).
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/loopcontext/msgcat"
)

func main() {
	dir, err := os.MkdirTemp("", "msgcat-cldr-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	en := []byte(`default:
  short: Unexpected error
  long: Message not found
set:
  person.cats:
    short_forms:
      one: "{{name}} has {{count}} cat."
      other: "{{name}} has {{count}} cats."
    long_forms:
      one: "{{name}} has one cat."
      other: "{{name}} has {{count}} cats."
    plural_param: count
`)
	if err := os.WriteFile(filepath.Join(dir, "en.yaml"), en, 0o600); err != nil {
		panic(err)
	}

	catalog, err := msgcat.NewMessageCatalog(msgcat.Config{ResourcePath: dir})
	if err != nil {
		panic(err)
	}

	ctx := context.WithValue(context.Background(), "language", "en")

	for _, count := range []int{0, 1, 2, 5} {
		msg := catalog.GetMessageWithCtx(ctx, "person.cats", msgcat.Params{"name": "Nick", "count": count})
		fmt.Printf("count=%d: %s\n", count, msg.ShortText)
	}
}
