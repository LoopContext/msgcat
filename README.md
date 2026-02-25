# msgcat

Message catalog to handle i18n of your messages and errors. msg-meow!

## Description

msgcat allows to setup a yaml formatted, context driven message catalog, 
that is, is possible to setup different language files for the implementing project.

Since it uses context key "language" (configurable as needed) to get the current 
language, it's possible to get the language of a given request from start to finish, 
this is specially good for i18n on APIs :)

## First thigs first

### go get msgcat onto your project

```bash
go get github.com/loopcontext/msgcat
```

### Create the resource files
The project layout should be like this by default (you can override the configuration of the catalog)

```ascii-art
/ (project root)
|
 -resources
     |
      - messages
         |
          - en.yaml
         |
          - es.yaml
```
---
`en.yaml`

```yaml
default:
  short: Unexpected error
  long: Unexpected message code [{{0}}] was received and was not found in this catalog
set:
  1:
    short: This is a message in english
  2:
    short: This is an error message
    long: Any variable can be injected here {{0}}
```
---
`es.yaml`

```yaml
default:
  short: Error inesperado
  long: Error inesperado con el con el código [{{0}}] ha sido solicitado y no encontrado en este catálogo.
set:
  1:
    short: Este mensaje es en español
```

Once have the files have been created, it's time to code!

```go
package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/loopcontext/msgcat"
)

func main() {
	ctx := context.WithValue(context.Background(), "language", "en")
	catalog, err := msgcat.NewMessageCatalog(msgcat.Config{
		// CtxLanguageKey: "lang",
		// ResourcePath: "path/to/resource",
	})
	if err != nil {
		fmt.Printf("%q",err)
	}
	printMessage(ctx, catalog.GetMessageWithCtx(ctx, 1))
	printMessage(ctx, catalog.GetMessageWithCtx(ctx, 2, 2*2))
	// The message can be get as a error
	printError(ctx, catalog.GetErrorWithCtx(ctx, 1))
	// The message can wrap another error
	printError(ctx, catalog.WrapErrorWithCtx(ctx, errors.New("This error is wrapped"), 1))

	ctx = context.WithValue(context.Background(), "language", "es")
	printMessage(ctx, catalog.GetMessageWithCtx(ctx, 1))
	printMessage(ctx, catalog.GetMessageWithCtx(ctx, 2)) // this will fail
	printError(ctx, catalog.GetErrorWithCtx(ctx, 1))
	printError(ctx, catalog.WrapErrorWithCtx(ctx, errors.New("This error is wrapped"), 1))
}

func printMessage(ctx context.Context, msg *msgcat.Message) {
	fmt.Printf("\nLanguage: %s\nShort: %s\nLong: %s\n", ctx.Value("language"), msg.ShortText, msg.LongText)
}

func printError(ctx context.Context, err error) {
	wrapped := errors.Unwrap(err)
	if wrapped != nil {
		err = wrapped
	}
	fmt.Printf("\nLanguage: %s\nError: %q\n", ctx.Value("language"), err)
}
```

This is the output:

```
Language: en
Short: This is a message in english
Long: 

Language: en
Short: This is an error message
Long: Any variable can be injected here 4

Language: en
Error: "This is a message in english"

Language: en
Error: "This error is wrapped"

Language: es
Short: Este mensaje es en español
Long: 

Language: es
Short: Error inesperado
Long: Error inesperado con el con el código [2] ha sido solicitado y no encontrado en este catálogo.

Language: es
Error: "Este mensaje es en español"

Language: es
Error: "This error is wrapped"
```
