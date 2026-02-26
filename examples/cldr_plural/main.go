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
  person.dogs:
    short: "{{name}} has {{plural:count|zero:no dogs|one:one dog|other:{{count}} dogs}}"
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
		fmt.Printf("cats count=%d: %s\n", count, msg.ShortText)
		msgDog := catalog.GetMessageWithCtx(ctx, "person.dogs", msgcat.Params{"name": "Nick", "count": count})
		fmt.Printf("dogs count=%d: %s\n", count, msgDog.ShortText)
	}

	ar := []byte(`default:
  short: خطأ
  long: خطأ
set:
  person.dogs:
    short: "{{name}} لديه {{plural:count|zero:لا كلاب|one:كلب واحد|two:كلبان|few:{{count}} كلاب|many:{{count}} كلباً|other:{{count}} كلب}}"
`)
	if err := os.WriteFile(filepath.Join(dir, "ar.yaml"), ar, 0o600); err != nil {
		panic(err)
	}
	msgcat.Reload(catalog)

	ctxAR := context.WithValue(context.Background(), "language", "ar")
	for _, count := range []int{0, 1, 2, 5, 11, 100} {
		msgDog := catalog.GetMessageWithCtx(ctxAR, "person.dogs", msgcat.Params{"name": "Nick", "count": count})
		fmt.Printf("AR dogs count=%d: %s\n", count, msgDog.ShortText)
	}
}
