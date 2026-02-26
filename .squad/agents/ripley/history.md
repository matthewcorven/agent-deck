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
