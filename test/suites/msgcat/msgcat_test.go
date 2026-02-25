package test_test

import (
	"context"
	"errors"

	"github.com/loopcontext/msgcat"
	"github.com/loopcontext/msgcat/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

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
})
