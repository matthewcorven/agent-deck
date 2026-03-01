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

### 2026-02-28 — Phase 2 Tests: Copilot Command Builder, Options, and Storage

**Tests written (22 across 3 files):**

**`internal/session/tooloptions_test.go` (13 tests):**
- `TestCopilotOptions_ToolName` — verifies ToolName() returns "copilot"
- `TestCopilotOptions_ToArgs` — 11-case table-driven: empty, new, continue, resume+ID, resume-no-ID, model, agent, yolo-true, yolo-false, config-dir, all-combined
- `TestCopilotOptions_ToArgsForFork` — 6-case table-driven: empty, session-mode-excluded, model+agent, yolo, config-dir, all-non-session-flags
- `TestCopilotOptions_MarshalUnmarshal` — full round-trip via ToolOptionsWrapper
- `TestNewCopilotOptions` — from config with DefaultModel, DefaultAgent, YoloMode, ConfigDir
- `TestNewCopilotOptions_NilConfig` — nil config returns sane defaults
- `TestUnmarshalCopilotOptions_EmptyData` — nil and empty byte slice
- `TestUnmarshalCopilotOptions_WrongTool` — claude data returns nil for copilot unmarshal

**`internal/session/instance_test.go` (5 tests):**
- `TestBuildCopilotCommand_New` — fresh start, no session ID, verifies env vars and no --resume
- `TestBuildCopilotCommand_Resume` — CopilotSessionID set → --resume ID + tmux set-environment
- `TestBuildCopilotCommand_WithOptions` — model + agent + yolo + config-dir present in output
- `TestBuildCopilotCommand_CustomCommand` — non-matching base command passes through with env prefix
- `TestBuildCopilotCommand_NonCopilotTool` — non-copilot tool returns command unmodified
- `TestGetSetCopilotOptions` — full round-trip via JSON, including clear-on-nil

**`internal/statedb/migrate_test.go` (4 tests, NEW FILE):**
- `TestMarshalUnmarshalToolData_CopilotFields` — round-trip with CopilotSessionID + CopilotDetectedAt, verifies other fields survive
- `TestMarshalUnmarshalToolData_CopilotEmpty` — empty Copilot fields round-trip as zero values
- `TestMarshalUnmarshalToolData_CopilotWithToolOptions` — Copilot fields + ToolOptions JSON survive together
- `TestUnmarshalToolData_EmptyData` — nil data returns zero values

**Patterns observed:**
- `buildCopilotCommand` follows the same env-prefix + binary detection pattern as Gemini, but uses `tmux set-environment COPILOT_SESSION_ID` for resume (no uuidgen/capture-resume pattern like Claude)
- `buildCopilotExtraFlags` reads from GetCopilotOptions() then falls back to config defaults — tests need SetCopilotOptions() to exercise all flags
- Pre-existing test failures are all OpenCode E2E (require running OpenCode) and conductor plist (requires agent-deck in PATH) — unrelated to Copilot work
- `boolPtr` helper lives in `gemini_yolo_test.go` and is available package-wide
- statedb had no `migrate_test.go` — created as new file for MarshalToolData/UnmarshalToolData round-trip coverage

### 2026-03-01T02:44:40Z — Phase 3 Tests: Copilot Session Detection + Resume

**Tests written (14 in `internal/session/instance_test.go`):**

**queryCopilotSession (9 tests):**
- `TestQueryCopilotSession_MatchingProject` — matching cwd returns correct workspace UUID
- `TestQueryCopilotSession_NonMatchingProject` — different cwd returns ""
- `TestQueryCopilotSession_TimeWindowFiltering` — workspace.yaml older than CopilotStartedAt is skipped
- `TestQueryCopilotSession_MultipleCandidates` — two matching → most recently modified wins
- `TestQueryCopilotSession_ExcludeIDs` — excluded session ID is skipped
- `TestQueryCopilotSession_AllowUnscopedFallback` — no cwd/git_root → returned only when allowUnscoped=true
- `TestQueryCopilotSession_EmptyDirectory` — no session-state dir → ""
- `TestQueryCopilotSession_CorruptYAML` — invalid YAML skipped gracefully, no panic
- `TestQueryCopilotSession_MissingIDField` — no id field → falls back to directory name as ID

**getCopilotHomeDir (3 tests):**
- `TestGetCopilotHomeDir_Default` — returns ~/.copilot when no overrides
- `TestGetCopilotHomeDir_EnvOverride` — COPILOT_HOME env var takes precedence over default
- `TestGetCopilotHomeDir_ConfigOverride` — UserConfig copilot.config_dir wins over env var (highest priority)

**Resume integration (2 tests):**
- `TestCopilotResume_WithSessionID` — CopilotSessionID set → `--resume` in command output
- `TestCopilotResume_ContinueFallback` — no CopilotSessionID → fresh command, no `--resume`

**Patterns observed:**
- `getCopilotHomeDir()` reads config from `~/.agent-deck/config.toml` (via `GetAgentDeckDir()`), not `~/.config/agent-deck/`
- `queryCopilotSession()` falls back to directory name as ID when `workspace.yaml` has no `id` field — important for robustness
- Time window filtering uses directory mtime via `entry.Info()`, controlled in tests via `os.Chtimes`
- `t.Setenv("COPILOT_HOME", tmpDir)` + `ClearUserConfigCache()` is the standard isolation pattern for all getCopilotHomeDir tests
- Unscoped fallback (no cwd/git_root) is tracked separately from scoped matches — scoped always wins

### 2026-03-01T03:03:14Z — Phase 4 Tests: Copilot Status Detection Patterns + Tool Detection

**Tests written (5 functions, 22 cases across 2 files):**

**`internal/tmux/patterns_test.go` (4 functions, 16 cases):**
- `TestDefaultRawPatterns_Copilot` — sanity check: busy/prompt non-empty, no SpinnerChars, no WhimsicalWords (PASS)
- `TestCopilotBusyPatterns` — 6-case table: ◉ Thinking, ◐ Running, ◐ streaming, ∙ Planning all match busy; idle prompt and shell output do NOT (PASS)
- `TestCopilotPromptPatterns` — 4-case table: normal idle, plan mode idle match prompt; thinking and shell do NOT (PASS)
- `TestCopilotPatternsNoCollision` — 5-case: generic shell, Claude busy, Gemini prompt, Codex prompt, OpenCode busy content must NOT trigger copilot busy patterns (PASS)

**`internal/tmux/tmux_test.go` (1 function + 2 test expansions, 6 cases):**
- `TestDetectToolFromCommand` — added 3 copilot cases: bare "copilot", "--resume abc123", full path "/usr/local/bin/copilot" (PASS)
- `TestDetectToolFromContentCopilot` — 4-case: idle prompt → copilot, busy state → copilot, plan mode → copilot, path-containing-word → shell (2 FAIL — see blockers)

**Blockers for Parker:**
1. `detectToolFromContent` uses `\bcopilot\b` which is too broad — matches "copilot" in paths like "copilot-project" (false positive, `-` is a word boundary)
2. `detectToolFromContent` doesn't have copilot-specific UI patterns — "◉ Thinking\nEsc to cancel" returns "shell" instead of "copilot"
3. Fix: use copilot UI-specific patterns (like Claude does with "claude code", "trust the files") — e.g. `Type @ to mention files`, `Esc to cancel` combined with state icon regex

**Patterns observed:**
- Copilot patterns in patterns.go (DefaultRawPatterns) are correct and well-structured — state icons regex + string patterns
- Tool detection in tmux.go needs the same specificity approach Claude uses: no bare tool name matching in content, only UI-specific signatures
- All 4 pattern tests pass green because patterns.go copilot case is properly implemented

### 2026-03-01T03:07:25Z — Cross-agent: Phase 4 tool detection fix applied

Coordinator resolved the `\bcopilot\b` false-positive flagged in `TestDetectToolFromContentCopilot`. Replaced with state-icon regex `(?m)^[◉◐◎∙]\s` in `toolDetectionPatterns`. All 22 Phase 4 test cases now pass. Decision merged to decisions.md by Scribe.
