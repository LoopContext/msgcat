package main

import (
	"expvar"
	"log"
	"time"

	"github.com/loopcontext/msgcat"
)

type expvarObserver struct {
	fallbacks *expvar.Map
	missingL  *expvar.Map
	missingM  *expvar.Map
	tplIssues *expvar.Map
}

func newExpvarObserver() *expvarObserver {
	return &expvarObserver{
		fallbacks: expvar.NewMap("msgcat_fallbacks"),
		missingL:  expvar.NewMap("msgcat_missing_languages"),
		missingM:  expvar.NewMap("msgcat_missing_messages"),
		tplIssues: expvar.NewMap("msgcat_template_issues"),
	}
}

func (o *expvarObserver) OnLanguageFallback(requestedLang string, resolvedLang string) {
	o.fallbacks.Add(requestedLang+"->"+resolvedLang, 1)
}
func (o *expvarObserver) OnLanguageMissing(lang string) {
	o.missingL.Add(lang, 1)
}
func (o *expvarObserver) OnMessageMissing(lang string, msgCode int) {
	o.missingM.Add(lang, 1)
}
func (o *expvarObserver) OnTemplateIssue(lang string, msgCode int, issue string) {
	o.tplIssues.Add(issue, 1)
}

func main() {
	observer := newExpvarObserver()
	catalog, err := msgcat.NewMessageCatalog(msgcat.Config{
		ResourcePath:      "./resources/messages",
		CtxLanguageKey:    "language",
		DefaultLanguage:   "en",
		FallbackLanguages: []string{"es"},
		Observer:          observer,
		ObserverBuffer:    1024,
		StatsMaxKeys:      512,
		ReloadRetries:     2,
		ReloadRetryDelay:  50 * time.Millisecond,
	})
	if err != nil {
		log.Fatalf("catalog init failed: %v", err)
	}
	defer func() { _ = msgcat.Close(catalog) }()

	log.Printf("stats endpoint available on expvar: /debug/vars (when imported via net/http/pprof or expvar HTTP setup)")
	_ = catalog
}
