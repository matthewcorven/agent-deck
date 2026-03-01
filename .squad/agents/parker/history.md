# Parker — History

## Project Context

Agent Deck is a Go CLI/TUI tool for managing multiple AI coding agent sessions via tmux. Built with charmbracelet (bubbletea/bubbles/lipgloss), SQLite (modernc.org/sqlite), and WebSockets. Currently adding GitHub Copilot CLI as a first-class tool alongside Claude Code, Gemini CLI, and OpenCode.

**User:** Matthew Corven
**Module:** `github.com/asheshgoplani/agent-deck`
**Go version:** 1.24+

## Copilot CLI Integration

Design docs at `docs/plans/copilot-cli/` — 7 phases (0–6).
Binary: `copilot` (standalone, `brew install copilot-cli@prerelease` or `npm install -g @github/copilot`)

## Learnings

<!-- Append Copilot integration patterns, decisions, and file paths below -->

- **2026-02-24:** Updated `docs/plans/copilot-cli/phase-0-live-capture.md` with additional captures per Ripley's recommendations: CLI metadata section (2a: --help, --version, ~/.copilot/ dir structure), MCP server output and pane title rows in the states table, dual-mode capture (plain text + ANSI escapes), and expanded commit/exit criteria. These captures feed Phase 2 (command builder flags), Phase 3 (session ID detection), and status detection patterns.

- **2026-02-24 — CLI vs SDK/ACP Deep Analysis:**
  - Researched three "SDK" concepts: ACP (Agent Client Protocol), Copilot Language Server (`@github/copilot-language-server`), and Copilot Extensions SDK (`copilot-extensions/preview-sdk.js`).
  - **Copilot Extensions SDK is NOT relevant** — it's for building server-side extensions inside Copilot Chat, not for controlling Copilot from external tools.
  - **ACP is the relevant programmatic approach** — `copilot --acp --stdio` / `copilot-language-server --acp` provides JSON-RPC structured communication: `session/new`, `session/load`, `session/prompt`, `session/update`, `session/cancel`, `session/request_permission`.
  - ACP offers superior status detection (structured events vs tmux scraping) and richer control (cancel, permission flows, tool call visibility), but requires implementing fs provider, terminal provider, permission UI, and a Go JSON-RPC bidi client — essentially mini-IDE capabilities.
  - ACP mode is "Preview", protocol v1, SDK at v0.14.1 (TypeScript-only, no Go SDK). Listed in ACP Registry as "GitHub Copilot 1.430.0".
  - **Key finding:** Dual-process hybrid (CLI for display + ACP sidecar for status) doesn't work — they'd be separate sessions with separate contexts.
  - **Recommendation:** CLI for v1 (~37h), ACP as Phase 7+ alternative mode (~89h for ACP track). Decision filed at `.squad/decisions/inbox/parker-copilot-cli-vs-sdk.md`.
  - Full analysis: `docs/plans/copilot-cli/copilot-cli-vs-sdk-analysis.md`

- **2026-02-25 — Cross-agent update (Scribe):** Phase 0 hard gate RESOLVED. Ripley chose filesystem-based session ID strategy — scan `~/.copilot/session-state/*/workspace.yaml`, match by cwd/git_root + creation time. Dual-ID model (workspace UUID + session UUID). Fallback: `--continue` flag. **Phase 3 is now unblocked.** See `docs/plans/copilot-cli/phase-0-findings.md` and decision in `.squad/decisions.md`.

- **2026-03-01 — Phase 1 (Config Surface) implemented:**
  - `CopilotSettings` struct added to `internal/session/userconfig.go` with fields: Command, YoloMode, DefaultModel, DefaultAgent, ConfigDir, EnvFile. `GetCommand()` defaults to "copilot".
  - `Copilot CopilotSettings` field added to `UserConfig` struct (after Codex, TOML tag `copilot`).
  - `"copilot"` added to builtins map in `GetCustomToolNames()` and to `GetToolIcon()` switch → 🛸.
  - `[copilot]` section added to `CreateExampleConfig()` example string.
  - `IconCopilot = "🛸"` constant added to `internal/ui/styles.go`. `ToolIcon()` and `ToolColor()` both updated (color: `#6e40c9` GitHub purple).
  - Copilot added to: `buildPresetCommands()` in newdialog.go, `toolNames`/`toolValues` in settings_panel.go, `toolOptions` in setup_wizard.go.
  - `StatusProvider` interface + `ToolStatus` type created in `internal/session/status_provider.go` (~40 lines). Constants prefixed `ToolStatus*` to avoid collision with existing `Status` string type in instance.go.
  - `skills/agent-deck/references/config-reference.md` updated: TOC, full `[copilot]` section, built-in icons list, complete example.
  - **Key collision found:** `instance.go` already defines `StatusIdle`/`StatusError` as `Status` (string). Used `ToolStatusIdle`/`ToolStatusError` (int) to avoid name collision. Lambert's pre-written test file `status_provider_test.go` uses unprefixed names — needs updating by Lambert.
  - **Dual icon pattern confirmed:** `GetToolIcon()` in userconfig.go AND `ToolIcon()` in styles.go both need copilot — done in both.

- **2026-03-01T02:05:16Z — Phase 2, Task A: CopilotOptions added to tooloptions.go:**
  - `CopilotOptions` struct with 6 fields: SessionMode, ResumeSessionID, Model, Agent, YoloMode (*bool, nil=inherit), ConfigDir.
  - `ToolName()` → "copilot", `ToArgs()` (session mode switch + all flags), `ToArgsForFork()` (omits session mode flags).
  - `NewCopilotOptions(config)` populates from `config.Copilot.*` (YoloMode, DefaultModel, DefaultAgent, ConfigDir).
  - `UnmarshalCopilotOptions(data)` follows exact `UnmarshalCodexOptions` pattern (ToolOptionsWrapper → tool check → unmarshal).
  - Pattern notes: Copilot uses `--continue`/`--resume` (long flags, not `-c`/`-r` like Claude), `--yolo` (not `--dangerously-skip-permissions`), `--model`/`--agent` (long flags like OpenCode's `-m`/`--agent`).
  - Clean compile confirmed. File: `internal/session/tooloptions.go`.

- **2026-03-01T02:14:48Z — Phase 2, Task C: Instance fields, command builder, and all dispatch points:**
  - **C1:** Added `CopilotStartedAt int64` to Instance struct (Dallas had already added CopilotSessionID/CopilotDetectedAt).
  - **C2:** `buildCopilotCommand()` — follows buildOpenCodeCommand pattern: env prefix, AGENTDECK env vars, copilot binary from config.Copilot.GetCommand(), resume via `--resume {id}` with tmux set-environment, extra flags from buildCopilotExtraFlags.
  - **C3:** `buildCopilotExtraFlags()` — follows buildOpenCodeExtraFlags pattern (single config load): yolo, model, agent, config-dir from GetCopilotOptions() with NewCopilotOptions fallback.
  - **C4:** `GetCopilotOptions()` / `SetCopilotOptions()` — follows exact GetCodexOptions/SetCodexOptions pattern using UnmarshalCopilotOptions.
  - **C5/C6:** Start() and StartWithMessage() — added `case "copilot"` dispatching to buildCopilotCommand + CopilotStartedAt timestamp.
  - **C7:** Restart() respawn-pane block — tmux env recovery for COPILOT_SESSION_ID, resume or fresh start, respawn-pane.
  - **C8:** Restart() fallback switch — added copilot to both the if/else-if resume chain and the fresh-start switch.
  - **C9:** CanRestart() — copilot can always restart (with or without session ID).
  - **C10:** UpdateStatus hook fast path — added `"copilot"` to hook freshness check and waiting-status branch.
  - **C10b:** UpdateStatus hook session ID sync — added `case "copilot"` to update CopilotSessionID/CopilotDetectedAt from hookSessionID.
  - **C11:** UpdateHookStatus — added `case "copilot"` for session ID sync from hook payload with tmux env sync.
  - **C12:** PostStartSync — added `case "copilot"` as no-op (async detection like Codex).
  - **Additional:** hookFastPathFreshnessForTool — added copilot to the codex-style freshness check. UpdateCopilotSession() — tmux env read method (filesystem detection deferred to Phase 3). SyncSessionIDsToTmux — added COPILOT_SESSION_ID sync. UpdateStatus active/waiting tracking — added copilot to Codex-style session tracking call.
  - **Verification:** 6 `case "codex"` → 6 `case "copilot"`. Clean `go build ./...`. All dispatch points covered by grep -n cross-check.

- **2026-03-01T02:39:57Z — Phase 3: Copilot Session Detection + Resume:**
  - Implemented 6 new functions/methods in `internal/session/instance.go`:
    1. `DetectCopilotSession()` — public wrapper for restored sessions.
    2. `detectCopilotSessionAsync()` — retry loop (1s init sleep, 3 attempts at 0/1s/2s). On success: sets CopilotSessionID, CopilotDetectedAt, syncs to tmux env COPILOT_SESSION_ID.
    3. `getCopilotHomeDir()` — resolves config directory: CopilotSettings.ConfigDir > COPILOT_HOME env > ~/.copilot.
    4. `queryCopilotSession(excludeIDs, allowUnscoped)` — walks `~/.copilot/session-state/*/workspace.yaml`, YAML-unmarshal with `copilotWorkspaceYAML` struct, matches by `normalizePath(cwd)` or `normalizePath(git_root)` against ProjectPath. Filters by CopilotStartedAt, excludes other sessions' IDs. Returns best scoped match (most recent) or unscoped fallback.
    5. `collectOtherCopilotSessionIDs()` — enumerates tmux sessions, reads COPILOT_SESSION_ID from each, returns exclusion map.
    6. Updated `UpdateCopilotSession()` — added filesystem fallback after tmux env check: when CopilotSessionID empty and CopilotStartedAt > 0, calls queryCopilotSession(allowUnscoped=false) and syncs back to tmux env.
  - Added `go i.detectCopilotSessionAsync()` dispatch in both `Start()` and `StartWithMessage()` (after existing Codex blocks).
  - Added `gopkg.in/yaml.v3` import (already indirect in go.mod, now direct in instance.go).
  - Clean `go build ./...` and `go vet ./internal/session/...`.
  - **Architecture:** workspace UUID from workspace.yaml `id` field IS the resume key stored in CopilotSessionID. buildCopilotCommand() already uses it for `--resume`. No new Instance struct fields needed.
  - **Key pattern differences from Codex:** Codex uses JSONL walking with regex UUID extraction from filenames; Copilot uses directory-based walking with YAML parsing. Codex scans date-organized dirs; Copilot scans flat UUID dirs under session-state/.
