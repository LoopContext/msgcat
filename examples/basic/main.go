// Basic demonstrates: NewMessageCatalog, GetMessageWithCtx (nil params and with Params),
// GetErrorWithCtx, WrapErrorWithCtx, and the msgcat.Error interface.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/loopcontext/msgcat"
)

func main() {
	dir, err := os.MkdirTemp("", "msgcat-basic-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	en := []byte(`default:
  short: Unexpected error
  long: Message not found in catalog
set:
  greeting.hello:
    code: 1
    short: Hello
    long: Hello, welcome.
  greeting.template:
    code: 2
    short: Hello {{name}}, role {{role}}
    long: Hello {{name}}, you are {{role}}.
  error.gone:
    code: 404
    short: Not found
    long: Resource not found.
`)
	if err := os.WriteFile(filepath.Join(dir, "en.yaml"), en, 0o600); err != nil {
		panic(err)
	}

	catalog, err := msgcat.NewMessageCatalog(msgcat.Config{ResourcePath: dir})
	if err != nil {
		panic(err)
	}

	ctx := context.WithValue(context.Background(), "language", "en")

	// GetMessageWithCtx with nil params (no placeholders)
	msg := catalog.GetMessageWithCtx(ctx, "greeting.hello", nil)
	fmt.Println("nil params:", msg.ShortText, "| code:", msg.Code)

	// GetMessageWithCtx with Params
	msg = catalog.GetMessageWithCtx(ctx, "greeting.template", msgcat.Params{"name": "juan", "role": "admin"})
	fmt.Println("with params:", msg.ShortText, "| code:", msg.Code)

	// GetErrorWithCtx (error without wrapping)
	err = catalog.GetErrorWithCtx(ctx, "error.gone", nil)
	fmt.Println("GetErrorWithCtx:", err.Error())

	// WrapErrorWithCtx and msgcat.Error interface
	inner := errors.New("db: connection refused")
	err = catalog.WrapErrorWithCtx(ctx, inner, "error.gone", nil)
	var catErr msgcat.Error
	if errors.As(err, &catErr) {
		fmt.Println("WrapError short:", catErr.GetShortMessage())
		fmt.Println("WrapError code:", catErr.ErrorCode())
		fmt.Println("WrapError key (use when code is empty):", catErr.ErrorKey())
		fmt.Println("Unwrap:", catErr.Unwrap() == inner)
	}
}
