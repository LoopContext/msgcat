# Conversion Plan: String Keys + Named Parameters

Plan for converting msgcat from **numeric message codes** and **positional template parameters** to **string message keys** and **named parameters**. No backward compatibility required.

---

## 1. Summary of changes

| Area | Current | Target |
|------|---------|--------|
| Message lookup key | `int` (e.g. `1`, `2`) | `string` (e.g. `"greeting.hello"`, `"error.not_found"`) |
| YAML `set` keys | Numeric: `1:`, `2:` | String: `"greeting.hello":`, `"error.template":` |
| Template params | Positional: `{{0}}`, `{{1}}`, `{{plural:0\|...}}` | Named: `{{name}}`, `{{plural:count\|...}}` |
| API params | `msgParams ...interface{}` (ordered) | `params Params` (`map[string]interface{}`) |
| Returned `Message.Code` | `msgCode + Group` | Per-entry `RawMessage.Code` (optional; 0 if unset) |
| `Messages.Group` | Used to compute display code | **Removed** (no longer needed) |
| Observer / stats | `msgCode int` | `msgKey string` |

---

## 2. Type and struct changes

### 2.1 `structs.go`

- **Messages**
  - `Set`: `map[int]RawMessage` → `map[string]RawMessage`
  - **Remove** `Group int` (display code comes from `RawMessage.Code` per entry).

- **RawMessage**
  - Keep: `LongTpl`, `ShortTpl` (YAML: `long`, `short`).
  - **Code**: keep `int`; meaning is “numeric code for API/HTTP response”. In YAML and `LoadMessages`, set per entry; if 0, returned `Message.Code` can be 0 when message is found, or use a sentinel when missing.
  - No `Key` field: the map key is the message key.

- **Message** (return type)
  - Unchanged: `LongText`, `ShortText`, `Code int` (still the value to expose to API clients).

- **Observer**
  - `OnMessageMissing(lang string, msgCode int)` → `OnMessageMissing(lang string, msgKey string)`
  - `OnTemplateIssue(lang string, msgCode int, issue string)` → `OnTemplateIssue(lang string, msgKey string, issue string)`

- **Params type** (for named parameters)
  - Add: `type Params map[string]interface{}` (replaces the unused `MessageParams` struct, or keep both and have API accept `Params`).
  - Callers use: `msgcat.Params{"name": "juan", "count": 3}`.

### 2.2 `msgcat.go` (internal types)

- **DefaultMessageCatalog**
  - `runtimeMessages`: `map[string]map[int]RawMessage` → `map[string]map[string]RawMessage`.

- **observerEvent**
  - `msgCode int` → `msgKey string`.

- **catalogStats**
  - Stat keys today: `"lang:code"` (e.g. `"en:2"`). Change to `"lang:msgKey"` (e.g. `"en:error.not_found"`). No type change; keys are already strings.

- **MessageParams**
  - Remove `MessageParams struct { Params map[string]interface{} }` and use `type Params = map[string]interface{}` (or keep a type alias only).

---

## 3. YAML format

### 3.1 Before (current)

```yaml
group: 0
default:
  short: Unexpected error
  long: Unexpected error was received and was not found in catalog
set:
  1:
    short: Hello short description
    long: Hello veeery long description.
  2:
    short: Hello template {{0}}, this is nice {{1}}
    long: Hello veeery long {{0}} description. Details {{1}}.
  4:
    short: "You have {{0}} {{plural:0|item|items}}"
    long: "Total: {{num:1}} generated at {{date:2}}"
```

### 3.2 After (target)

```yaml
default:
  short: Unexpected error
  long: Unexpected error was received and was not found in catalog
set:
  greeting.hello:
    short: Hello short description
    long: Hello veeery long description.
  greeting.template:
    code: 1002   # optional: numeric code for API/HTTP
    short: Hello template {{name}}, this is nice {{detail}}
    long: Hello veeery long {{name}} description. Details {{detail}}.
  items.count:
    short: "You have {{count}} {{plural:count|item|items}}"
    long: "Total: {{num:amount}} generated at {{date:generatedAt}}"
```

- **Key format**: recommend `[a-zA-Z0-9_.-]+` (e.g. `greeting.hello`, `error.not_found`). Reject empty or invalid keys in validation.
- **Optional `code`**: if present, use as `Message.Code`; if absent, use 0 (or define a convention).

---

## 4. Template syntax (named only)

- **Simple**: `{{name}}` — identifier `[a-zA-Z_][a-zA-Z0-9_]*` (or allow dots: `{{user.name}}`).
- **Plural**: `{{plural:count|singular|plural}}` — same as today but first token is param name.
- **Number**: `{{num:amount}}` — param name instead of index.
- **Date**: `{{date:when}}` — param name instead of index.

Regexes (conceptually):

- Simple: `\{\{([a-zA-Z_][a-zA-Z0-9_.]*)\}\}`
- Plural: `\{\{plural:([a-zA-Z_][a-zA-Z0-9_.]*)\|([^|}]*)\|([^}]*)\}\}`
- Number: `\{\{num:([a-zA-Z_][a-zA-Z0-9_.]*)\}\}`
- Date: `\{\{date:([a-zA-Z_][a-zA-Z0-9_.]*)\}\}`

Parsing: extract **name** (string) instead of **index** (int). In `renderTemplate`, resolve with `params[name]`. Missing param: same behavior as today (observer + optional strict placeholder like `<missing:name>`).

---

## 5. Public API

### 5.1 Catalog interface

```go
type MessageCatalog interface {
	LoadMessages(lang string, messages []RawMessage) error
	GetMessageWithCtx(ctx context.Context, msgKey string, params Params) *Message
	WrapErrorWithCtx(ctx context.Context, err error, msgKey string, params Params) error
	GetErrorWithCtx(ctx context.Context, msgKey string, params Params) error
}
```

- `Params` is `map[string]interface{}` or `type Params map[string]interface{}`. Nil treated as empty (no params).
- All call sites: `catalog.GetMessageWithCtx(ctx, "greeting.hello", nil)` or `catalog.GetMessageWithCtx(ctx, "greeting.template", msgcat.Params{"name": "juan", "detail": "nice"})`.

### 5.2 LoadMessages

- **RawMessage** for runtime load must include the **key**; since the slice is not a map, each item needs an explicit key:
  - Add **Key** to **RawMessage** (used only for `LoadMessages`; YAML key is the key when loading from file).
  - So: `RawMessage` has `Key string` (optional in YAML/unmarshal; required when used in `LoadMessages`).
  - Actually: in YAML the map key is the message key, so when unmarshaling we get key from the map. For `LoadMessages(lang, []RawMessage)` we need each RawMessage to carry its key. So add `Key string` to RawMessage. When loading from YAML, after unmarshal we have map[string]RawMessage — the key is in the map. When building from YAML we don’t need to set RawMessage.Key. When loading via LoadMessages we do: each RawMessage in the slice must have Key set (and optionally Code, ShortTpl, LongTpl).
  - **Restriction for LoadMessages**: only allow keys with a reserved prefix (e.g. `sys.`) so YAML and runtime don’t collide. So: `Key` must be non-empty and for LoadMessages must start with `sys.` (or similar).
  - Validation: `Key` format `[a-zA-Z0-9_.-]+`, and for LoadMessages `strings.HasPrefix(key, "sys.")`.

---

## 6. Internal implementation outline

### 6.1 Normalization and validation

- **normalizeAndValidateMessages**
  - Remove Group validation.
  - Ensure `Set` is `map[string]RawMessage`.
  - For each key in Set: validate key format (non-empty, allowed chars); set `raw.Code` from YAML `code` if present (else leave 0); no longer set `raw.Code = code` from numeric key.

### 6.2 loadFromYaml merge

- When merging runtime messages into `messageByLang[lang]`, iterate `runtimeSet` as `map[string]RawMessage` and assign `msgSet.Set[key] = msg` for each key.

### 6.3 GetMessageWithCtx

- Signature: `(ctx, msgKey string, params Params)`.
- Resolve language as today.
- Lookup: `langMsgSet.Set[msgKey]`; if missing, call `onMessageMissing(resolvedLang, msgKey)` and use default message, set `Message.Code = CodeMissingMessage`.
- If found: use `raw.ShortTpl`, `raw.LongTpl`, and `Message.Code = raw.Code` (0 if not set).
- Call `renderTemplate(resolvedLang, msgKey, shortMessage, params)` (and same for long); pass `params` as `map[string]interface{}`.

### 6.4 renderTemplate

- Signature: `(lang, msgKey string, template string, params map[string]interface{}) string`.
- Replace placeholders by **name**: for each token, parse name (string), then `v, ok := params[name]`; if !ok, report missing and use replaceMissing; else use value for plural/num/date/simple.
- All four regex passes use name-based lookup.

### 6.5 Observer and stats

- observerEvent: `msgKey string`.
- incrementMissingMessage(lang, msgKey string): key e.g. `lang + ":" + msgKey`.
- incrementTemplateIssue(lang, msgKey string, issue string): same.
- Observer interface: already updated in structs (OnMessageMissing(lang, msgKey string), OnTemplateIssue(lang, msgKey string, issue string)).

### 6.6 LoadMessages

- Require each RawMessage to have `Key` set and `Key` prefixed with `sys.` (or chosen prefix).
- Store: `langMsgSet.Set[msg.Key] = normalizedMessage`, `dmc.runtimeMessages[normalizedLang][msg.Key] = normalizedMessage`.
- No longer check numeric Code range; optional Code in RawMessage for API response.

### 6.7 RawMessage.Key and YAML

- When unmarshaling YAML, we have `map[string]RawMessage`. The key is the message key; we don’t need to put it inside RawMessage for YAML. So RawMessage.Key is only needed for **LoadMessages** (slice of RawMessage). So:
  - In YAML, RawMessage does not need a `key` field; the map key is the key.
  - In Go, when building RawMessage for LoadMessages, caller sets Key. So RawMessage has `Key string` (used when adding via LoadMessages; can be empty when coming from YAML).

---

## 7. Constants and errors

- **CodeMissingMessage**, **CodeMissingLanguage**: keep as int; still used as `Message.Code` when message or language is missing.
- **SystemMessageMinCode / MaxCode**: no longer used for LoadMessages; replace with “key must have prefix `sys.`” (or keep constant for doc purposes and use for nothing).
- **newCatalogError(code, ...)**: unchanged; still takes int code (from Message.Code).

---

## 8. File-by-file checklist

| File | Changes |
|------|--------|
| **structs.go** | Messages.Set → map[string]RawMessage; remove Group; RawMessage add Key (optional in YAML); Observer signatures; add Params type. |
| **msgcat.go** | Regexes for named placeholders; observerEvent.msgKey; catalogStats keys by msgKey; runtimeMessages map[string]map[string]RawMessage; normalizeAndValidateMessages (string keys, optional code); loadFromYaml merge by string key; renderTemplate(lang, msgKey, template, params map[string]interface{}); GetMessageWithCtx(ctx, msgKey string, params Params); WrapErrorWithCtx, GetErrorWithCtx; LoadMessages by RawMessage.Key with sys.* validation; remove MessageParams struct if replaced by Params. |
| **error.go** | No change (still int code). |
| **test/suites/msgcat/resources/messages/*.yaml** | String keys; named placeholders; remove group; optional code per entry. |
| **test/suites/msgcat/msgcat_test.go** | All GetMessageWithCtx(..., code, a, b, c) → GetMessageWithCtx(..., "key", Params{...}); LoadMessages with RawMessage{Key: "sys.xxx", ...}; Observer expectations with msgKey string. |
| **test/mock/msgcat.go** | Regenerate (or manually) interface with msgKey string, params Params. |
| **msgcat_fuzz_test.go** | Use string keys and Params. |
| **msgcat_bench_test.go** | Use string keys and Params. |
| **examples/** | Use string keys and Params. |
| **docs/** | Update CONTEXT7*.md, README, etc., with new API and YAML format. |

---

## 9. Implementation order

1. **Types (structs.go)**  
   Messages.Set string key, remove Group, RawMessage.Key + optional Code in YAML, Observer, Params.

2. **Catalog storage and YAML (msgcat.go)**  
   DefaultMessageCatalog maps to string key; readMessagesFromYaml / normalizeAndValidateMessages for string keys and optional code; loadFromYaml merge by string key.

3. **Named template engine (msgcat.go)**  
   New regexes; parse by name; renderTemplate(lang, msgKey, template, params map[string]interface{}).

4. **Public API (msgcat.go)**  
   GetMessageWithCtx(ctx, msgKey string, params Params); Wire nil params; WrapErrorWithCtx / GetErrorWithCtx; use raw.Code for Message.Code.

5. **LoadMessages (msgcat.go)**  
   RawMessage.Key required; validate sys.* (or chosen prefix); store by Key.

6. **Observer and stats (msgcat.go)**  
   observerEvent.msgKey; stats keys lang:msgKey; Observer callbacks with msgKey.

7. **Tests and examples**  
   Update YAML fixtures, tests, mocks, fuzz, bench, examples.

8. **Docs**  
   README, CONTEXT7, CONVERSION_PLAN cross-links.

---

## 10. Key naming convention (recommendation)

- Use dot-separated segments: `domain.concept` (e.g. `greeting.hello`, `error.validation`, `order.status.pending`).
- For runtime-only messages: prefix `sys.` (e.g. `sys.overloaded`, `sys.maintenance`).
- Validation: allow `[a-zA-Z0-9_.-]+`, reject empty.

---

## 11. Params type and nil

- Define `type Params map[string]interface{}`.
- In GetMessageWithCtx / WrapErrorWithCtx / GetErrorWithCtx: if params is nil, pass empty map to renderTemplate so templates see no params (missing placeholders get missing-param behavior).

This plan is the single source of truth for the total conversion to string keys and named parameters.
