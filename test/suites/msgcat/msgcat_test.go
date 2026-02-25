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

func (o *mockObserver) OnMessageMissing(lang string, msgCode int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.missingCodes = append(o.missingCodes, fmt.Sprintf("%s:%d", lang, msgCode))
}

func (o *mockObserver) OnTemplateIssue(lang string, msgCode int, issue string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.templateIssues = append(o.templateIssues, fmt.Sprintf("%s:%d:%s", lang, msgCode, issue))
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
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, 1)
		Expect(message.Code).To(Equal(1))
	})

	It("should return short message", func() {
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, 1)
		Expect(message.ShortText).To(Equal("Hello short description"))
	})

	It("should return long message", func() {
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, 1, "1")
		Expect(message.LongText).To(Equal("Hello veeery long description. You can only see me in details page."))
	})

	It("should return message code (with template)", func() {
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, 2, 1, "CODE")
		Expect(message.Code).To(Equal(2))
	})

	It("should return short message (with template)", func() {
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, 2, 1, "CODE")
		Expect(message.ShortText).To(Equal("Hello template 1, this is nice CODE"))
	})

	It("should return long message (with template)", func() {
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, 2, 1, "CODE")
		Expect(message.LongText).To(Equal("Hello veeery long 1 description. You can only see me in details CODE page."))
	})

	It("should not panic if template is wrong", func() {
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, 3, 1, "CODE")
		Expect(message.ShortText).To(HavePrefix("Invalid entry .p0"))
	})

	It("should return message in correct language", func() {
		ctx.SetValue("language", "es")
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, 1)
		Expect(message.ShortText).To(Equal("Hola, breve descripción"))
	})

	It("should read language with typed context key", func() {
		ctx.SetValue(msgcat.ContextKey("language"), "es")
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, 1)
		Expect(message.ShortText).To(Equal("Hola, breve descripción"))
	})

	It("should fallback from regional language to base language", func() {
		ctx.SetValue("language", "es-AR")
		message := messageCatalog.GetMessageWithCtx(ctx.Ctx, 1)
		Expect(message.ShortText).To(Equal("Hola, breve descripción"))
	})

	It("should return error with correct message", func() {
		ctx.SetValue("language", "es")
		err := messageCatalog.GetErrorWithCtx(ctx.Ctx, 1)
		Expect(err.Error()).To(Equal("Hola, breve descripción"))
	})

	It("should return error with correct message components", func() {
		ctx.SetValue("language", "es")
		err := messageCatalog.GetErrorWithCtx(ctx.Ctx, 1)
		castedError := err.(msgcat.Error)
		Expect(castedError.GetShortMessage()).To(Equal("Hola, breve descripción"))
		Expect(castedError.GetLongMessage()).To(Equal("Hola, descripción muy larga. Solo puedes verme en la página de detalles."))
		Expect(castedError.ErrorCode()).To(Equal(1))
	})

	It("should be able to load messages from code", func() {
		err := messageCatalog.LoadMessages("en", []msgcat.RawMessage{{
			LongTpl:  "Some long system message",
			ShortTpl: "Some short system message",
			Code:     9001,
		}})
		Expect(err).NotTo(HaveOccurred())
		err = messageCatalog.GetErrorWithCtx(ctx.Ctx, 9001)
		Expect(err.Error()).To(Equal("Some short system message"))
	})

	It("should load code messages for a new language without panic", func() {
		err := messageCatalog.LoadMessages("pt", []msgcat.RawMessage{{
			LongTpl:  "Mensagem longa de sistema",
			ShortTpl: "Mensagem curta de sistema",
			Code:     9001,
		}})
		Expect(err).NotTo(HaveOccurred())

		ctx.SetValue("language", "pt")
		err = messageCatalog.GetErrorWithCtx(ctx.Ctx, 9001)
		Expect(err.Error()).To(Equal("Mensagem curta de sistema"))
	})

	It("should load code messages when YAML set is missing", func() {
		tmpDir, err := os.MkdirTemp("", "msgcat-missing-set-*")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tmpDir)

		content := []byte("group: 0\ndefault:\n  short: Unexpected error\n  long: Unexpected error from missing set file\n")
		err = os.WriteFile(filepath.Join(tmpDir, "en.yaml"), content, 0o600)
		Expect(err).NotTo(HaveOccurred())

		customCatalog, err := msgcat.NewMessageCatalog(msgcat.Config{
			ResourcePath: tmpDir,
		})
		Expect(err).NotTo(HaveOccurred())

		err = customCatalog.LoadMessages("en", []msgcat.RawMessage{{
			LongTpl:  "Loaded from code",
			ShortTpl: "Loaded from code short",
			Code:     9001,
		}})
		Expect(err).NotTo(HaveOccurred())
		Expect(customCatalog.GetMessageWithCtx(ctx.Ctx, 9001).ShortText).To(Equal("Loaded from code short"))
	})

	It("should allow to load system messages between 9000-9999", func() {
		err := messageCatalog.LoadMessages("en", []msgcat.RawMessage{{
			LongTpl:  "Some long system message",
			ShortTpl: "Some short system message",
			Code:     8999,
		}})
		Expect(err).To(HaveOccurred())
		err = messageCatalog.LoadMessages("en", []msgcat.RawMessage{{
			LongTpl:  "Some long system message",
			ShortTpl: "Some short system message",
			Code:     0,
		}})
		Expect(err).To(HaveOccurred())
	})

	It("should wrap error", func() {
		err := errors.New("original error")
		ctErr := messageCatalog.WrapErrorWithCtx(ctx.Ctx, err, 1)
		Expect(errors.Is(ctErr, err)).To(BeTrue())
		Expect(errors.Unwrap(ctErr)).To(Equal(err))
	})

	It("should render pluralization and localized number/date tokens", func() {
		date := time.Date(2026, time.January, 3, 10, 0, 0, 0, time.UTC)
		msgEN := messageCatalog.GetMessageWithCtx(ctx.Ctx, 4, 3, 12345.5, date)
		Expect(msgEN.ShortText).To(Equal("You have 3 items"))
		Expect(msgEN.LongText).To(Equal("Total: 12,345.5 generated at 01/03/2026"))

		ctx.SetValue("language", "es")
		msgES := messageCatalog.GetMessageWithCtx(ctx.Ctx, 4, 1, 12345.5, date)
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

		msg := strictCatalog.GetMessageWithCtx(ctx.Ctx, 2, 1)
		Expect(msg.ShortText).To(Equal("Hello template 1, this is nice <missing:1>"))

		stats, err := msgcat.SnapshotStats(strictCatalog)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(stats.TemplateIssues)).To(BeNumerically(">", 0))
		Expect(len(observer.templateIssues)).To(BeNumerically(">", 0))
	})

	It("should reload yaml changes and keep runtime loaded messages", func() {
		tmpDir, err := os.MkdirTemp("", "msgcat-reload-*")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tmpDir)

		initial := []byte("group: 0\ndefault:\n  short: Unexpected error\n  long: Unexpected error from reload file\nset:\n  1:\n    short: Hello before reload\n")
		err = os.WriteFile(filepath.Join(tmpDir, "en.yaml"), initial, 0o600)
		Expect(err).NotTo(HaveOccurred())

		customCatalog, err := msgcat.NewMessageCatalog(msgcat.Config{
			ResourcePath: tmpDir,
		})
		Expect(err).NotTo(HaveOccurred())

		err = customCatalog.LoadMessages("en", []msgcat.RawMessage{{
			LongTpl:  "Runtime long",
			ShortTpl: "Runtime short",
			Code:     9001,
		}})
		Expect(err).NotTo(HaveOccurred())

		updated := []byte("group: 0\ndefault:\n  short: Unexpected error\n  long: Unexpected error from reload file\nset:\n  1:\n    short: Hello after reload\n")
		err = os.WriteFile(filepath.Join(tmpDir, "en.yaml"), updated, 0o600)
		Expect(err).NotTo(HaveOccurred())

		err = msgcat.Reload(customCatalog)
		Expect(err).NotTo(HaveOccurred())

		Expect(customCatalog.GetMessageWithCtx(ctx.Ctx, 1).ShortText).To(Equal("Hello after reload"))
		Expect(customCatalog.GetMessageWithCtx(ctx.Ctx, 9001).ShortText).To(Equal("Runtime short"))
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
		Expect(observedCatalog.GetMessageWithCtx(ctx.Ctx, 1).ShortText).To(Equal("Hola, breve descripción"))

		ctx.SetValue("language", "pt-BR")
		Expect(observedCatalog.GetMessageWithCtx(ctx.Ctx, 404).Code).To(Equal(msgcat.CodeMissingMessage))

		stats, err := msgcat.SnapshotStats(observedCatalog)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(stats.LanguageFallbacks)).To(BeNumerically(">", 0))
		Expect(len(stats.MissingMessages)).To(BeNumerically(">", 0))
		Expect(strings.Join(observer.fallbacks, ",")).To(ContainSubstring("es-mx->es"))
		Expect(len(observer.missingCodes)).To(BeNumerically(">", 0))
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
					msg := messageCatalog.GetMessageWithCtx(ctx.Ctx, 1)
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
				code := 9000 + i
				err := messageCatalog.LoadMessages("en", []msgcat.RawMessage{{
					LongTpl:  fmt.Sprintf("Long %d", code),
					ShortTpl: fmt.Sprintf("Short %d", code),
					Code:     code,
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
