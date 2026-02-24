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
