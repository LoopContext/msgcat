package test_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/loopcontext/msgcat"
	"github.com/loopcontext/msgcat/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type mockObserver struct {
	mu             sync.Mutex
	fallbacks      []string
	missingLangs   []string
	missingCodes   []string
	templateIssues []string
}

type panicObserver struct{}

func (panicObserver) OnLanguageFallback(requestedLang string, resolvedLang string) {
	panic("observer panic")
}
func (panicObserver) OnLanguageMissing(lang string)                          { panic("observer panic") }
func (panicObserver) OnMessageMissing(lang string, msgKey string)              { panic("observer panic") }
func (panicObserver) OnTemplateIssue(lang string, msgKey string, issue string) { panic("observer panic") }

type slowObserver struct {
	delay time.Duration
}

func (o slowObserver) OnLanguageFallback(requestedLang string, resolvedLang string) {}
func (o slowObserver) OnLanguageMissing(lang string)                                {}
func (o slowObserver) OnMessageMissing(lang string, msgKey string) {
	time.Sleep(o.delay)
}
func (o slowObserver) OnTemplateIssue(lang string, msgKey string, issue string) {}

func (o *mockObserver) OnLanguageFallback(requestedLang string, resolvedLang string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.fallbacks = append(o.fallbacks, requestedLang+"->"+resolvedLang)
}

func (o *mockObserver) OnLanguageMissing(lang string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.missingLangs = append(o.missingLangs, lang)
}

func (o *mockObserver) OnMessageMissing(lang string, msgKey string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.missingCodes = append(o.missingCodes, fmt.Sprintf("%s:%s", lang, msgKey))
}

func (o *mockObserver) OnTemplateIssue(lang string, msgKey string, issue string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.templateIssues = append(o.templateIssues, fmt.Sprintf("%s:%s:%s", lang, msgKey, issue))
}

var _ = Describe("Message Catalog", func() {
	var messageCatalog msgcat.MessageCatalog
	var ctx *test.MockContext

	BeforeEach(func() {
		var err error
		ctx = &test.MockContext{Ctx: context.Background()}
		messageCatalog, err = msgcat.NewMessageCatalog(msgcat.Config{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should return message code", func() {
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, "greeting.hello", nil)
		Expect(message.Code).To(Equal(1))
	})

	It("should return short message", func() {
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, "greeting.hello", nil)
		Expect(message.ShortText).To(Equal("Hello short description"))
	})

	It("should return long message", func() {
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, "greeting.hello", nil)
		Expect(message.LongText).To(Equal("Hello veeery long description. You can only see me in details page."))
	})

	It("should return message code (with template)", func() {
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, "greeting.template", msgcat.Params{"name": 1, "detail": "CODE"})
		Expect(message.Code).To(Equal(2))
	})

	It("should return short message (with template)", func() {
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, "greeting.template", msgcat.Params{"name": 1, "detail": "CODE"})
		Expect(message.ShortText).To(Equal("Hello template 1, this is nice CODE"))
	})

	It("should return long message (with template)", func() {
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, "greeting.template", msgcat.Params{"name": 1, "detail": "CODE"})
		Expect(message.LongText).To(Equal("Hello veeery long 1 description. You can only see me in details CODE page."))
	})

	It("should not panic if template is wrong", func() {
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, "error.invalid_entry", msgcat.Params{"name": "x", "detail": "y"})
		Expect(message.ShortText).To(HavePrefix("Invalid entry x"))
	})

	It("should return message in correct language", func() {
		ctx.SetValue("language", "es")
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, "greeting.hello", nil)
		Expect(message.ShortText).To(Equal("Hola, breve descripción"))
	})

	It("should read language with typed context key", func() {
		ctx.SetValue(msgcat.ContextKey("language"), "es")
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, "greeting.hello", nil)
		Expect(message.ShortText).To(Equal("Hola, breve descripción"))
	})

	It("should fallback from regional language to base language", func() {
		ctx.SetValue("language", "es-AR")
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, "greeting.hello", nil)
		Expect(message.ShortText).To(Equal("Hola, breve descripción"))
	})

	It("should return error with correct message", func() {
		ctx.SetValue("language", "es")
		err := messageCatalog.GetErrorWithCtx(ctx.Ctx, "greeting.hello", nil)
		Expect(err.Error()).To(Equal("Hola, breve descripción"))
	})

	It("should return error with correct message components", func() {
		ctx.SetValue("language", "es")
		err := messageCatalog.GetErrorWithCtx(ctx.Ctx, "greeting.hello", nil)
		castedError := err.(msgcat.Error)
		Expect(castedError.GetShortMessage()).To(Equal("Hola, breve descripción"))
		Expect(castedError.GetLongMessage()).To(Equal("Hola, descripción muy larga. Solo puedes verme en la página de detalles."))
		Expect(castedError.ErrorCode()).To(Equal(1))
	})

	It("should be able to load messages from code", func() {
		err := messageCatalog.LoadMessages("en", []msgcat.RawMessage{{
			Key:      "sys.9001",
			LongTpl:  "Some long system message",
			ShortTpl: "Some short system message",
			Code:     9001,
		}})
		Expect(err).NotTo(HaveOccurred())
		err = messageCatalog.GetErrorWithCtx(ctx.Ctx, "sys.9001", nil)
		Expect(err.Error()).To(Equal("Some short system message"))
	})

	It("should load code messages for a new language without panic", func() {
		err := messageCatalog.LoadMessages("pt", []msgcat.RawMessage{{
			Key:      "sys.9001",
			LongTpl:  "Mensagem longa de sistema",
			ShortTpl: "Mensagem curta de sistema",
			Code:     9001,
		}})
		Expect(err).NotTo(HaveOccurred())

		ctx.SetValue("language", "pt")
		err = messageCatalog.GetErrorWithCtx(ctx.Ctx, "sys.9001", nil)
		Expect(err.Error()).To(Equal("Mensagem curta de sistema"))
	})

	It("should load code messages when YAML set is missing", func() {
		tmpDir, err := os.MkdirTemp("", "msgcat-missing-set-*")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tmpDir)

		content := []byte("default:\n  short: Unexpected error\n  long: Unexpected error from missing set file\nset: {}\n")
		err = os.WriteFile(filepath.Join(tmpDir, "en.yaml"), content, 0o600)
		Expect(err).NotTo(HaveOccurred())

		customCatalog, err := msgcat.NewMessageCatalog(msgcat.Config{
			ResourcePath: tmpDir,
		})
		Expect(err).NotTo(HaveOccurred())

		err = customCatalog.LoadMessages("en", []msgcat.RawMessage{{
			Key:      "sys.loaded",
			LongTpl:  "Loaded from code",
			ShortTpl: "Loaded from code short",
			Code:     9001,
		}})
		Expect(err).NotTo(HaveOccurred())
		Expect(customCatalog.GetMessageWithCtx(ctx.Ctx, "sys.loaded", nil).ShortText).To(Equal("Loaded from code short"))
	})

	It("should require sys. prefix for LoadMessages", func() {
		err := messageCatalog.LoadMessages("en", []msgcat.RawMessage{{
			Key:      "app.custom",
			LongTpl:  "Some long system message",
			ShortTpl: "Some short system message",
		}})
		Expect(err).To(HaveOccurred())
		err = messageCatalog.LoadMessages("en", []msgcat.RawMessage{{
			Key:      "",
			LongTpl:  "Some long system message",
			ShortTpl: "Some short system message",
		}})
		Expect(err).To(HaveOccurred())
	})

	It("should wrap error", func() {
		err := errors.New("original error")
		ctErr := messageCatalog.WrapErrorWithCtx(ctx.Ctx, err, "greeting.hello", nil)
		Expect(errors.Is(ctErr, err)).To(BeTrue())
		Expect(errors.Unwrap(ctErr)).To(Equal(err))
	})

	It("should render pluralization and localized number/date tokens", func() {
		date := time.Date(2026, time.January, 3, 10, 0, 0, 0, time.UTC)
		params := msgcat.Params{"count": 3, "amount": 12345.5, "generatedAt": date}
		msgEN := messageCatalog.GetMessageWithCtx(ctx.Ctx, "items.count", params)
		Expect(msgEN.ShortText).To(Equal("You have 3 items"))
		Expect(msgEN.LongText).To(Equal("Total: 12,345.5 generated at 01/03/2026"))

		ctx.SetValue("language", "es")
		paramsES := msgcat.Params{"count": 1, "amount": 12345.5, "generatedAt": date}
		msgES := messageCatalog.GetMessageWithCtx(ctx.Ctx, "items.count", paramsES)
		Expect(msgES.ShortText).To(Equal("Tienes 1 elemento"))
		Expect(msgES.LongText).To(Equal("Total: 12.345,5 generado el 03/01/2026"))
	})

	It("should support strict template checks and report template issues", func() {
		observer := &mockObserver{}
		strictCatalog, err := msgcat.NewMessageCatalog(msgcat.Config{
			ResourcePath:      "./resources/messages",
			StrictTemplates:   true,
			DefaultLanguage:   "en",
			CtxLanguageKey:    "language",
			FallbackLanguages: []string{"es"},
			Observer:          observer,
		})
		Expect(err).NotTo(HaveOccurred())

		msg := strictCatalog.GetMessageWithCtx(ctx.Ctx, "greeting.template", msgcat.Params{"name": 1})
		Expect(msg.ShortText).To(Equal("Hello template 1, this is nice <missing:detail>"))

		stats, err := msgcat.SnapshotStats(strictCatalog)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(stats.TemplateIssues)).To(BeNumerically(">", 0))
		Eventually(func() int {
			observer.mu.Lock()
			defer observer.mu.Unlock()
			return len(observer.templateIssues)
		}, 500*time.Millisecond, 10*time.Millisecond).Should(BeNumerically(">", 0))
	})

	It("should reload yaml changes and keep runtime loaded messages", func() {
		tmpDir, err := os.MkdirTemp("", "msgcat-reload-*")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tmpDir)

		initial := []byte("default:\n  short: Unexpected error\n  long: Unexpected error from reload file\nset:\n  greeting.hello:\n    short: Hello before reload\n")
		err = os.WriteFile(filepath.Join(tmpDir, "en.yaml"), initial, 0o600)
		Expect(err).NotTo(HaveOccurred())

		customCatalog, err := msgcat.NewMessageCatalog(msgcat.Config{
			ResourcePath: tmpDir,
		})
		Expect(err).NotTo(HaveOccurred())

		err = customCatalog.LoadMessages("en", []msgcat.RawMessage{{
			Key:      "sys.runtime",
			LongTpl:  "Runtime long",
			ShortTpl: "Runtime short",
			Code:     9001,
		}})
		Expect(err).NotTo(HaveOccurred())

		updated := []byte("default:\n  short: Unexpected error\n  long: Unexpected error from reload file\nset:\n  greeting.hello:\n    short: Hello after reload\n")
		err = os.WriteFile(filepath.Join(tmpDir, "en.yaml"), updated, 0o600)
		Expect(err).NotTo(HaveOccurred())

		err = msgcat.Reload(customCatalog)
		Expect(err).NotTo(HaveOccurred())

		Expect(customCatalog.GetMessageWithCtx(ctx.Ctx, "greeting.hello", nil).ShortText).To(Equal("Hello after reload"))
		Expect(customCatalog.GetMessageWithCtx(ctx.Ctx, "sys.runtime", nil).ShortText).To(Equal("Runtime short"))
	})

	It("should expose observability counters for fallback and misses", func() {
		observer := &mockObserver{}
		observedCatalog, err := msgcat.NewMessageCatalog(msgcat.Config{
			ResourcePath:      "./resources/messages",
			DefaultLanguage:   "en",
			FallbackLanguages: []string{"es"},
			Observer:          observer,
		})
		Expect(err).NotTo(HaveOccurred())

		ctx.SetValue("language", "es-MX")
		Expect(observedCatalog.GetMessageWithCtx(ctx.Ctx, "greeting.hello", nil).ShortText).To(Equal("Hola, breve descripción"))

		ctx.SetValue("language", "pt-BR")
		Expect(observedCatalog.GetMessageWithCtx(ctx.Ctx, "missing.key", nil).Code).To(Equal(msgcat.CodeMissingMessage))

		stats, err := msgcat.SnapshotStats(observedCatalog)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(stats.LanguageFallbacks)).To(BeNumerically(">", 0))
		Expect(len(stats.MissingMessages)).To(BeNumerically(">", 0))
		Eventually(func() string {
			observer.mu.Lock()
			defer observer.mu.Unlock()
			return strings.Join(observer.fallbacks, ",")
		}, 500*time.Millisecond, 10*time.Millisecond).Should(ContainSubstring("es-mx->es"))
		Eventually(func() int {
			observer.mu.Lock()
			defer observer.mu.Unlock()
			return len(observer.missingCodes)
		}, 500*time.Millisecond, 10*time.Millisecond).Should(BeNumerically(">", 0))
	})

	It("should keep observer failures from crashing request path", func() {
		catalogWithPanickingObserver, err := msgcat.NewMessageCatalog(msgcat.Config{
			ResourcePath: "./resources/messages",
			Observer:     panicObserver{},
		})
		Expect(err).NotTo(HaveOccurred())
		defer msgcat.Close(catalogWithPanickingObserver)

		ctx.SetValue("language", "es-MX")
		msg := catalogWithPanickingObserver.GetMessageWithCtx(ctx.Ctx, "missing.key", nil)
		Expect(msg).NotTo(BeNil())
	})

	It("should not block request path on slow observer", func() {
		catalogWithSlowObserver, err := msgcat.NewMessageCatalog(msgcat.Config{
			ResourcePath:   "./resources/messages",
			Observer:       slowObserver{delay: 150 * time.Millisecond},
			ObserverBuffer: 1,
		})
		Expect(err).NotTo(HaveOccurred())
		defer msgcat.Close(catalogWithSlowObserver)

		start := time.Now()
		msg := catalogWithSlowObserver.GetMessageWithCtx(ctx.Ctx, "missing.key", nil)
		elapsed := time.Since(start)
		Expect(msg).NotTo(BeNil())
		Expect(elapsed).To(BeNumerically("<", 80*time.Millisecond))
	})

	It("should cap stats cardinality and reset counters", func() {
		tmpDir, err := os.MkdirTemp("", "msgcat-empty-*")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tmpDir)

		emptyCatalog, err := msgcat.NewMessageCatalog(msgcat.Config{
			ResourcePath: tmpDir,
			StatsMaxKeys: 2,
		})
		Expect(err).NotTo(HaveOccurred())
		defer msgcat.Close(emptyCatalog)

		langs := []string{"aa", "bb", "cc", "dd"}
		for _, lang := range langs {
			ctx.SetValue("language", lang)
			_ = emptyCatalog.GetMessageWithCtx(ctx.Ctx, "any.key", nil)
		}

		stats, err := msgcat.SnapshotStats(emptyCatalog)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(stats.MissingLanguages)).To(BeNumerically("<=", 2))
		Expect(stats.MissingLanguages).To(HaveKey("__overflow__"))

		err = msgcat.ResetStats(emptyCatalog)
		Expect(err).NotTo(HaveOccurred())
		stats, err = msgcat.SnapshotStats(emptyCatalog)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(stats.MissingLanguages)).To(Equal(0))
		Expect(len(stats.MissingMessages)).To(Equal(0))
	})

	It("should apply now function timestamp and reload retries", func() {
		tmpDir, err := os.MkdirTemp("", "msgcat-retry-*")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tmpDir)

		initial := []byte("default:\n  short: Init\n  long: Init\nset:\n  greeting.hello:\n    short: before\n")
		err = os.WriteFile(filepath.Join(tmpDir, "en.yaml"), initial, 0o600)
		Expect(err).NotTo(HaveOccurred())

		fixedTime := time.Date(2026, time.February, 25, 12, 30, 0, 0, time.UTC)
		catalogWithRetry, err := msgcat.NewMessageCatalog(msgcat.Config{
			ResourcePath:     tmpDir,
			NowFn:            func() time.Time { return fixedTime },
			ReloadRetries:    2,
			ReloadRetryDelay: 20 * time.Millisecond,
		})
		Expect(err).NotTo(HaveOccurred())
		defer msgcat.Close(catalogWithRetry)

		err = os.WriteFile(filepath.Join(tmpDir, "en.yaml"), []byte("invalid: ["), 0o600)
		Expect(err).NotTo(HaveOccurred())
		go func() {
			time.Sleep(10 * time.Millisecond)
			_ = os.WriteFile(filepath.Join(tmpDir, "en.yaml"), []byte("default:\n  short: Init\n  long: Init\nset:\n  greeting.hello:\n    short: after\n"), 0o600)
		}()

		err = msgcat.Reload(catalogWithRetry)
		Expect(err).NotTo(HaveOccurred())
		msg := catalogWithRetry.GetMessageWithCtx(ctx.Ctx, "greeting.hello", nil)
		Expect(msg.ShortText).To(Equal("after"))

		stats, err := msgcat.SnapshotStats(catalogWithRetry)
		Expect(err).NotTo(HaveOccurred())
		Expect(stats.LastReloadAt).To(Equal(fixedTime))
	})

	It("should be safe under concurrent reads and writes", func() {
		ctx.SetValue("language", "en")

		const (
			readers       = 12
			readerIters   = 200
			writerEntries = 20
		)

		errCh := make(chan error, readers+writerEntries)
		var wg sync.WaitGroup

		for i := 0; i < readers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < readerIters; j++ {
					msg := messageCatalog.GetMessageWithCtx(ctx.Ctx, "greeting.hello", nil)
					if msg.ShortText == "" {
						errCh <- fmt.Errorf("received empty message")
						return
					}
				}
			}()
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < writerEntries; i++ {
				key := fmt.Sprintf("sys.concurrent_%d", i)
				err := messageCatalog.LoadMessages("en", []msgcat.RawMessage{{
					Key:      key,
					LongTpl:  fmt.Sprintf("Long %s", key),
					ShortTpl: fmt.Sprintf("Short %s", key),
					Code:     9000 + i,
				}})
				if err != nil {
					errCh <- err
					return
				}
			}
		}()

		wg.Wait()
		close(errCh)

		for err := range errCh {
			Expect(err).NotTo(HaveOccurred())
		}
	})
})
