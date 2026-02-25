### 2026-02-24: StatusProvider Interface Required in Phase 1

**Author:** Ripley (Lead) · **Status:** Enforced

A `StatusProvider` interface must be defined in `internal/session/status_provider.go` during Phase 1 to future-proof the ACP transition path.

```go
type StatusProvider interface {
    Status() ToolStatus           // busy, idle, prompting, error
    LastActivity() time.Time
    SessionID() (string, bool)    // (id, detected)
}
```

**Rationale:** Without this abstraction, every consumer of status data (UI status bar, conductor orchestration, notify daemon, web streaming) would need to be touched when ACP arrives in Phase 7+. With it, ACP becomes a drop-in `StatusProvider` implementation. The CLI implementation wraps tmux pattern matching; the future ACP implementation wraps `session/update` event subscription.

**Scope:** ~30 lines. Interface + `ToolStatus` type with constants (`Unknown`, `Busy`, `Idle`, `Prompting`, `Error`). No consumer rewiring needed in Phase 1 — wiring happens in Phase 4/5.

**Updated:** `docs/plans/copilot-cli/phase-1-config-surface.md` (new Task #6 + Exit Criteria)
