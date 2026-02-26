# Migration Guide

## Scope

This guide covers migration to the current production profile (`v1.x`) of `msgcat`.

## 1) Update module import

No change if you already use:

```go
import "github.com/loopcontext/msgcat"
```

## 2) Recommended config migration

Old:

```go
msgcat.Config{
  ResourcePath: "./resources/messages",
}
```

New recommended:

```go
msgcat.Config{
  ResourcePath:      "./resources/messages",
  CtxLanguageKey:    "language",
  DefaultLanguage:   "en",
  FallbackLanguages: []string{"es"},
  StrictTemplates:   true,
  ObserverBuffer:    1024,
  StatsMaxKeys:      512,
  ReloadRetries:     2,
  ReloadRetryDelay:  50 * time.Millisecond,
}
```

## 3) Observer behavior change

Previous mental model: observer callbacks ran inline.

Current behavior:
- callbacks are dispatched asynchronously,
- callback panic is recovered,
- queue overflow increments `stats.DroppedEvents`.

Action: if you depended on synchronous ordering, move that logic outside observer callbacks.

## 4) Stats behavior change

Current behavior includes capping:
- per-map key count is limited by `StatsMaxKeys`;
- overflow events are grouped as `__overflow__`.

Action: dashboards should include both explicit keys and `__overflow__`.

## 5) Reload behavior change

`Reload` now retries transient failures.

Action:
- prefer atomic deploy of YAML files (`write temp` + `rename`),
- configure retries for your rollout pattern.

## 6) Lifecycle recommendation

If you configured an observer, call `msgcat.Close(catalog)` on shutdown to flush/stop worker goroutine cleanly.

## 7) Backward compatibility notes (v1.x pre–string-keys)

- Context key lookup remains compatible with both typed key and plain string key.
- (If you are on a release with **string message keys and named parameters**, see section 9 below.)

## 8) Go version (v1.0.8+)

Module requires **Go 1.26** or later. If you are on an older toolchain, upgrade before updating to msgcat v1.0.8.

---

## 9) Migration to string keys and named parameters (breaking)

If you are moving from numeric message codes and positional template parameters to the new API:

- **Message keys** — Use string keys everywhere (e.g. `"greeting.hello"`, `"error.not_found"`). Replace `GetMessageWithCtx(ctx, 1, a, b)` with `GetMessageWithCtx(ctx, "greeting.hello", msgcat.Params{"name": a, "detail": b})`.
- **Templates** — Replace `{{0}}`, `{{1}}` with named placeholders: `{{name}}`, `{{plural:count|item|items}}`, `{{num:amount}}`, `{{date:when}}`. Pass a single `msgcat.Params` map.
- **Code field** — Code is now optional and string. In YAML use `code: 404` or `code: "ERR_NOT_FOUND"`. In Go use `msgcat.CodeInt(503)` or `msgcat.CodeString("ERR_MAINT")`. Codes are not required to be unique. When code is empty, use `Message.Key` or `ErrorKey()` as the API identifier.
- **LoadMessages** — Each message must have `Key` with prefix `sys.` (e.g. `sys.alert`). No numeric code range; use `Code: msgcat.CodeInt(9001)` or `Code: msgcat.CodeString("SYS_LOADED")` if you need a code.
- **Observer** — `OnMessageMissing(lang, msgKey string)` and `OnTemplateIssue(lang, msgKey string, issue string)` now take string `msgKey` instead of `msgCode int`.
- **YAML** — Remove `group`. Use string keys under `set:` and optional `code` per entry. See README and `docs/CONVERSION_PLAN.md`.
