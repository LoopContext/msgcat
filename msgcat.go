package msgcat

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

//go:generate mockgen -source=msgcat.go -package mock_msgcat -destination=test/mock/msgcat.go

const MessageCatalogNotFound = "Unexpected error in message catalog, language [%s] not found. %s"

type MessageCatalog interface {
	// Allows to load more messages (9000 - 9999 - reserved to system messages)
	LoadMessages(lang string, messages []RawMessage) error
	GetMessageWithCtx(ctx context.Context, msgCode int, msgParams ...interface{}) *Message
	WrapErrorWithCtx(ctx context.Context, err error, msgCode int, msgParams ...interface{}) error
	GetErrorWithCtx(ctx context.Context, msgCode int, msgParams ...interface{}) error
}

type DefaultMessageCatalog struct {
	messages map[string]Messages // language with messages indexed by id
	cfg      Config
}

type MessageParams struct {
	Params map[string]interface{}
}

func (dmc *DefaultMessageCatalog) loadFromYaml() error {
	if dmc.cfg.ResourcePath == "" {
		dmc.cfg.ResourcePath = "./resources/messages"
	}
	messageFiles, err := ioutil.ReadDir(dmc.cfg.ResourcePath)
	if err != nil {
		return fmt.Errorf("failed to find messages %v", err)
	}

	messageByLang := map[string]Messages{}

	for _, messageFile := range messageFiles {
		fileName := messageFile.Name()
		if strings.HasSuffix(fileName, ".yaml") {
			var messages Messages
			lang := strings.TrimSuffix(fileName, ".yaml")
			yamlFile, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", dmc.cfg.ResourcePath, fileName))
			if err != nil {
				return fmt.Errorf("failed to read message file: %v", err)
			}
			err = yaml.Unmarshal(yamlFile, &messages)
			if err != nil {
				return fmt.Errorf("failed to unmarshal messages: %v", err)
			}
			messageByLang[lang] = messages
		}
	}

	dmc.messages = messageByLang

	return nil
}

func (dmc *DefaultMessageCatalog) LoadMessages(lang string, messages []RawMessage) error {
	if _, foundLangMsg := dmc.messages[lang]; !foundLangMsg {
		dmc.messages[lang] = Messages{}
	}
	langMsgSet := dmc.messages[lang]
	for _, message := range messages {
		if message.Code < 9000 || message.Code > 9999 {
			return fmt.Errorf("application messages should be loaded using YAML file, allowed range only between 9000 and 9999")
		} else if _, foundMsg := langMsgSet.Set[message.Code]; foundMsg {
			return fmt.Errorf("message with %d already exists in message set for language %s", message.Code, lang)
		}
		langMsgSet.Set[message.Code] = RawMessage{
			LongTpl:  message.LongTpl,
			ShortTpl: message.ShortTpl,
			Code:     message.Code,
		}
	}

	return nil
}

func (dmc *DefaultMessageCatalog) GetMessageWithCtx(ctx context.Context, msgCode int, msgParams ...interface{}) *Message {
	lang := ctx.Value(dmc.cfg.CtxLanguageKey)
	if lang == nil {
		lang = "en"
	}
	if langMsgSet, foundLangMsg := dmc.messages[lang.(string)]; foundLangMsg {
		shortMessage := langMsgSet.Default.ShortTpl
		longMessage := langMsgSet.Default.LongTpl
		code := 999999998
		if msg, foundMsg := langMsgSet.Set[msgCode]; foundMsg {
			// prepare mapped params
			shortMessage = msg.ShortTpl
			longMessage = msg.LongTpl
			code = msgCode + langMsgSet.Group
		} else {
			msgParams = []interface{}{msgCode}
		}
		for paramIdx, paramVal := range msgParams {
			shortMessage = strings.ReplaceAll(shortMessage, fmt.Sprintf("{{%d}}", paramIdx), fmt.Sprintf("%v", paramVal))
			longMessage = strings.ReplaceAll(longMessage, fmt.Sprintf("{{%d}}", paramIdx), fmt.Sprintf("%v", paramVal))
		}
		return &Message{
			LongText:  longMessage,
			ShortText: shortMessage,
			Code:      code,
		}
	}

	return &Message{
		ShortText: fmt.Sprintf(MessageCatalogNotFound, lang, ""),
		LongText:  fmt.Sprintf(MessageCatalogNotFound, lang, "Please, contact support."),
		Code:      99999999,
	}
}

func (dmc *DefaultMessageCatalog) WrapErrorWithCtx(ctx context.Context, err error, msgCode int, msgParams ...interface{}) error {
	message := dmc.GetMessageWithCtx(ctx, msgCode, msgParams...)

	return newCatalogError(message.Code, message.ShortText, message.LongText, err)
}

func (dmc *DefaultMessageCatalog) GetErrorWithCtx(ctx context.Context, msgCode int, msgParams ...interface{}) error {
	return dmc.WrapErrorWithCtx(ctx, nil, msgCode, msgParams...)
}

func NewMessageCatalog(cfg Config) (MessageCatalog, error) {
	if cfg.CtxLanguageKey == "" {
		cfg.CtxLanguageKey = "language"
	}
	dmc := DefaultMessageCatalog{
		cfg: cfg,
	}
	err := dmc.loadFromYaml()

	return &dmc, err
}
