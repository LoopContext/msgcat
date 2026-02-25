package msgcat

import (
	"context"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

//go:generate mockgen -source=$GOFILE -package mock_msgcat -destination=test/mock/$GOFILE

const MessageCatalogNotFound = "Unexpected error in message catalog, language [%s] not found. %s"

const (
	SystemMessageMinCode = 9000
	SystemMessageMaxCode = 9999
	CodeMissingMessage   = 999999998
	CodeMissingLanguage  = 99999999
	overflowStatKey      = "__overflow__"
)

var (
	simplePlaceholderRegex = regexp.MustCompile(`\{\{(\d+)\}\}`)
	pluralPlaceholderRegex = regexp.MustCompile(`\{\{plural:(\d+)\|([^|}]*)\|([^}]*)\}\}`)
	numberPlaceholderRegex = regexp.MustCompile(`\{\{num:(\d+)\}\}`)
	datePlaceholderRegex   = regexp.MustCompile(`\{\{date:(\d+)\}\}`)
)

type MessageCatalog interface {
	// Allows to load more messages (9000 - 9999 - reserved to system messages)
	LoadMessages(lang string, messages []RawMessage) error
	GetMessageWithCtx(ctx context.Context, msgCode int, msgParams ...interface{}) *Message
	WrapErrorWithCtx(ctx context.Context, err error, msgCode int, msgParams ...interface{}) error
	GetErrorWithCtx(ctx context.Context, msgCode int, msgParams ...interface{}) error
}

type observerEventType int

const (
	observerEventLanguageFallback observerEventType = iota
	observerEventLanguageMissing
	observerEventMessageMissing
	observerEventTemplateIssue
)

type observerEvent struct {
	kind          observerEventType
	requested     string
	resolved      string
	lang          string
	msgCode       int
	templateIssue string
}

type catalogStats struct {
	mu                sync.Mutex
	languageFallbacks map[string]int
	missingLanguages  map[string]int
	missingMessages   map[string]int
	templateIssues    map[string]int
	droppedEvents     map[string]int
	maxKeys           int
	lastReloadAt      time.Time
}

func sanitizeStatKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return "unknown"
	}
	if len(key) > 120 {
		return key[:120]
	}
	return key
}

func (s *catalogStats) increment(target map[string]int, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if target == nil {
		return
	}
	key = sanitizeStatKey(key)
	if s.maxKeys > 0 {
		if _, exists := target[key]; !exists {
			if _, hasOverflow := target[overflowStatKey]; hasOverflow {
				if len(target) >= s.maxKeys {
					key = overflowStatKey
				}
			} else if len(target) >= s.maxKeys-1 {
				key = overflowStatKey
			}
		}
	}
	target[key]++
}

func (s *catalogStats) incrementLanguageFallback(requestedLang string, resolvedLang string) {
	s.increment(s.languageFallbacks, fmt.Sprintf("%s->%s", requestedLang, resolvedLang))
}

func (s *catalogStats) incrementMissingLanguage(lang string) {
	s.increment(s.missingLanguages, normalizeLangTag(lang))
}

func (s *catalogStats) incrementMissingMessage(lang string, msgCode int) {
	s.increment(s.missingMessages, fmt.Sprintf("%s:%d", lang, msgCode))
}

func (s *catalogStats) incrementTemplateIssue(lang string, msgCode int, issue string) {
	s.increment(s.templateIssues, fmt.Sprintf("%s:%d:%s", lang, msgCode, issue))
}

func (s *catalogStats) incrementDroppedEvent(reason string) {
	s.increment(s.droppedEvents, reason)
}

func (s *catalogStats) setLastReloadAt(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastReloadAt = t
}

func (s *catalogStats) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.languageFallbacks = map[string]int{}
	s.missingLanguages = map[string]int{}
	s.missingMessages = map[string]int{}
	s.templateIssues = map[string]int{}
	s.droppedEvents = map[string]int{}
	s.lastReloadAt = time.Time{}
}

func (s *catalogStats) snapshot() MessageCatalogStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	copyMap := func(input map[string]int) map[string]int {
		output := make(map[string]int, len(input))
		for k, v := range input {
			output[k] = v
		}
		return output
	}

	return MessageCatalogStats{
		LanguageFallbacks: copyMap(s.languageFallbacks),
		MissingLanguages:  copyMap(s.missingLanguages),
		MissingMessages:   copyMap(s.missingMessages),
		TemplateIssues:    copyMap(s.templateIssues),
		DroppedEvents:     copyMap(s.droppedEvents),
		LastReloadAt:      s.lastReloadAt,
	}
}

type DefaultMessageCatalog struct {
	mu              sync.RWMutex
	messages        map[string]Messages // language with messages indexed by id
	runtimeMessages map[string]map[int]RawMessage
	cfg             Config
	stats           catalogStats
	observerCh      chan observerEvent
	observerDone    chan struct{}
}

type MessageParams struct {
	Params map[string]interface{}
}

func (dmc *DefaultMessageCatalog) readMessagesFromYaml() (map[string]Messages, error) {
	resourcePath := dmc.cfg.ResourcePath
	if resourcePath == "" {
		resourcePath = "./resources/messages"
	}

	messageFiles, err := ioutil.ReadDir(resourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to find messages %v", err)
	}

	messageByLang := map[string]Messages{}

	for _, messageFile := range messageFiles {
		fileName := messageFile.Name()
		if !strings.HasSuffix(fileName, ".yaml") {
			continue
		}
		var messages Messages
		lang := normalizeLangTag(strings.TrimSuffix(fileName, ".yaml"))
		yamlFile, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", resourcePath, fileName))
		if err != nil {
			return nil, fmt.Errorf("failed to read message file: %v", err)
		}
		err = yaml.Unmarshal(yamlFile, &messages)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal messages: %v", err)
		}
		if err := normalizeAndValidateMessages(lang, &messages); err != nil {
			return nil, err
		}
		messageByLang[lang] = messages
	}

	return messageByLang, nil
}

func (dmc *DefaultMessageCatalog) readMessagesFromYamlWithRetry() (map[string]Messages, error) {
	retries := dmc.cfg.ReloadRetries
	if retries < 0 {
		retries = 0
	}
	delay := dmc.cfg.ReloadRetryDelay
	if delay <= 0 {
		delay = 50 * time.Millisecond
	}

	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		messageByLang, err := dmc.readMessagesFromYaml()
		if err == nil {
			return messageByLang, nil
		}
		lastErr = err
		if attempt < retries {
			time.Sleep(delay)
		}
	}

	return nil, lastErr
}

func (dmc *DefaultMessageCatalog) loadFromYaml() error {
	messageByLang, err := dmc.readMessagesFromYamlWithRetry()
	if err != nil {
		return err
	}

	dmc.mu.Lock()
	defer dmc.mu.Unlock()
	if dmc.runtimeMessages != nil {
		for lang, runtimeSet := range dmc.runtimeMessages {
			msgSet, found := messageByLang[lang]
			if !found {
				msgSet = Messages{Set: map[int]RawMessage{}}
			}
			if msgSet.Set == nil {
				msgSet.Set = map[int]RawMessage{}
			}
			for code, msg := range runtimeSet {
				msgSet.Set[code] = msg
			}
			messageByLang[lang] = msgSet
		}
	}
	dmc.messages = messageByLang
	dmc.stats.setLastReloadAt(dmc.cfg.NowFn())

	return nil
}

func normalizeAndValidateMessages(lang string, messages *Messages) error {
	if messages.Group < 0 {
		return fmt.Errorf("invalid message group for language %s: must be >= 0", lang)
	}
	if messages.Default.ShortTpl == "" && messages.Default.LongTpl == "" {
		return fmt.Errorf("invalid default message for language %s: at least short or long text is required", lang)
	}
	if messages.Set == nil {
		messages.Set = map[int]RawMessage{}
	}
	for code, raw := range messages.Set {
		if code <= 0 {
			return fmt.Errorf("invalid message code %d for language %s: must be > 0", code, lang)
		}
		raw.Code = code
		messages.Set[code] = raw
	}

	return nil
}

func normalizeLangTag(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	lang = strings.ReplaceAll(lang, "_", "-")
	return lang
}

func baseLangTag(lang string) string {
	if idx := strings.Index(lang, "-"); idx > 0 {
		return lang[:idx]
	}
	return lang
}

func appendLangIfMissing(target *[]string, seen map[string]struct{}, lang string) {
	if lang == "" {
		return
	}
	if _, exists := seen[lang]; exists {
		return
	}
	seen[lang] = struct{}{}
	*target = append(*target, lang)
}

func isPluralOne(value interface{}) (bool, bool) {
	switch typed := value.(type) {
	case int:
		return typed == 1, true
	case int8:
		return typed == 1, true
	case int16:
		return typed == 1, true
	case int32:
		return typed == 1, true
	case int64:
		return typed == 1, true
	case uint:
		return typed == 1, true
	case uint8:
		return typed == 1, true
	case uint16:
		return typed == 1, true
	case uint32:
		return typed == 1, true
	case uint64:
		return typed == 1, true
	case float32:
		return typed == 1, true
	case float64:
		return typed == 1, true
	default:
		return false, false
	}
}

func groupDigits(input string, separator string) string {
	if len(input) <= 3 {
		return input
	}
	start := len(input) % 3
	if start == 0 {
		start = 3
	}
	var b strings.Builder
	b.WriteString(input[:start])
	for i := start; i < len(input); i += 3 {
		b.WriteString(separator)
		b.WriteString(input[i : i+3])
	}
	return b.String()
}

func formatNumberByLang(lang string, value interface{}) (string, bool) {
	decimalSeparator := "."
	groupSeparator := ","
	switch baseLangTag(lang) {
	case "es", "pt", "fr", "de", "it":
		decimalSeparator = ","
		groupSeparator = "."
	}

	formatFloat := func(number float64) string {
		s := strconv.FormatFloat(number, 'f', -1, 64)
		parts := strings.SplitN(s, ".", 2)
		intPart := parts[0]
		sign := ""
		if strings.HasPrefix(intPart, "-") {
			sign = "-"
			intPart = strings.TrimPrefix(intPart, "-")
		}
		intPart = sign + groupDigits(intPart, groupSeparator)
		if len(parts) == 1 {
			return intPart
		}
		return intPart + decimalSeparator + parts[1]
	}

	switch typed := value.(type) {
	case int:
		return groupDigits(strconv.Itoa(typed), groupSeparator), true
	case int8:
		return groupDigits(strconv.FormatInt(int64(typed), 10), groupSeparator), true
	case int16:
		return groupDigits(strconv.FormatInt(int64(typed), 10), groupSeparator), true
	case int32:
		return groupDigits(strconv.FormatInt(int64(typed), 10), groupSeparator), true
	case int64:
		return groupDigits(strconv.FormatInt(typed, 10), groupSeparator), true
	case uint:
		return groupDigits(strconv.FormatUint(uint64(typed), 10), groupSeparator), true
	case uint8:
		return groupDigits(strconv.FormatUint(uint64(typed), 10), groupSeparator), true
	case uint16:
		return groupDigits(strconv.FormatUint(uint64(typed), 10), groupSeparator), true
	case uint32:
		return groupDigits(strconv.FormatUint(uint64(typed), 10), groupSeparator), true
	case uint64:
		return groupDigits(strconv.FormatUint(typed, 10), groupSeparator), true
	case float32:
		return formatFloat(float64(typed)), true
	case float64:
		return formatFloat(typed), true
	default:
		return "", false
	}
}

func formatDateByLang(lang string, value interface{}) (string, bool) {
	var date time.Time
	switch typed := value.(type) {
	case time.Time:
		date = typed
	case *time.Time:
		if typed == nil {
			return "", false
		}
		date = *typed
	default:
		return "", false
	}

	layout := "01/02/2006"
	switch baseLangTag(lang) {
	case "es", "pt", "fr", "de", "it":
		layout = "02/01/2006"
	}

	return date.Format(layout), true
}

func safeObserverCall(fn func()) {
	defer func() {
		_ = recover()
	}()
	fn()
}

func (dmc *DefaultMessageCatalog) startObserverWorker() {
	if dmc.cfg.Observer == nil || dmc.observerCh != nil {
		return
	}
	dmc.observerCh = make(chan observerEvent, dmc.cfg.ObserverBuffer)
	dmc.observerDone = make(chan struct{})
	go func() {
		defer close(dmc.observerDone)
		for evt := range dmc.observerCh {
			switch evt.kind {
			case observerEventLanguageFallback:
				safeObserverCall(func() {
					dmc.cfg.Observer.OnLanguageFallback(evt.requested, evt.resolved)
				})
			case observerEventLanguageMissing:
				safeObserverCall(func() {
					dmc.cfg.Observer.OnLanguageMissing(evt.lang)
				})
			case observerEventMessageMissing:
				safeObserverCall(func() {
					dmc.cfg.Observer.OnMessageMissing(evt.lang, evt.msgCode)
				})
			case observerEventTemplateIssue:
				safeObserverCall(func() {
					dmc.cfg.Observer.OnTemplateIssue(evt.lang, evt.msgCode, evt.templateIssue)
				})
			}
		}
	}()
}

func (dmc *DefaultMessageCatalog) stopObserverWorker() {
	if dmc.observerCh == nil {
		return
	}
	close(dmc.observerCh)
	<-dmc.observerDone
	dmc.observerCh = nil
	dmc.observerDone = nil
}

func (dmc *DefaultMessageCatalog) publishObserverEvent(evt observerEvent) {
	if dmc.cfg.Observer == nil || dmc.observerCh == nil {
		return
	}
	defer func() {
		if recover() != nil {
			dmc.stats.incrementDroppedEvent("observer_closed")
		}
	}()
	select {
	case dmc.observerCh <- evt:
	default:
		dmc.stats.incrementDroppedEvent("observer_queue_full")
	}
}

func (dmc *DefaultMessageCatalog) onLanguageFallback(requestedLang string, resolvedLang string) {
	dmc.stats.incrementLanguageFallback(requestedLang, resolvedLang)
	dmc.publishObserverEvent(observerEvent{
		kind:      observerEventLanguageFallback,
		requested: requestedLang,
		resolved:  resolvedLang,
	})
}

func (dmc *DefaultMessageCatalog) onLanguageMissing(lang string) {
	dmc.stats.incrementMissingLanguage(lang)
	dmc.publishObserverEvent(observerEvent{
		kind: observerEventLanguageMissing,
		lang: lang,
	})
}

func (dmc *DefaultMessageCatalog) onMessageMissing(lang string, msgCode int) {
	dmc.stats.incrementMissingMessage(lang, msgCode)
	dmc.publishObserverEvent(observerEvent{
		kind:    observerEventMessageMissing,
		lang:    lang,
		msgCode: msgCode,
	})
}

func (dmc *DefaultMessageCatalog) onTemplateIssue(lang string, msgCode int, issue string) {
	dmc.stats.incrementTemplateIssue(lang, msgCode, issue)
	dmc.publishObserverEvent(observerEvent{
		kind:          observerEventTemplateIssue,
		lang:          lang,
		msgCode:       msgCode,
		templateIssue: issue,
	})
}

func (dmc *DefaultMessageCatalog) resolveRequestedLang(ctx context.Context) string {
	lang := normalizeLangTag(dmc.cfg.DefaultLanguage)
	if lang == "" {
		lang = "en"
	}
	if ctx == nil {
		return lang
	}

	// Keep backward compatibility with callers that used plain string keys.
	if langKeyVal := ctx.Value(dmc.cfg.CtxLanguageKey); langKeyVal != nil {
		return normalizeLangTag(fmt.Sprintf("%v", langKeyVal))
	}
	if langKeyVal := ctx.Value(string(dmc.cfg.CtxLanguageKey)); langKeyVal != nil {
		return normalizeLangTag(fmt.Sprintf("%v", langKeyVal))
	}

	return lang
}

func (dmc *DefaultMessageCatalog) resolveLanguage(requestedLang string) (string, bool, bool) {
	normalizedRequested := normalizeLangTag(requestedLang)
	if normalizedRequested == "" {
		normalizedRequested = "en"
	}

	candidates := make([]string, 0, 6)
	seen := map[string]struct{}{}
	appendLangIfMissing(&candidates, seen, normalizedRequested)
	appendLangIfMissing(&candidates, seen, baseLangTag(normalizedRequested))
	for _, lang := range dmc.cfg.FallbackLanguages {
		appendLangIfMissing(&candidates, seen, normalizeLangTag(lang))
	}
	appendLangIfMissing(&candidates, seen, normalizeLangTag(dmc.cfg.DefaultLanguage))
	appendLangIfMissing(&candidates, seen, "en")

	dmc.mu.RLock()
	defer dmc.mu.RUnlock()
	for _, candidate := range candidates {
		if _, found := dmc.messages[candidate]; found {
			return candidate, true, candidate != normalizedRequested
		}
	}

	return normalizedRequested, false, false
}

func (dmc *DefaultMessageCatalog) renderTemplate(lang string, msgCode int, template string, params []interface{}) string {
	rendered := template
	replaceMissing := func(issue string, originalToken string, idx int) string {
		dmc.onTemplateIssue(lang, msgCode, issue)
		if dmc.cfg.StrictTemplates {
			return fmt.Sprintf("<missing:%d>", idx)
		}
		return originalToken
	}

	rendered = pluralPlaceholderRegex.ReplaceAllStringFunc(rendered, func(token string) string {
		matches := pluralPlaceholderRegex.FindStringSubmatch(token)
		if len(matches) != 4 {
			return token
		}
		idx, err := strconv.Atoi(matches[1])
		if err != nil || idx < 0 {
			return token
		}
		if idx >= len(params) {
			return replaceMissing(fmt.Sprintf("plural_missing_param_%d", idx), token, idx)
		}
		isOne, ok := isPluralOne(params[idx])
		if !ok {
			dmc.onTemplateIssue(lang, msgCode, fmt.Sprintf("plural_invalid_param_%d", idx))
			return token
		}
		if isOne {
			return matches[2]
		}
		return matches[3]
	})

	rendered = numberPlaceholderRegex.ReplaceAllStringFunc(rendered, func(token string) string {
		matches := numberPlaceholderRegex.FindStringSubmatch(token)
		if len(matches) != 2 {
			return token
		}
		idx, err := strconv.Atoi(matches[1])
		if err != nil || idx < 0 {
			return token
		}
		if idx >= len(params) {
			return replaceMissing(fmt.Sprintf("number_missing_param_%d", idx), token, idx)
		}
		formatted, ok := formatNumberByLang(lang, params[idx])
		if !ok {
			dmc.onTemplateIssue(lang, msgCode, fmt.Sprintf("number_invalid_param_%d", idx))
			return token
		}
		return formatted
	})

	rendered = datePlaceholderRegex.ReplaceAllStringFunc(rendered, func(token string) string {
		matches := datePlaceholderRegex.FindStringSubmatch(token)
		if len(matches) != 2 {
			return token
		}
		idx, err := strconv.Atoi(matches[1])
		if err != nil || idx < 0 {
			return token
		}
		if idx >= len(params) {
			return replaceMissing(fmt.Sprintf("date_missing_param_%d", idx), token, idx)
		}
		formatted, ok := formatDateByLang(lang, params[idx])
		if !ok {
			dmc.onTemplateIssue(lang, msgCode, fmt.Sprintf("date_invalid_param_%d", idx))
			return token
		}
		return formatted
	})

	rendered = simplePlaceholderRegex.ReplaceAllStringFunc(rendered, func(token string) string {
		matches := simplePlaceholderRegex.FindStringSubmatch(token)
		if len(matches) != 2 {
			return token
		}
		idx, err := strconv.Atoi(matches[1])
		if err != nil || idx < 0 {
			return token
		}
		if idx >= len(params) {
			return replaceMissing(fmt.Sprintf("simple_missing_param_%d", idx), token, idx)
		}
		return fmt.Sprintf("%v", params[idx])
	})

	return rendered
}

func (dmc *DefaultMessageCatalog) LoadMessages(lang string, messages []RawMessage) error {
	dmc.mu.Lock()
	defer dmc.mu.Unlock()

	normalizedLang := normalizeLangTag(lang)
	if normalizedLang == "" {
		return fmt.Errorf("language is required")
	}

	if dmc.messages == nil {
		dmc.messages = map[string]Messages{}
	}
	if dmc.runtimeMessages == nil {
		dmc.runtimeMessages = map[string]map[int]RawMessage{}
	}
	if _, foundLangMsg := dmc.messages[normalizedLang]; !foundLangMsg {
		dmc.messages[normalizedLang] = Messages{
			Set: map[int]RawMessage{},
		}
	}
	if _, foundRuntimeLang := dmc.runtimeMessages[normalizedLang]; !foundRuntimeLang {
		dmc.runtimeMessages[normalizedLang] = map[int]RawMessage{}
	}

	langMsgSet := dmc.messages[normalizedLang]
	if langMsgSet.Set == nil {
		langMsgSet.Set = map[int]RawMessage{}
	}

	for _, message := range messages {
		if message.Code < SystemMessageMinCode || message.Code > SystemMessageMaxCode {
			return fmt.Errorf("application messages should be loaded using YAML file, allowed range only between %d and %d", SystemMessageMinCode, SystemMessageMaxCode)
		}
		if _, foundMsg := langMsgSet.Set[message.Code]; foundMsg {
			return fmt.Errorf("message with %d already exists in message set for language %s", message.Code, normalizedLang)
		}
		normalizedMessage := RawMessage{
			LongTpl:  message.LongTpl,
			ShortTpl: message.ShortTpl,
			Code:     message.Code,
		}
		langMsgSet.Set[message.Code] = normalizedMessage
		dmc.runtimeMessages[normalizedLang][message.Code] = normalizedMessage
	}
	dmc.messages[normalizedLang] = langMsgSet

	return nil
}

func (dmc *DefaultMessageCatalog) GetMessageWithCtx(ctx context.Context, msgCode int, msgParams ...interface{}) *Message {
	requestedLang := dmc.resolveRequestedLang(ctx)
	resolvedLang, foundLangMsg, usedFallback := dmc.resolveLanguage(requestedLang)
	if !foundLangMsg {
		dmc.onLanguageMissing(requestedLang)
		return &Message{
			ShortText: fmt.Sprintf(MessageCatalogNotFound, requestedLang, ""),
			LongText:  fmt.Sprintf(MessageCatalogNotFound, requestedLang, "Please, contact support."),
			Code:      CodeMissingLanguage,
		}
	}
	if usedFallback {
		dmc.onLanguageFallback(requestedLang, resolvedLang)
	}

	dmc.mu.RLock()
	langMsgSet, langExists := dmc.messages[resolvedLang]
	if !langExists {
		dmc.mu.RUnlock()
		dmc.onLanguageMissing(requestedLang)
		return &Message{
			ShortText: fmt.Sprintf(MessageCatalogNotFound, requestedLang, ""),
			LongText:  fmt.Sprintf(MessageCatalogNotFound, requestedLang, "Please, contact support."),
			Code:      CodeMissingLanguage,
		}
	}

	shortMessage := langMsgSet.Default.ShortTpl
	longMessage := langMsgSet.Default.LongTpl
	code := CodeMissingMessage
	missingMessage := false
	if msg, foundMsg := langMsgSet.Set[msgCode]; foundMsg {
		shortMessage = msg.ShortTpl
		longMessage = msg.LongTpl
		code = msgCode + langMsgSet.Group
	} else {
		missingMessage = true
		msgParams = []interface{}{msgCode}
	}
	dmc.mu.RUnlock()
	if missingMessage {
		dmc.onMessageMissing(resolvedLang, msgCode)
	}

	shortMessage = dmc.renderTemplate(resolvedLang, msgCode, shortMessage, msgParams)
	longMessage = dmc.renderTemplate(resolvedLang, msgCode, longMessage, msgParams)
	return &Message{
		LongText:  longMessage,
		ShortText: shortMessage,
		Code:      code,
	}
}

func (dmc *DefaultMessageCatalog) WrapErrorWithCtx(ctx context.Context, err error, msgCode int, msgParams ...interface{}) error {
	message := dmc.GetMessageWithCtx(ctx, msgCode, msgParams...)

	return newCatalogError(message.Code, message.ShortText, message.LongText, err)
}

func (dmc *DefaultMessageCatalog) GetErrorWithCtx(ctx context.Context, msgCode int, msgParams ...interface{}) error {
	return dmc.WrapErrorWithCtx(ctx, nil, msgCode, msgParams...)
}

func (dmc *DefaultMessageCatalog) Reload() error {
	return dmc.loadFromYaml()
}

func (dmc *DefaultMessageCatalog) SnapshotStats() MessageCatalogStats {
	return dmc.stats.snapshot()
}

func (dmc *DefaultMessageCatalog) ResetStats() {
	dmc.stats.reset()
}

func (dmc *DefaultMessageCatalog) Close() {
	dmc.mu.Lock()
	defer dmc.mu.Unlock()
	dmc.stopObserverWorker()
}

func Reload(catalog MessageCatalog) error {
	reloadable, ok := catalog.(interface{ Reload() error })
	if !ok {
		return fmt.Errorf("catalog does not support reload")
	}
	return reloadable.Reload()
}

func SnapshotStats(catalog MessageCatalog) (MessageCatalogStats, error) {
	statsProvider, ok := catalog.(interface{ SnapshotStats() MessageCatalogStats })
	if !ok {
		return MessageCatalogStats{}, fmt.Errorf("catalog does not support stats snapshots")
	}
	return statsProvider.SnapshotStats(), nil
}

func ResetStats(catalog MessageCatalog) error {
	statsProvider, ok := catalog.(interface{ ResetStats() })
	if !ok {
		return fmt.Errorf("catalog does not support stats reset")
	}
	statsProvider.ResetStats()
	return nil
}

func Close(catalog MessageCatalog) error {
	closer, ok := catalog.(interface{ Close() })
	if !ok {
		return fmt.Errorf("catalog does not support close")
	}
	closer.Close()
	return nil
}

func NewMessageCatalog(cfg Config) (MessageCatalog, error) {
	if cfg.CtxLanguageKey == "" {
		cfg.CtxLanguageKey = "language"
	}
	if cfg.DefaultLanguage == "" {
		cfg.DefaultLanguage = "en"
	}
	if cfg.NowFn == nil {
		cfg.NowFn = time.Now
	}
	if cfg.ObserverBuffer <= 0 {
		cfg.ObserverBuffer = 1024
	}
	if cfg.StatsMaxKeys <= 0 {
		cfg.StatsMaxKeys = 512
	}
	if cfg.ReloadRetries < 0 {
		cfg.ReloadRetries = 0
	}
	if cfg.ReloadRetryDelay <= 0 {
		cfg.ReloadRetryDelay = 50 * time.Millisecond
	}

	dmc := DefaultMessageCatalog{
		cfg: cfg,
		stats: catalogStats{
			languageFallbacks: map[string]int{},
			missingLanguages:  map[string]int{},
			missingMessages:   map[string]int{},
			templateIssues:    map[string]int{},
			droppedEvents:     map[string]int{},
			maxKeys:           cfg.StatsMaxKeys,
		},
	}
	err := dmc.loadFromYaml()
	if err == nil {
		dmc.startObserverWorker()
	}

	return &dmc, err
}
