# Lambert — History

## Project Context

Agent Deck is a Go CLI/TUI tool for managing multiple AI coding agent sessions via tmux. Built with charmbracelet (bubbletea/bubbles/lipgloss), SQLite (modernc.org/sqlite), and WebSockets. Currently adding GitHub Copilot CLI as a first-class tool alongside Claude Code, Gemini CLI, and OpenCode.

**User:** Matthew Corven
**Module:** `github.com/asheshgoplani/agent-deck`
**Go version:** 1.24+
**Test framework:** github.com/stretchr/testify

## Learnings

<!-- Append test patterns, coverage gaps, and quality insights below -->

### 2026-03-01T01:45:07Z — Phase 1 Config Surface Tests

**Tests written (5):**
- **T1** `internal/ui/setup_wizard_test.go`: Updated `TestSetupWizard_ToolOptions` — added `"copilot"` to `expectedTools` (PASS)
- **T2** `internal/ui/newdialog_test.go`: Updated `TestDialogPresetCommands` — added `"copilot"` to `expectedCommands` (PASS)
- **T3** `internal/session/userconfig_test.go`: Added `TestCopilotSettings_GetCommandDefault`, `TestCopilotSettings_GetCommandCustom`, `TestCopilotSettings_YoloModeDefaultsFalse` (blocked by T5 build error — tests are correct)
- **T4** `internal/session/userconfig_test.go`: Added `TestCopilotSettings_TOMLRoundtrip`, `TestCopilotSettings_EmptySection` — table-driven TOML parsing tests covering all 6 fields (blocked by T5 build error — tests are correct)
- **T5** `internal/session/status_provider_test.go`: Created — `TestToolStatus_String`, `TestToolStatus_StringCoversAllValues`, `TestToolStatus_UnknownIsZeroValue` (BLOCKED — see below)

**Blocker found:** Parker's `status_provider.go` declares `StatusIdle` and `StatusError` as `ToolStatus` (int iota) constants, but `instance.go` already defines them as `Status` (string) constants. This name collision prevents the entire `internal/session` package from compiling tests. The fix belongs to Parker — the `ToolStatus` constants likely need a prefix like `ToolStatusIdle` or the existing `Status` constants need renaming. T3, T4, T5 are all blocked by this.

**Patterns observed:**
- Existing tests use raw `toml.DecodeFile` with temp files, not `LoadUserConfig()` — avoids cache/HOME coupling
- Setup wizard tests reference `wizard.toolOptions` directly (unexported field, same package)
- New dialog tests check `presetCommands` slice ordering — position-sensitive, not set-based
- `GetCommand()` uses pointer receiver on `CopilotSettings` — tests use addressable local vars so auto-addressing works
- The project does not use testify in UI tests — only stdlib `testing`

### 2026-02-28 — Cross-agent: ToolStatus naming collision resolved

**Source:** Coordinator (via Parker's decision)
The `ToolStatus` constant naming collision (T3/T4/T5 blocker) was resolved by the coordinator. Constants renamed from `StatusIdle`/`StatusError`/etc to `ToolStatusIdle`/`ToolStatusError`/etc. Lambert's test file `status_provider_test.go` was updated to match. All 5 tests now pass green. Decision recorded in `.squad/decisions/decisions.md`.
