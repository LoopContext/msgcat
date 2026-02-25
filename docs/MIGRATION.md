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

## 7) Backward compatibility notes

- Context key lookup remains compatible with both typed key and plain string key.
- Existing template syntax remains valid.
- `LoadMessages` system-code constraint (`9000-9999`) is unchanged.
