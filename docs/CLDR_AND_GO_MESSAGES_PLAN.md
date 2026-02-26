# CLDR plurals and messages in Go – design plan

Plan to add **CLDR-style plural forms** and **messages defined in Go with extract** to msgcat, closing the gap with go-i18n.

**Implementation status:** Done. Library has `ShortForms`/`LongForms`/`PluralParam` on `RawMessage`, **inline multi-form plural syntax** `{{plural:count|one:...|other:...}}`, `internal/plural` for form selection, `MessageDef` type, CLI extract finds MessageDef literals and merges into YAML, merge copies plural fields. See README, examples/cldr_plural, examples/msgdef.

---

## 1. CLDR plurals

### 1.1 Goal

Support the full set of CLDR plural categories (**zero**, **one**, **two**, **few**, **many**, **other**) so that languages with more than two forms (e.g. Arabic, Russian, Welsh) can have correct pluralization. Support both optional entry-level maps and **inline multi-form tokens**.

### 1.2 Backward compatibility

- Keep existing **short** / **long** strings and **`{{plural:count|singular|plural}}`** (binary) unchanged. No breaking change.
- Add **inline multi-form support**: `{{plural:count|one:item|other:items}}` uses the language rule to pick a form.
- Add **optional** plural form maps. When present, they are used instead of the binary plural token for that entry.

### 1.3 YAML format (optional plural forms)

Per entry in `set`, allow optional **short_forms** and **long_forms** maps from CLDR form name to template:

```yaml
set:
  items.count:
    short: "You have {{count}} {{plural:count|item|items}}"   # fallback / simple case
    long: "Total: {{num:amount}} items"
    # Optional CLDR forms (when set, used instead of short/long for this key when plural count is provided)
  person.cats:
    short_forms:
      one: "{{.Name}} has {{.Count}} cat."
      other: "{{.Name}} has {{.Count}} cats."
    long_forms:
      one: "{{.Name}} has one cat."
      other: "{{.Name}} has {{.Count}} cats."
    plural_param: count   # which param drives plural selection (default: "count")
```

- **Form names:** `zero`, `one`, `two`, `few`, `many`, `other` (CLDR standard).
- **plural_param:** optional; name of the param used for plural selection (default `"count"` when forms are present). Omitted when only `short`/`long` and binary `{{plural:...}}` are used.
- If **short_forms** / **long_forms** are present, resolution uses the plural param and the resolved language to pick a form (see below), then renders that template with the same named params as today.

### 1.4 Library types

- **RawMessage** (add optional fields):
  - `ShortForms map[string]string` `yaml:"short_forms,omitempty"` — keys: zero, one, two, few, many, other.
  - `LongForms  map[string]string` `yaml:"long_forms,omitempty"`
  - `PluralParam string` `yaml:"plural_param,omitempty"` — default `"count"` when forms are used.

- **Plural form selector:** add a small dependency or internal package that, given **(language tag, count)** returns the CLDR form for that locale (e.g. `en` + 1 → `one`, 5 → `other`; `ar` + 0 → `zero`, 1 → `one`, 2 → `two`, 3–10 → `few`, 11–99 → `many`, other → `other`). Options:
  - **A)** Depend on **golang.org/x/text** and use its plural/message support if it exposes form selection.
  - **B)** Vendor or copy the **go-i18n internal/plural** approach (generated rules from CLDR).
  - **C)** Minimal **internal/plural** in msgcat: embed a compact table of (locale → rule) and a small evaluator (operands + rule AST). More work but no new dependency.

- **Resolution:** When looking up a message:
  - If the entry has **ShortForms** (and **LongForms**): get **plural_param** from params (default `"count"`); if missing, fall back to **short**/long and existing binary plural token behavior. Otherwise compute CLDR form from (resolvedLang, count), then pick **ShortForms[form]** (or **ShortForms["other"]** if form missing); same for long. Render the chosen template with **renderTemplate** as today.
  - If the entry has only **short**/**long**, behavior is unchanged (including `{{plural:count|...}}`).

### 1.5 Edge cases

- **Missing form:** If the chosen form (e.g. `few`) is not in the map, fall back to `other`, then to `short`/`long` if no forms.
- **Missing plural param:** If **short_forms** is set but the plural param is missing in params, fall back to **short**/long (and binary plural token if present).
- **Invalid count type:** Same as today for binary plural: observer + optional strict placeholder.

### 1.6 Merge and CLI

- **merge:** When building translate files, copy **short_forms** / **long_forms** / **plural_param** from source to placeholder entries so translators can fill per-form strings.
- **extract:** Keys-only and sync modes unchanged; MessageDef extraction (see below) can emit **short_forms** / **long_forms** when defined in Go.

---

## 2. Messages in Go + extract

### 2.1 Goal

Let developers define message **content** in Go (like go-i18n’s `i18n.Message` with One/Other), and have **msgcat extract** discover those definitions and write them into the source YAML. So “messages in Go” is the source of truth for content at extract time; at runtime the catalog still loads from YAML (or from runtime-loaded messages).

### 2.2 New type: MessageDef

A struct that mirrors what can live in the catalog (key + short/long or plural forms + optional code). Used in Go for definition and for extract; not required at runtime for normal lookup.

```go
// MessageDef defines a message that can be extracted to YAML or used as default content.
// Use with msgcat extract to generate or update source message files from Go.
type MessageDef struct {
    Key         string            // Message key (e.g. "person.cats"). Required.
    Short       string            // Short template (or use ShortForms for CLDR).
    Long        string            // Long template (or use LongForms for CLDR).
    ShortForms  map[string]string  // Optional CLDR forms: zero, one, two, few, many, other.
    LongForms   map[string]string
    PluralParam string            // Param name for plural selection (default "count").
    Code        OptionalCode      // Optional code (e.g. CodeInt(404)).
}
```

- **Key** is required. **Short**/**Long** or **ShortForms**/**LongForms** (or both; forms take precedence when plural param is present).
- **PluralParam** defaults to `"count"` when ShortForms/LongForms are used.

### 2.3 Where MessageDef appears in Go

- **Standalone literals** for extract:
  ```go
  var personCats = msgcat.MessageDef{
      Key: "person.cats",
      ShortForms: map[string]string{
          "one":  "{{.Name}} has {{.Count}} cat.",
          "other": "{{.Name}} has {{.Count}} cats.",
      },
      LongForms: map[string]string{
          "one":  "{{.Name}} has one cat.",
          "other": "{{.Name}} has {{.Count}} cats.",
      },
  }
  ```
- **Slices/maps** of MessageDef (e.g. `[]msgcat.MessageDef{ ... }`, `map[string]msgcat.MessageDef`) so extract can find multiple definitions in one place.

- **Optional future:** Allow passing a `*MessageDef` as default to `GetMessageWithCtx` when the key is missing (like go-i18n’s DefaultMessage). Not required for “messages in Go + extract.”

### 2.4 Extract command extension

- **Current behavior:** Find keys from `GetMessageWithCtx` / `WrapErrorWithCtx` / `GetErrorWithCtx`; optionally sync those keys into source YAML with empty short/long.
- **New behavior:** Also find **msgcat.MessageDef** (and `*msgcat.MessageDef`) struct literals:
  - In variable declarations, slice literals, map literals.
  - Extract **Key**, **Short**, **Long**, **ShortForms**, **LongForms**, **PluralParam**, **Code** (handling string/int for Code like OptionalCode).
- **Merge with sync:** When running **extract -source en.yaml -out en.yaml**:
  - Keys from API calls: add missing keys with empty short/long (current behavior).
  - MessageDef literals: add or update entries by **Key** with the extracted Short/Long/ShortForms/LongForms/PluralParam/Code. So Go becomes the source of truth for those entries’ content in the generated YAML.

### 2.5 Implementation sketch (extract)

- In the same AST walk that finds API calls, also find:
  - **CompositeLit** whose type is `msgcat.MessageDef` or `*msgcat.MessageDef` (via selector from msgcat import).
  - **KeyValueExpr** in map literals whose value type is MessageDef.
  - **CompositeLit** elements in slice literals whose element type is MessageDef.
- For each MessageDef literal, collect Key and the rest; build a list of “message definitions from Go.”
- In sync mode: when writing the source YAML, for each such definition set `set[Key]` to the corresponding RawMessage (Short/Long or ShortForms/LongForms, PluralParam, Code). Keys-only from API calls that are not in any MessageDef still get empty short/long.

### 2.6 YAML output for MessageDef

- If MessageDef has **ShortForms**/LongForms, write **short_forms** / **long_forms** (and **plural_param** if not default) in the YAML.
- If it has only **Short**/Long, write **short** / **long** as today.
- **code** written per existing OptionalCode rules.

---

## 3. Implementation order

1. **CLDR plural selector (internal or dependency)**  
   Implement or depend on a function `Form(lang string, count int) string` returning one of zero/one/two/few/many/other. Add tests for a few locales (en, ar, ru).

2. **RawMessage plural forms**  
   Add **ShortForms**, **LongForms**, **PluralParam** to RawMessage. In **GetMessageWithCtx**, when these are set, get plural param from params, compute form, select template, render. Keep existing short/long and `{{plural:...}}` unchanged when forms are not set.

3. **MessageDef type and extract**  
   Add **MessageDef** in the library. Extend CLI extract to find MessageDef literals and collect Key + content. In sync mode, merge these into source YAML (add/update by Key).

4. **Merge and docs**  
   Ensure merge copies short_forms/long_forms/plural_param. Document CLDR forms and MessageDef + extract in README and CLI workflow plan.

---

## 4. Out of scope (for this plan)

- **Changing existing binary plural token:** `{{plural:count|singular|plural}}` stays as-is (backward compatible). Added **multi-form plural token** `{{plural:count|one:singular|other:plural|few:...}}` for inline CLDR rules.
- **Runtime default message from Go:** Passing a MessageDef into GetMessageWithCtx as fallback when key is missing is a possible later extension.
- **Hash/change detection** for merged translations (like goi18n) remains out of scope.

---

This plan is the single source of truth for adding CLDR plurals and “messages in Go + extract” to msgcat.
