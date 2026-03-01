# Dallas — History

## Project Context

Agent Deck is a Go CLI/TUI tool for managing multiple AI coding agent sessions via tmux. Built with charmbracelet (bubbletea/bubbles/lipgloss), SQLite (modernc.org/sqlite), and WebSockets. Currently adding GitHub Copilot CLI as a first-class tool alongside Claude Code, Gemini CLI, and OpenCode.

**User:** Matthew Corven
**Module:** `github.com/asheshgoplani/agent-deck`
**Go version:** 1.24+

## Learnings

<!-- Append system patterns, file paths, and insights below -->

### 2026-03-01T02:07:35Z — Task B: Storage Pipeline for Copilot CLI (Phase 2)

Added `CopilotSessionID` and `CopilotDetectedAt` fields across the full storage pipeline, following the established Codex field pattern. Files touched:

- **`internal/statedb/migrate.go`** — Added fields to `toolDataBlob` (int64 timestamps), `jsonInstanceData` (time.Time), `MigrateFromJSON` (mapping + time conversion), `MarshalToolData` (2 new positional params at end + body assignment + time conversion), `UnmarshalToolData` (2 new return values at end + extraction).
- **`internal/session/storage.go`** — Added fields to `InstanceData` struct, updated `MarshalToolData` call site (2 new args), updated both `UnmarshalToolData` call sites (LoadLite + LoadWithGroups) with new return vars + struct fields, updated `convertToInstances` Instance restore mapping.
- **`internal/session/instance.go`** — Added `CopilotSessionID` and `CopilotDetectedAt` to the `Instance` struct (required for the storage.go call sites and B10 mapping).

**Key pattern:** `MarshalToolData`/`UnmarshalToolData` use positional parameters — new tool params MUST go at the END (after codex, before latestPrompt) to preserve ordering. The `toolDataBlob` stores timestamps as `int64` Unix seconds; the public-facing structs use `time.Time`. Conversion happens in both directions with `.IsZero()` / `> 0` guards.

**Total:** ~11 insertion sites across 3 files, full project builds clean.
