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
