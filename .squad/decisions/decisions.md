# Decisions

## 2026-03-01: ToolStatus Constants Use `ToolStatus` Prefix to Avoid Collision

**Author:** Parker (Integration Dev) · **Status:** Decided

The `StatusProvider` interface's `ToolStatus` constants are prefixed `ToolStatus*` (e.g., `ToolStatusIdle`, `ToolStatusError`) instead of the shorter `Status*` form. This is because `instance.go` already defines `StatusIdle` and `StatusError` as constants of the `Status` (string) type. Using unprefixed names causes a redeclaration compile error.

**Rationale:** Go doesn't support overloaded constants across types in the same package. The `ToolStatus` prefix is clearer anyway — it distinguishes session-level status (`Status`) from tool-level status detection (`ToolStatus`).

---

## 2026-03-01: Phase 1 Config Surface — Work Plan Approved

**Author:** Ripley (Lead) · **Status:** Decided

Completed detailed code review of all Phase 1 target files. Produced exact file/line references for all 18 change sites across 7 files. Identified 3 locations the spec missed: `GetToolIcon()` in userconfig.go, `GetCustomToolNames()` builtins map, and `default_tool` comment strings. Confirmed `prepareCommand()` and `focusTarget` systems do NOT impact Phase 1.

**Rationale:** Parker needs precise insertion points to avoid merge conflicts and misplaced code. The spec's approximate line numbers were off by 5-15 lines due to the v0.19.19 upstream merge.

---

## 2026-02-28: Upstream Review #2 — v0.19.14→v0.19.19 (32 commits, 180 files)

**Author:** Ripley (Lead) · **Status:** Action required — merge upstream + update phase docs

Reviewed all 32 upstream commits across 180 files. Key changes: Docker sandbox subsystem (new `internal/docker/`, 2362 lines), ANSI-capture architecture change (`-e` flag + `StripANSI()` pipeline), tool detection refactored into standalone functions with `toolDetectionOrder` array, `applyWrapper()`→`prepareCommand()` signature change (now returns 3 values), Recent Sessions feature (statedb schema v2), NewDialog focus system rewritten to `focusTarget` enum, notifications minimal mode, `SetupConductor()` now 7 params, `ShowDeleteSession()` now 3 params.

**Rationale:** Delayed merging increases conflict surface and makes our phase implementations target stale code. Phase 2 (command builder) and Phase 4 (status detection) are most impacted.
