# CLI workflow plan: extract + merge

Plan for adding a **msgcat** CLI that closes the workflow gap vs go-i18n: **extract** (discover message keys from code and/or YAML) and **merge** (prepare translation files from a source language).

**Goals**

- Developers can discover which message keys are used in Go and ensure the source catalog has them.
- Translators get per-language files that list only missing/empty messages with source text as placeholder.
- Optional **group** (int or string) in YAML for organization (e.g. `group: "api"` or `group: 0`); CLI and library support it.
- CLI is additive; library gets a small, backward-compatible extension for group.

**Scope**

- New package or module: `cmd/msgcat` (or `internal/msgcatcli`) producing a `msgcat` binary.
- Two subcommands: `extract`, `merge`.
- Message format: current msgcat YAML (`default` + `set` with optional `code`) plus optional **group** (int or string)—see §7.
- Small library extension for group support so YAML and CLI stay in sync.

---

## 1. Extract command

**Purpose:** Discover message keys referenced in Go and optionally sync them into the source language YAML so every used key exists (with empty or placeholder content for new keys).

### 1.1 Behavior

- **Scan Go code** for message key string literals in:
  - `GetMessageWithCtx(ctx, "key", ...)`
  - `WrapErrorWithCtx(ctx, err, "key", ...)`
  - `GetErrorWithCtx(ctx, "key", ...)`
- Keys are the first string literal argument to these functions. Support both direct string literals and concatenation of string literals (optional).
- **Option A (keys only):** Output the unique set of keys (e.g. one per line to stdout, or to a file). Use case: “what keys does the code use?” or input to tooling.
- **Option B (sync to source YAML):** Read the source language file (e.g. `en.yaml`), add any key that appears in Go but not in `set`; new entries get empty `short`/`long` (or a comment/placeholder). Write back to source file or `-out`. Use case: “after adding new GetMessageWithCtx calls, update en.yaml so translators see new keys.”

We can implement both: e.g. `extract -keys` prints keys; `extract -source en.yaml -out en.yaml` syncs keys into that file.

### 1.2 Implementation sketch

- Walk directories for `*.go` (respect `-exclude` for vendor, etc.).
- Skip `_test.go` unless `-include-tests` (default: skip).
- Use `go/ast` to find call expressions whose selector is `GetMessageWithCtx`, `WrapErrorWithCtx`, or `GetErrorWithCtx` and package is `msgcat` (or configurable import path). Extract first string argument (handle basic `+` concatenation if desired).
- Dedupe keys.
- **Keys-only mode:** print keys (e.g. sorted) to stdout or `-out`.
- **Sync mode:** parse source YAML (reuse or mirror msgcat’s `Messages` struct), merge in missing keys with empty short/long, write YAML (preserve order/comment where feasible or at least valid YAML).

### 1.3 Flags (proposed)

| Flag | Description |
|------|-------------|
| `paths` | Directories or files to scan (default: `.`) |
| `-out` | Output file (keys mode: one key per line; sync mode: YAML path) |
| `-source` | Source language YAML path (enables sync mode; e.g. `resources/messages/en.yaml`) |
| `-format` | For keys: `keys` (one per line) or `yaml` (minimal YAML stub). For sync: ignored, output is YAML. |
| `-include-tests` | Include `_test.go` files (default: false) |
| `-msgcat-pkg` | Import path for msgcat (default: `github.com/loopcontext/msgcat`) so we detect the right calls |

---

## 2. Merge command

**Purpose:** From a source language file (e.g. `en.yaml`), produce per-language **translate** files that contain every key from the source; for keys missing or empty in the target, use source short/long as placeholder so translators can fill them.

### 2.1 Behavior

- **Input:**
  - One **source** message file (e.g. `en.yaml`). All keys from this file define the canonical set.
  - Optional **target** message files (e.g. `es.yaml`, `fr.yaml`). If a target file exists, we use it to prefill already-translated entries.
- **Output:**
  - One file per target language: **`translate.<lang>.yaml`** (e.g. `translate.es.yaml`).
  - Content: same structure as msgcat YAML (`default` + `set`). For each key in source:
    - If target has a non-empty `short` and `long` for that key, use target’s content (considered translated).
    - Otherwise, use source’s `short`/`long` as placeholder (and optional `code` from source).
  - So translators open `translate.es.yaml`, see English where Spanish is missing, and replace with Spanish. When done, they **rename or copy** `translate.es.yaml` → `es.yaml` for use at runtime. No separate “active” file in msgcat’s loader; the directory just has `en.yaml`, `es.yaml`, etc.

### 2.2 Edge cases

- **Target file missing:** Treat as “no translations yet”; output `translate.<lang>.yaml` with all keys from source (source content as placeholder).
- **Key in target but not in source:** Optionally drop (so translate file is “keys we care about”) or keep (document in plan). Recommend: drop so translate file is exactly “keys from source that need translation.”
- **default block:** Copy source `default` into each translate file so the file is valid; translators can replace with localized default.
- **code field:** Copy from source when creating placeholder entries; if target had a value, keep target’s (or always use source for consistency—document choice).

### 2.3 Implementation sketch

- Parse source YAML into a structure that matches msgcat’s `Messages` (default + set).
- For each target language (from `-targetLangs` and/or from existing `*.yaml` in a directory):
  - Parse target file if present.
  - Build merged `set`: for each key in source, if target has non-empty short and long, use target; else use source.
  - Write `translate.<lang>.yaml` with merged content (and default from source).
- Language tag comes from filename (e.g. `es.yaml` → `es`) or from `-targetLangs es,fr`.

### 2.4 Flags (proposed)

| Flag | Description |
|------|-------------|
| `-source` | Source message file (e.g. `resources/messages/en.yaml`) |
| `-targetLangs` | Comma-separated target language tags (e.g. `es,fr`). If not set, can infer from `-targetDir` *.yaml (excluding source and translate.*). |
| `-targetDir` | Directory containing target YAMLs (e.g. `resources/messages`). Optional; can pass target files as positional args. |
| `-outdir` | Where to write `translate.<lang>.yaml` (default: same dir as source or `.`) |
| `-translatePrefix` | Filename prefix for translation files (default: `translate.`) so output is `translate.es.yaml`. |

---

## 3. Workflow summary

**Initial setup**

1. Maintain source language (e.g. `en.yaml`) with all messages.
2. Run: `msgcat extract -source resources/messages/en.yaml -out resources/messages/en.yaml` when new keys are added in code (or run extract -keys and add keys manually).

**Adding a new language (e.g. Spanish)**

1. Run: `msgcat merge -source resources/messages/en.yaml -targetLangs es -outdir resources/messages`
2. Get `resources/messages/translate.es.yaml` with all keys and English placeholders.
3. Translators fill in Spanish.
4. Rename/copy `translate.es.yaml` → `es.yaml` in the same directory. msgcat loads `es.yaml` at runtime.

**Adding new keys to en.yaml later**

1. Update `en.yaml` (or run extract -source to add keys from Go).
2. Run merge again: `msgcat merge -source resources/messages/en.yaml -targetDir resources/messages -outdir resources/messages`
3. New keys appear in existing `translate.es.yaml` (and other targets) with English placeholders; existing translations are preserved in the merge output.

**Optional: validate keys in code vs YAML**

1. `msgcat extract -out keys.txt` (keys only).
2. Compare keys.txt to keys in en.yaml to find “in code but not in catalog” or “in catalog but not in code.”

---

## 4. Implementation order

0. **Optional group (library)**  
   Add `OptionalGroup` type (unmarshal/marshal int or string), add optional `Group` to `Messages`. No runtime behavior change. Tests for YAML round-trip. Enables CLI to preserve group in extract/merge.

1. **Extract (keys only)**  
   AST walk, collect keys from the three API calls, output unique list. Tests with a few fixture .go files.

2. **Extract (sync to source YAML)**  
   Parse msgcat YAML (reuse types or duplicate minimal struct in CLI to avoid coupling), add missing keys with empty short/long, preserve `group`, write YAML. Tests with in-memory YAML.

3. **Merge**  
   Parse source + target YAMLs, copy source `group` into translate files, build translate.*.yaml per target, write to outdir. Tests with fixture YAMLs.

4. **CLI wiring**  
   `cmd/msgcat/main.go` with subcommands extract/merge, flags, and clear usage. Install with `go install github.com/loopcontext/msgcat/cmd/msgcat@latest`.

5. **Docs**  
   README section “CLI workflow (extract & merge)”, link from main README; add "Optional group"; optional CONTEXT7 update.

---

## 5. File layout

- **Option A:** `cmd/msgcat/` in the same repo (same module). Binary is `msgcat`. Dependencies: only stdlib + YAML parser (and go/ast). Prefer not to depend on msgcat package for parsing so CLI works even if YAML structs are internal; we can duplicate minimal YAML structs in the CLI or use a generic map + marshal.
- **Option B:** Separate module `github.com/loopcontext/msgcat/cmd/msgcat` or `github.com/loopcontext/msgcat-cli`. Same repo is simpler; same module keeps one go.mod.

Recommendation: **same repo, same module**, `cmd/msgcat/` with minimal duplication of YAML structures (or import msgcat and use its `Messages`/`RawMessage` if they stay public and we don’t pull in heavy deps). If we want zero dependency on msgcat at parse time, the CLI can define its own `messagesDoc` struct for YAML and produce the same format.

---

## 6. Out of scope (for this plan)

- **CLDR plurals:** Merge does not need to understand plural forms; msgcat’s current `{{plural:count|singular|plural}}` is preserved as literal strings in YAML.
- **Hash / change detection:** We could add optional hash of source content per key to detect “translation was for old version” (like goi18n). Defer to a later iteration.
- **Other formats (TOML/JSON):** Only YAML in/out for now to match msgcat.

---

## 7. Optional group (int or string)

**Purpose:** Allow message files or entries to be tagged with a **group** that can be either an integer or a string (e.g. `group: "api"` or `group: 0`). Use for organization, filtering, or tooling—e.g. all API errors in group `"api"`, or numeric groups for legacy systems.

### 7.1 Library (msgcat) changes

- **Type `OptionalGroup`**  
  Same pattern as `OptionalCode`: a type that unmarshals from **int** or **string** in YAML. Internal representation can be string (e.g. `0` → `"0"`, `"api"` → `"api"`); marshal back as string, or preserve kind so that numeric input round-trips as `group: 0` and string as `group: "api"` (implementation choice).

- **Where group lives**
  - **File-level (recommended):** Add optional `Group OptionalGroup` to **`Messages`**. In YAML, top-level `group: "api"` or `group: 0` applies to the whole file. One group per file is the common case.
  - **Per-entry (optional):** Add optional `Group OptionalGroup` to **`RawMessage`**. In YAML, each key in `set` can have `group: "api"` or `group: 0` to override or sub-categorize. Implement if needed after file-level is done.

- **Runtime behavior**  
  The catalog does not interpret group; it is only stored and available for tooling or future use (e.g. filtering, export). No change to `Message` or `GetMessageWithCtx` return type unless we later add a way to expose group (e.g. `Message.Group`). For this plan, adding the field to the YAML struct and parsing is enough.

- **YAML example (file-level)**

  ```yaml
  group: api
  default:
    short: Unexpected error
    long: ...
  set:
    error.not_found:
      short: Not found
      long: ...
  ```

  Or numeric:

  ```yaml
  group: 0
  default:
    short: Unexpected error
  set:
    greeting.hello:
      short: Hello
      long: ...
  ```

### 7.2 CLI behavior

- **Extract:** When reading/writing source YAML (sync mode), preserve the existing `group` field. When creating a new source file from keys only, omit group (or add a default) per project preference.
- **Merge:** When building translate files, copy the source file’s `group` into each output `translate.<lang>.yaml` so the translated file has the same group as the source. If per-entry group is added later, copy from source entry when creating placeholders.

### 7.3 Implementation notes

- **OptionalGroup** can live in the same package as `OptionalCode` (e.g. `code.go` or new `group.go`). UnmarshalYAML: accept `int`, `int64`, `string`; store as string for simplicity. MarshalYAML: if the string is numeric (e.g. `strconv.Atoi` succeeds), emit as int for readability; otherwise emit as string. That gives `group: 0` and `group: "api"` round-trip.
- Validation: no uniqueness or allowed-values check; group is opaque to the library.

---

This plan is the single source of truth for implementing the extract/merge CLI workflow and optional group support in msgcat.
