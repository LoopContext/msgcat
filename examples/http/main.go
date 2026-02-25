package main

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/loopcontext/msgcat"
)

func languageMiddleware(next http.Handler, key msgcat.ContextKey) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := parseAcceptLanguage(r.Header.Get("Accept-Language"))
		ctx := context.WithValue(r.Context(), key, lang)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func parseAcceptLanguage(header string) string {
	if header == "" {
		return "en"
	}
	part := strings.Split(header, ",")[0]
	part = strings.TrimSpace(strings.Split(part, ";")[0])
	if part == "" {
		return "en"
	}
	return part
}

func main() {
	catalog, err := msgcat.NewMessageCatalog(msgcat.Config{
		ResourcePath:      "./resources/messages",
		CtxLanguageKey:    "language",
		DefaultLanguage:   "en",
		FallbackLanguages: []string{"es"},
		StrictTemplates:   true,
	})
	if err != nil {
		log.Fatalf("catalog init failed: %v", err)
	}
	defer func() { _ = msgcat.Close(catalog) }()

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		msg := catalog.GetMessageWithCtx(r.Context(), 1, "user")
		_, _ = w.Write([]byte(msg.ShortText))
	})

	http.Handle("/", languageMiddleware(h, "language"))
	log.Println("listening on :8080")
	_ = http.ListenAndServe(":8080", nil)
}
