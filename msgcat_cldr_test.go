package msgcat

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGetMessageWithCtx_CLDRForms(t *testing.T) {
	dir := t.TempDir()
	en := []byte(`default:
  short: Unexpected error
  long: Unexpected message
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
	if err := os.WriteFile(filepath.Join(dir, "en.yaml"), en, 0644); err != nil {
		t.Fatal(err)
	}
	catalog, err := NewMessageCatalog(Config{ResourcePath: dir})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), "language", "en")

	// count=1 -> one
	msg1 := catalog.GetMessageWithCtx(ctx, "person.cats", Params{"name": "Nick", "count": 1})
	if msg1.ShortText != "Nick has 1 cat." {
		t.Errorf("count=1 short: got %q", msg1.ShortText)
	}
	if msg1.LongText != "Nick has one cat." {
		t.Errorf("count=1 long: got %q", msg1.LongText)
	}

	// count=2 -> other
	msg2 := catalog.GetMessageWithCtx(ctx, "person.cats", Params{"name": "Nick", "count": 2})
	if msg2.ShortText != "Nick has 2 cats." {
		t.Errorf("count=2 short: got %q", msg2.ShortText)
	}
	if msg2.LongText != "Nick has 2 cats." {
		t.Errorf("count=2 long: got %q", msg2.LongText)
	}
}

func TestGetMessageWithCtx_CLDRForms_fallbackToOther(t *testing.T) {
	dir := t.TempDir()
	en := []byte(`default:
  short: Err
  long: Err
set:
  items:
    short_forms:
      other: "{{count}} items"
    long_forms:
      other: "{{count}} items total"
`)
	if err := os.WriteFile(filepath.Join(dir, "en.yaml"), en, 0644); err != nil {
		t.Fatal(err)
	}
	catalog, err := NewMessageCatalog(Config{ResourcePath: dir})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), "language", "en")
	msg := catalog.GetMessageWithCtx(ctx, "items", Params{"count": 1})
	// only "other" form defined; Form(en,1)=one but we fall back to other
	if msg.ShortText != "1 items" {
		t.Errorf("short: got %q", msg.ShortText)
	}
}

func TestLoadMessages_preservesShortFormsLongFormsPluralParam(t *testing.T) {
	dir := t.TempDir()
	en := []byte(`default:
  short: Err
  long: Err
set: {}
`)
	if err := os.WriteFile(filepath.Join(dir, "en.yaml"), en, 0644); err != nil {
		t.Fatal(err)
	}
	catalog, err := NewMessageCatalog(Config{ResourcePath: dir})
	if err != nil {
		t.Fatal(err)
	}
	err = catalog.LoadMessages("en", []RawMessage{{
		Key:   "sys.cats",
		ShortForms: map[string]string{"one": "{{name}} has {{n}} cat.", "other": "{{name}} has {{n}} cats."},
		LongForms:  map[string]string{"one": "One cat.", "other": "{{n}} cats."},
		PluralParam: "n",
	}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), "language", "en")
	msg1 := catalog.GetMessageWithCtx(ctx, "sys.cats", Params{"name": "Alice", "n": 1})
	if msg1.ShortText != "Alice has 1 cat." {
		t.Errorf("count=1 short: got %q", msg1.ShortText)
	}
	if msg1.LongText != "One cat." {
		t.Errorf("count=1 long: got %q", msg1.LongText)
	}
	msg2 := catalog.GetMessageWithCtx(ctx, "sys.cats", Params{"name": "Alice", "n": 2})
	if msg2.ShortText != "Alice has 2 cats." {
		t.Errorf("count=2 short: got %q", msg2.ShortText)
	}
}
