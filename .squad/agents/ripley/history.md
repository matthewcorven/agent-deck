# Ripley — History

## Project Context

Agent Deck is a Go CLI/TUI tool for managing multiple AI coding agent sessions via tmux. Built with charmbracelet (bubbletea/bubbles/lipgloss), SQLite (modernc.org/sqlite), and WebSockets. Currently adding GitHub Copilot CLI as a first-class tool alongside Claude Code, Gemini CLI, and OpenCode.

**User:** Matthew Corven
**Module:** `github.com/asheshgoplani/agent-deck`
**Go version:** 1.24+

## Learnings

<!-- Append architecture decisions, patterns, and insights below -->

### 2026-02-23 — PRD-Level Documentation Audit

**Finding:** No formal PRD, ARCHITECTURE.md, DESIGN.md, SPEC.md, or ADR directory exists at the repo root or in `docs/`. The project's architectural guidance is distributed across:

1. **README.md** — Product vision ("AI agent command center"), feature list, multi-tool support table, install/quick-start. Serves as the de facto product overview.
2. **docs/plans/2026-02-14-vagrant-mode-design.md** — Full design doc for Vagrant sandbox mode. 621+ lines covering architecture, MCP compatibility, crash recovery, testing strategy. Status: design complete, implementation pending.
3. **docs/plans/2026-02-17-copilot-cli-support.md** — Full design doc + viability assessment for Copilot CLI integration. Covers acceptance criteria, non-goals, command reference, 7-phase implementation plan. Status: Phase 0 blocking (live capture needed).
4. **docs/plans/copilot-cli/** — 7 phase implementation docs (phase-0 through phase-6) broken out for agent-friendly execution.
5. **.claude/plans/DECISIONS.md** — Architecture decisions for Vagrant mode (wrapper approach, forced skip-permissions, VM-aware MCP, health checks, crash recovery).
6. **skills/agent-deck/SKILL.md** — Operational reference (CLI commands, TUI shortcuts, MCP management, sub-agent launch, worktree workflows). Not a design doc but defines the public interface.
7. **skills/agent-deck/references/** — `config-reference.md`, `cli-reference.md`, `tui-reference.md`, `troubleshooting.md`. Operational docs, not architectural.
8. **CONTRIBUTING.md** — Dev setup, branch naming, PR process. Describes project structure briefly.
9. **CHANGELOG.md** — Detailed release history from v0.1.0 through v0.19.0. Rich source of implicit architectural decisions.
10. **llms.txt** — LLM-facing summary of the product.

**Key architectural constraints inferred from docs:**
- All tools follow the same integration pattern: config struct → command builder → status detection → session lifecycle
- Tools: Claude Code (full), Gemini CLI (full), OpenCode (status), Codex (status), custom tools via config
- State stored in SQLite (`internal/statedb/`), tool-specific data as JSON blob in `tool_data` column
- tmux is the session runtime; status detection via content scraping or hooks
- MCP pooling via Unix sockets; three scopes (LOCAL, GLOBAL, USER)
- New tool integrations must respect: `userconfig.go` (settings struct), `tooloptions.go` (options struct), `instance.go` (lifecycle), `tmux/patterns.go` (detection)

**Alignment with Copilot CLI plan:** The phased plan is well-aligned with existing patterns. No gaps found — it mirrors the Codex/OpenCode integration approach exactly. The parent design doc's acceptance criteria, non-goals, and viability assessment are thorough.

### 2026-02-24 — Phase 0 Execution Plan

Recommended the Phase 0 execution approach for Copilot CLI integration:
- **Matthew** performs manual captures (launch, session flow, exit, errors, `--help`, `--version`, `~/.copilot/`, MCP output, pane title)
- **Parker** drafts detection patterns from captured data
- **Ripley** gates session ID extraction strategy before Phase 1 proceeds
- Top risks: session ID detection (may not be exposed by Copilot CLI) and TUI rendering in tmux
- Advisory only — no decisions recorded, no code changes

### 2026-02-24 — Upstream Review (v0.19.2 → v0.19.14, 51 commits)

**Reviewed 59 files, +5873/−807 lines.** Full analysis written to `.squad/decisions/inbox/ripley-upstream-review.md`.

**Key findings affecting our Copilot integration:**
- `expandHomePath()` → `ExpandPath()` (renamed+extended, handles `$HOME/${VAR}`). All phase docs referencing old name must update.
- `resolveEnvFilePath()` → `resolvePath()` (renamed). Internal but our phase docs reference it.
- `expandTilde()` in storage.go REMOVED entirely. Replaced with `fixMalformedTildePath()` + `ExpandPath()`.
- `SetupConductor()` gained 6th parameter (`customPolicyMD`).
- `sendMessageWhenReady()` now accepts `"idle"` as ready state alongside `"waiting"`.
- New `GetLastResponseBestEffort()` — multi-fallback response retrieval. Must use for Copilot.
- New `CapturePaneFresh()` in tmux — bypasses cache for reliable snapshots. Critical for Phase 0 captures and Phase 4 status detection.

### 2026-02-28 — Phase 1 Config Surface: Execution Complete

Work plan approved and fully executed. All 18+ implementation sites completed by Parker, all 5 test items by Lambert. Naming collision with `Status` constants in `instance.go` was the only blocker — resolved by coordinator with `ToolStatus*` prefix convention. Phase 1 is done. Phase 2 (command builder) is next.
- New `resolveSessionCommand()` pipeline in CLI — our `detectTool("copilot")` must feed into this.
- New transition daemon/notifier system — automatically detects Copilot sessions once Phase 4 status patterns exist.
- New `SendSessionMessageReliable()` helper in send_helper.go.
- New `ManageMCPJson *bool` config field — demonstrates nil-pointer-with-default pattern for optional booleans.
- SQLite retry-on-busy wrapper (`migrateStateDBWithRetry`) protects against daemon contention.
- TUI shortcuts remapped: m=MCP, s=Skills, M=Move, r=Rename, R=Restart, S=Settings.

**Merge recommendation:** `git merge upstream/main` immediately. 4-5 file conflicts expected, all additive/resolvable. No rebase — preserves our commit history.

**Phase docs needing update:** Phase 0 (minor), Phase 1 (high — ExpandPath, resolveSessionCommand), Phase 2 (high — command resolution pipeline), Phase 3 (medium — session recovery chain), Phase 4 (medium — idle status, CapturePaneFresh, transition daemon).

### 2026-02-24 — CLI vs ACP Analysis Review

**Reviewed Parker's comparative analysis** (`docs/plans/copilot-cli/copilot-cli-vs-sdk-analysis.md`). Approved CLI-first recommendation (v1), ACP deferred to Phase 7+. Key architectural insights:

- **`StatusProvider` interface:** Requested this be added in Phase 1 to abstract status detection from tmux. Small change (~30 lines) that prevents a cross-cutting refactor when ACP arrives. Without it, UI/conductor/notify/web all couple directly to tmux pattern matching.
- **ACP effort is understated:** The 89h estimate doesn't account for the bidirectional nature of ACP — the client must *serve* methods (fs/terminal providers), not just call them. Realistic estimate: 100–110h.
- **Session ID detection is the critical gating risk:** ACP gives session IDs for free; CLI requires scraping/discovery. This remains the #1 unknown for Phase 0 and should block Phase 1 until resolved.
- **Upstream mitigants:** `CapturePaneFresh()` and `GetLastResponseBestEffort()` from v0.19.14 partially offset CLI pattern fragility — these should be cited in Phase 4 planning.

### 2026-02-24 — Phase 0 & Phase 1 Doc Updates (CLI vs SDK Review Items)

Incorporated two findings from the CLI vs ACP review into the phase docs:

1. **Phase 0 — Session ID Hard Blocker:** Added a prominent warning callout after the intro marking session ID detection as the #1 open risk and a hard blocker for Phase 3. Upgraded Task 3 to a "GATING DELIVERABLE" with explicit fallback strategies (synthetic IDs, `--continue`-only, defer Phase 3). Strengthened the exit criteria to mark the session ID item as a hard gate.

2. **Phase 1 — StatusProvider Interface:** Added a new file section (`internal/session/status_provider.go`) defining the `StatusProvider` interface and `ToolStatus` enum type. Interface has `Status()`, `LastActivity()`, and `SessionID()` methods. Added rationale explaining this prevents coupling to tmux pattern matching and makes ACP a drop-in implementation. Added 4 new exit criteria items.

### 2026-02-25 — Phase 0 Live Capture Analysis: Session ID Strategy Resolved

**GATING DELIVERABLE RESOLVED.** Analyzed Matthew's live Copilot CLI captures (`/session` modal, log file, `workspace.yaml`). Phase 3 is unblocked.

**Session ID detection strategy: Filesystem-based via `~/.copilot/session-state/*/workspace.yaml`.**
- `workspace.yaml` contains `id` (workspace UUID), `cwd`, `git_root`, `repository`, `branch` — YAML, trivially parseable.
- Mirrors the Codex `queryCodexSession()` pattern exactly: filesystem walk, file parsing, project path matching, time-window filtering.
- Fallback: `--continue` for resume when filesystem detection fails.

**Dual-ID model discovered:**
- **Workspace ID** — process-scoped UUID in `workspace.yaml`, created per `copilot` invocation. Use for directory-based session matching.
- **Session ID** — conversational session UUID, found as directory name containing `events.jsonl`. Use for resume/history operations.

### 2026-02-25 — Phase 0 TUI Capture Analysis: Pattern Strategy for Copilot CLI

Analyzed all 9 TUI state captures (Startup, Idle, Thinking, Tool, Plan, Error, MCP, PaneTitle, PaneTitle-renamed). Key architectural findings:

**Pattern strategy decided: String-primary, no SpinnerChars.** Copilot uses state icons (`◉◐◎∙`), not cycling spinners. These are per-state indicators — `◉` = loading/thinking, `◐` = streaming, `◎` = tool execution, `∙` = plan-mode thinking. Handled via regex `^[◉◐◎∙]\s` in BusyPatterns. No SpinnerChars/WhimsicalWords — would be architecturally misleading.

**Primary busy signal:** `Esc to cancel` — appears in EVERY active processing state. Highest confidence. Case: capital "E" (differs from Gemini's lowercase "esc to cancel").

**Primary idle signal:** `Type @ to mention files` — appears in both normal mode and plan mode idle states. Highest confidence.

**No pane title support:** Copilot CLI does not set a custom tmux pane title. Detection must rely entirely on content scraping. This is a limitation vs. tools that do set titles.

**Error marker:** `✗ Execution failed:` observed but cannot be captured by current `RawPatterns` struct (no ErrorPatterns field). TUI returns to idle after errors, so prompt pattern catches the transition. Error-specific status can be added in Phase 4 if `StatusProvider.Status()` needs a distinct `Error` state.

**Plan mode covered by shared pattern:** `Type @ to mention files` appears in both normal and plan mode idle prompts. No plan-mode-specific pattern needed. If distinct plan-mode detection is later required, `Describe a plan` can be added as a supplementary prompt pattern.

**Phase 0 is complete.** All captures done, session ID strategy resolved, patterns drafted. Phase 1 and Phase 2 are fully unblocked.
- Both are standard UUID v4 format. Instance struct needs `CopilotWorkspaceID` and `CopilotSessionID` fields.

**`/session` TUI modal is NOT viable for scraping** — it's a blocking full-screen overlay with a live clock, requires `Enter` to dismiss. Confirms filesystem detection is the only reliable non-interactive approach.

**Log file patterns discovered:**
- `[ERROR]` level on MCP events is misleading — these are verbose/debug messages, NOT actual errors. All MCP connections succeed.
- Startup sequence: ~1.4 seconds from first log line to ready. `Welcome {username}!` marks auth complete.
- MCP transports: stdio (aspire), npx-launched (playwright), remote HTTP/SSE (microsoft-learn, github-mcp-server), IDE socket (VS Code).
- `Using default model: claude-sonnet-4.6` logged 3x during startup — need dedup for analytics.
- Memory subsystem exists but disabled (`Memory enablement check: disabled`).

**Key file paths for Copilot integration:**
- Session state: `~/.copilot/session-state/{uuid}/workspace.yaml`
- Events: `~/.copilot/session-state/{uuid}/events.jsonl`
- Logs: `~/.copilot/logs/process-{timestamp}-{pid}.log`

**Findings written to:** `docs/plans/copilot-cli-captures/findings.md`
**Decision recorded:** `.squad/decisions/inbox/ripley-phase0-session-id-strategy.md`
**Phase 0 doc updated:** Tasks 1, 2a, 3 marked ✅. Exit criteria updated.

### 2026-02-28 — Phase 0 Resume Capture Analysis: All Ambiguities Closed

Analyzed 4 resume captures (Resume_11d97e41, Resume_155f69ab, plus ANSI variants). Phase 0 is now fully complete — no remaining open questions.

**Key findings:**
- **`--resume` accepts both workspace UUID and session UUID.** Both successfully restore the target session with full conversation history. The CLI's exit message canonically recommends the workspace UUID (`Resume this session with copilot --resume=<workspace-uuid>`), even when the session was resumed by session UUID. This confirms workspace UUID as the primary resume key for Agent Deck.
- **`--continue` confirmed working** — returns to the previous session (Matthew manual test).
- **Welcome banner is version-unstable:** Changed from "Describe a task to get started" (v0.0.418) to "Copilot uses AI. Check for mistakes." (v0.0.420). Dropped `"Describe a task to get started"` from PromptPatterns draft. The primary `"Type @ to mention files"` pattern is confirmed stable across both versions.

### 2026-02-28 — Phase 2 Design Review: Verification & Execution Plan

Verified Phase 2 doc (`phase-2-command-builder.md`) against current codebase. Found 7 discrepancies requiring doc amendment before implementation:

**Discrepancies found:**
1. **Line numbers are stale.** `buildGeminiCommand` is line 535 (doc says ~470), `buildCodexCommand` is line 687 (doc says ~616), `buildOpenCodeCommand` is line 610 (doc says ~531). Start() is 1408, StartWithMessage() is 1498, Restart() is 3223.
2. **Missing dispatch points.** Doc lists Start/StartWithMessage/Restart but omits 6 additional tool switch sites: UpdateStatus hook handling (lines 2010+2252), PostStartSync (line 2481), CanRestart (line 3629), CanFork (line 3672), and Instance restore in storage.go (line 686).
3. **Storage path underscoped.** Doc says "~4 files" but the full storage pipeline touches: `toolDataBlob` struct, `jsonInstanceData` struct, `MarshalToolData` (positional params), `UnmarshalToolData` (positional params + return values), `InstanceData` struct in storage.go, the `MarshalToolData` call (line 257), two `UnmarshalToolData` calls (lines 404, 486), and the Instance restore mapping (line 706). That's 4 files but 10+ insertion sites.
4. **`buildCopilotExtraFlags` is over-engineered.** The doc's version loads config twice (once for YoloMode fallback, once already done). Should follow the cleaner `buildOpenCodeExtraFlags()` pattern (single config load, options-then-fallback).
5. **No `buildCopilotCommandWithMessage` addressed.** Claude has a separate `buildClaudeCommandWithMessage` that embeds `-p "MESSAGE"` in the command. Copilot supports `-i "PROMPT"`. Doc says "use send-keys-after-ready" but doesn't specify if `-i` should be used instead for initial messages. Decision needed.
6. **`CopilotStartedAt` field type inconsistency.** Doc proposes `int64` (Unix millis) matching Codex/OpenCode pattern — this is correct and consistent.
7. **Doc's `buildCopilotCommand` includes AGENTDECK env vars inline** — this matches the Codex pattern correctly.

**Phase 1 verification: CopilotSettings confirmed present** in userconfig.go with all expected fields (Command, YoloMode, DefaultModel, DefaultAgent, ConfigDir, EnvFile) plus `GetCommand()` accessor.

**Key pattern to follow:** OpenCode is the closest template — has model/agent flags, session resume, extra-flags helper, async session detection. Copilot should mirror it.
- **events.jsonl structure mapped:** Event types include `session.start`, `user.message`, `assistant.turn_start/message/turn_end`, `tool.execution_start/complete`, `session.model_change`. `session.start` payload contains `sessionId`, `copilotVersion`, `context: {cwd, gitRoot, branch, repository}`. Parent-child chain via `parentId`. Valuable as an alternative/supplementary detection path to `workspace.yaml`.
- **New observable patterns:** Exit summary block (session time, code changes, resume hint), `IDE connection lost:`, `Error auto updating:`, user-aborted ops (`✗` + `Operation aborted by user`).

**Decisions recorded:**
- `.squad/decisions/inbox/ripley-resume-analysis.md` — workspace UUID is canonical for `--resume`
- `.squad/decisions/inbox/ripley-pattern-fragility.md` — drop "Describe a task to get started" from PromptPatterns

**Findings updated:** `docs/plans/copilot-cli-captures/findings.md` — §6 revised, §7 updated, §8 appended.

### 2026-03-01 — Phase 5 Preflight Design Review & Execution Plan

Verified Phase 5 design doc against current codebase. Key findings:

**Line number verification:**
- `Start()` is at line 1745 (not referenced in doc by line, OK)
- `StartWithMessage()` is at line 1843
- `buildCopilotCommand()` is at line 726
- `createSessionInGroupWithWorktreeAndOptions()` is at line 5115 (tool switch at ~5156)
- `Restart()` has copilot handling at lines 3891 and 3908

**Missing `case "copilot"` in home.go UI tool switch (P0 bug):** The switch at line ~5156 handles claude/gemini/aider/codex/opencode but NOT copilot. Sessions created via the new-session dialog with command="copilot" fall through to the custom tool check, which returns nil (no ToolDef for "copilot"), so the raw command is used and `tool` stays "shell". This breaks status detection, command building, resume, and session options. Missed in Phase 1. Must fix.

**Architectural decision: standalone `preflightCopilot()` function in instance.go.** Reasons:
1. Reusable pattern for future tool preflight checks
2. Follows project convention — all tool lifecycle logic in instance.go
3. No justification for a separate preflight module at this scope
4. Uses `exec.LookPath` with resolved command from `CopilotSettings.GetCommand()`
5. Error includes brew + npm install methods

**Settings panel (Task D): Deferred.** Panel has no per-tool prerequisite display. Sections are: Theme, Default Tool, Claude, Gemini, Codex, Updates, Logs, Global Search, Preview, Maintenance, MCP Servers & Custom Tools. Adding a Copilot section with just a prerequisite note would be inconsistent. Revisit when Copilot gets a settings-worthy option (e.g., YoloMode toggle).

**Error propagation path confirmed:** `Start()` returns error → `sessionCreatedMsg{err: err}` in UI → `h.setError(msg.err)` in home.go Update handler → displayed in TUI footer. The preflight error will surface cleanly through this existing path.

### 2026-03-01 — Phase 6 Execution Plan: Documentation, Polish, Enable by Default

Performed full codebase verification against all 7 Phase 6 tasks. Key findings:

**Tasks already complete (no work needed):**
- Task 2 (sample config): `[copilot]` section in `CreateExampleConfig()` has all 6 CopilotSettings fields. Perfect 1:1 match.
- Task 6 (feature flag removal): No feature flag ever existed. Copilot was always additive (no gating). The `internal/experiments/` package is unrelated (manages `agent-deck try` folders).

**Tasks needing work (5 total, all parallelizable):**
- Task 1A: README Multi-Tool Support table missing Copilot row (line ~268)
- Task 1B: README "The Problem" section doesn't mention Copilot (line 59)
- Task 3: Troubleshooting section needed in `skills/agent-deck/references/troubleshooting.md` (follows existing precedent of per-tool subsections in shared doc)
- Task 4: Session details panel in `home.go` is Claude-only (~L7835). CopilotSessionID is detected/stored but never rendered. Only functional code change in Phase 6.
- Task 7: CHANGELOG entry for v0.20.0

**Deferred:** Task 5 (experimental features `--experimental`, `--available-tools`) — optional, additive, doesn't block v1.

**Key architectural insight:** Session details panel is currently Claude-specific. Adding Copilot establishes a precedent for multi-tool detail panels. Future work should consider a tool-agnostic detail renderer rather than per-tool if-blocks.

**All verified integration points:**
- `internal/tmux/patterns.go:78` — copilot busy/prompt patterns ✅
- `internal/ui/styles.go:193,599,620` — icon (🛸) + color (#6e40c9 GitHub purple) ✅
- `internal/session/tooloptions.go:299-400` — CopilotOptions complete (6 fields, 5 methods) ✅
- `internal/session/userconfig.go:519-540` — CopilotSettings complete (6 fields) ✅
- `internal/session/instance.go:100-102` — CopilotSessionID, CopilotDetectedAt, CopilotStartedAt ✅
- `internal/ui/home.go:5151` — case "copilot" in session creation ✅
- All UI tool lists (setup wizard, new dialog, settings panel) include copilot ✅
- No TODOs/FIXMEs related to copilot in instance.go ✅

**Execution plan written to:** `.squad/plans/phase-6-execution-plan.md`

**Restart() also needs preflight:** `Restart()` at line 3607 calls `buildCopilotCommand()` at lines 3891/3908. If a user deletes the copilot binary between sessions, restart will fail cryptically. Same preflight check should guard restart.

### 2026-02-28 — Upstream Review #2 (v0.19.14 → v0.19.19, 32 commits, 180 files)

Full deep review of upstream divergence. 5 version bumps (v0.19.15–v0.19.19). Major new subsystems and API changes documented below.

**Major new subsystem: Docker Sandbox (`internal/docker/`)**
- Entirely new package: 9 files, +2362 lines. Container lifecycle management, config, detection, platform-specific keychain handling.
- `Instance` struct gained `Sandbox *SandboxConfig` and `SandboxContainer string` fields.
- `applyWrapper()` → `prepareCommand()` — **signature change from `(string, error)` to `(string, string, error)`**. Returns container name as 2nd value. All callers updated.
- New `wrapIgnoreSuspend()` function wraps commands in `bash -c 'stty susp undef; ...'` — disables Ctrl+Z.
- `Kill()` now handles sandbox container cleanup (force remove, keychain credential cleanup).
- `Start()` and `StartWithMessage()` both updated: new `containerName` variable, `buildTmuxOptionOverrides()` replaces inline tmux options, `RunCommandAsInitialProcess` field.
- `Restart()` updated: all resume paths use `prepareCommand()` instead of `applyWrapper()`.
- `IsSandboxed()` helper method on Instance.
- `NewSandboxConfig()` factory function.
- Phase 1/2 impact: We need `prepareCommand()` for Copilot too. Our command builder must return the same `(string, string, error)` tuple. Sandbox support is free if we follow the pattern.

**Major new feature: Recent Sessions**
- New `recent_sessions` table in statedb (SchemaVersion bumped to 2).
- `RecentSessionRow` struct with SHA-256 dedup key.
- `Storage.SaveRecentSession()` / `LoadRecentSessions()` methods.
- New dialog in `newdialog.go`: `recentSessions`, `showRecentPicker`, `dialogSnapshot`, `previewRecentSession()`.
- Home model loads recent sessions into picker on dialog open.
- Phase 1 impact: When we add Copilot to the tool picker, it automatically becomes available in the recent sessions picker — no extra work.

**ANSI handling architecture change (CRITICAL)**
- `CapturePane()` and `CapturePaneFresh()` now use `-e` flag instead of `-J`, returning raw ANSI output.
- All callers now call `tmux.StripANSI()` before pattern matching (status detection, readiness checks, preview content).
- New import: `github.com/charmbracelet/x/ansi` used in home.go for `ansi.Strip()`.
- Pattern: `rawContent, err := CapturePane/Fresh(); content := tmux.StripANSI(rawContent)`.
- **Phase 4 impact: HIGH.** Our Copilot status detection MUST follow this pattern — raw capture → strip ANSI → pattern match. All our BusyPatterns/PromptPatterns will work against clean text.

**Tool detection refactored**
- `detectToolFromCommand()` and `detectToolFromContent()` extracted as standalone functions in tmux.go.
- `toolDetectionOrder` array added for deterministic iteration order: `["claude", "gemini", "opencode", "codex"]`.
- Claude detection patterns tightened: no longer matches bare `(?i)claude` — now requires `\bclaude\s+code\b`, `no, and tell claude`, or `do you trust the files`.
- **Phase 2 impact: HIGH.** We MUST add `"copilot"` to `toolDetectionOrder` and add entries to `toolDetectionPatterns` map.

**Pane dead detection (new)**
- `PaneInfo` struct gained `Dead bool` field.
- `RefreshPaneInfoCache()` now queries `#{pane_dead}` + `#{window_index}` + `#{pane_index}`, only caching window 0, pane 0.
- `Session.IsPaneDead()` method added — uses cached info or direct tmux check.
- `GetStatus()` now checks `IsPaneDead()` before title/content detection — dead pane → "inactive".
- Phase impact: Our Copilot sessions get dead-pane detection for free via the status detection pipeline.

**NewDialog focus system rewritten**
- Old: integer focus indices (0=name, 1=path, 2=command, 3=branch/options).
- New: `focusTarget` enum type with named constants: `focusName`, `focusPath`, `focusCommand`, `focusWorktree`, `focusSandbox`, `focusInherited`, `focusBranch`, `focusOptions`.
- `rebuildFocusTargets()` dynamically builds focus order based on dialog state.
- **Phase 1 impact: MEDIUM.** If Copilot needs options in the dialog, we add them via the `focusOptions` target (same as Claude/Gemini/Codex). No integer index arithmetic.

**Notification system: minimal mode**
- `NewNotificationManager` gains 3rd param: `minimal bool`.
- Minimal mode: icon+count summary (e.g., `● 2 │ ◐ 3 │ ○ 1`) instead of session names.
- `statusCounts` map for per-status counts. `StatusStarting` counted as `StatusRunning` in display.
- Phase impact: None — Copilot sessions automatically counted.

**Transition notifier simplified**
- Removed fallback conductor routing. Now only routes through explicit parent linkage.
- `transitionDeliveryFallbackSent` constant removed.
- Phase impact: Minimal. Our Copilot sessions just need `ParentSessionID` set (same as other tools).

**Conductor changes**
- `SetupConductor()` gained 7th parameter: `clearOnCompact bool` (was 6 params in our last review — now 7: name, profile, heartbeatEnabled, clearOnCompact, description, customClaudeMD, customPolicyMD).
- `ConductorMeta` gained `ClearOnCompact *bool` field with `GetClearOnCompact()` method (default true).
- `Instance.ConductorClearOnCompact()` method added.
- `findAgentDeck()` refactored: now uses `agentDeckPathFromArg0()`, `normalizeExecutablePath()`, `isExecutablePath()`.
- `buildDaemonPath()` refactored for dedup.
- Phase impact: None for Copilot directly but Phase 1 conductor_cmd.go tests must pass with new signature.

**Storage changes**
- `SaveWithGroups()` now calls `UpdateClaudeSessionsWithDedup()` before persisting — enforces dedup at storage layer, not just in-memory.
- `SaveRecentSession()` / `LoadRecentSessions()` added.
- Phase impact: Copilot sessions stored/loaded same as others. No special handling needed.

**Confirm dialog API change**
- `ShowDeleteSession()` signature changed: added 3rd parameter `sandboxed bool`.
- Phase impact: Any code that calls this must pass the sandbox flag.

**`UpdateClaudeSessionsWithDedup` behavior change**
- Now works on a copy (`ordered`) to avoid mutating the input slice order as side effect.
- Uses `sort.SliceStable` instead of `sort.Slice`.

**OpenCode patterns enriched**
- BusyPatterns expanded: added `"esc to exit"`, `"thinking..."`, `"generating..."`, `"building tool call..."`, `"waiting for tool response..."`.
- PromptPatterns expanded: added `"press enter to send"`.
- SpinnerChars added: `"█", "▓", "▒", "░"`.
- Phase impact: Establishes pattern for how tool-specific patterns grow. Our Copilot patterns follow the same structure.

**buildBashExportPrefix() extracted**
- Common bash export logic (AGENTDECK_INSTANCE_ID + optional CLAUDE_CONFIG_DIR) extracted into `buildBashExportPrefix()` method.
- Phase 2 impact: Our Copilot command builder should follow this pattern (extract common prefix logic).

**slog formatting cleanup**
- Throughout instance.go, tmux.go: multi-arg slog calls reformatted from single-line to multi-line for readability. No functional change.

**Merge analysis:**
- Our fork has 30 `.squad/` files deleted upstream + 36 `docs/plans/copilot-cli*` files deleted upstream = **66 files deleted upstream that we need to keep**.
- `git merge upstream/main` will create delete-vs-modify conflicts for our .squad/ and docs/plans/ files.
- Resolution: `git checkout HEAD -- .squad/ docs/plans/copilot-cli/ docs/plans/copilot-cli-captures/ docs/plans/2026-02-17-copilot-cli-support.md` after merge to restore our files.
- No go.mod/go.sum changes — zero dependency conflicts.
- Version bumped from 0.19.14 → 0.19.19.

**Phase plan impact summary:**
- Phase 1 (Config Surface): Update `SetupConductor` call sites (7 params). `focusTarget` enum system for UI pickers. No other blocking changes.
- Phase 2 (Command Builder): Must use `prepareCommand()` pattern (3-return-value). Add `"copilot"` to `toolDetectionOrder` + `toolDetectionPatterns`. Follow `buildBashExportPrefix()` extraction pattern.
- Phase 3 (Session Detection): No new blockers — session filesystem detection unaffected.
- Phase 4 (Status Detection): MUST use raw-ANSI-capture → `StripANSI()` → pattern-match pipeline. Dead-pane detection is free. Pane title detection N/A for Copilot (already documented).
- Phase 5 (Preflight): No impact.
- Phase 6 (Docs/Polish): Config reference format unchanged.

### 2026-03-01 — Phase 1 Config Surface: Detailed Code Review & Work Plan

**Reviewed all 7 target files against the Phase 1 spec.** Verified line numbers, struct locations, tool list arrays, and icon/color function switch cases.

**Spec accuracy findings:**
- Line numbers in spec are slightly off vs current codebase (e.g., `CodexSettings` is at line 508, not "~500"; `Codex` field in `UserConfig` is at line 65, not "~56"). Produced exact line references for Parker.
- Spec misses 3 locations that also need "copilot" added: `GetToolIcon()` in userconfig.go (line 1002, duplicates styles.go logic), `GetCustomToolNames()` builtins map (line 987), and `default_tool` comment strings (lines 25 and 1391).
- `prepareCommand()` 3-return-value and `focusTarget` enum do NOT affect Phase 1. Copilot has no options panel yet (`updateToolOptions` falls to `default: nil`), and command building is Phase 2.
- `ForkDialog` is Claude-only (hardcoded `*ClaudeOptionsPanel`) — no tool picker to update. Not Phase 1.
- `toolDetectionOrder` in tmux.go is detection, not a picker — Phase 2 territory.
- `detectTool()` in main.go needs "copilot" case — Phase 2 territory.
- 6 tool list arrays found across 4 files (newdialog.go, settings_panel.go, setup_wizard.go). 2 test files also have hardcoded tool lists (Lambert's domain).

**Key architectural note:** `GetToolIcon()` in userconfig.go (line 1002) and `ToolIcon()` in styles.go (line 585) are parallel implementations — both have switch cases for the same tools. Comment in styles.go explains circular import prevents delegation. Both must be updated in lockstep.

### 2026-03-01 — Cross-Agent Update: Phase 3 Complete

**Phase 3 (Session Detection + Resume) fully implemented and tested.**
- Parker implemented 6 functions in `copilot.go`: `DetectCopilotSession`, `detectCopilotSessionAsync`, `getCopilotHomeDir`, `queryCopilotSession`, `collectOtherCopilotSessionIDs`, plus `UpdateCopilotSession` filesystem fallback in `instance.go`.
- Async detection wired into `Start()` and `StartWithMessage()`. Clean build/vet.
- Lambert wrote 14 tests covering: query matching, time windows, multi-candidate selection, ID exclusion, unscoped queries, empty dir, corrupt YAML, missing ID, home dir resolution (default/env/config override), and 2 resume integration tests. All green.
- The Phase 0 hard gate (filesystem session ID detection) is now fully resolved in code.

### 2026-03-01 — Cross-Agent Update: Phase 4 Complete

**Phase 4 (Status Detection) fully implemented and tested.**
- Parker added `DefaultRawPatterns("copilot")` (BusyPatterns: "Esc to cancel" + state-icon regex, PromptPatterns: "Type @ to mention files"), tool detection order/patterns, `detectToolFromCommand` copilot case, and `CanFork` copilot case across patterns.go, tmux.go, and instance.go.
- Lambert wrote 5 test functions (22 cases) and caught a false-positive: `\bcopilot\b` in `detectToolFromContent` matches paths containing "copilot". Coordinator resolved by replacing with state-icon regex `(?m)^[◉◐◎∙]\s`. All tmux tests pass.

### 2026-03-01 — Cross-Agent Update: Phase 6 Complete

**Phase 6 (Documentation, TUI Polish, CHANGELOG) fully implemented and tested.**
- Ripley analyzed Phase 6 design doc: 2 tasks already complete (sample config, feature flag), 1 deferred (experimental features), 5 actionable. Produced execution plan at `.squad/plans/phase-6-execution-plan.md`.
- Parker executed all 5 tasks: README multi-tool table + problem section, troubleshooting Copilot section, Copilot session details panel in home.go, CHANGELOG v0.20.0 entry. Also fixed settings_panel.go hardcoded index bug (4 → `len(toolValues)-1`).
- Lambert updated settings_panel_test.go with Copilot entries, verified all tests pass (ui, tmux, cmd packages).
