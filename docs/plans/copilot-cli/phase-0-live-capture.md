# Phase 0 — Live Capture

> **Goal:** Capture terminal output from a real Copilot CLI session in tmux to finalize status detection patterns and session ID strategy.  
> **Depends on:** Nothing (BLOCKING prerequisite for all other phases).  
> **Estimated scope:** Manual exploration + commit captures. No code changes.

## Why This Phase Exists

Every other tool (Claude, Gemini, OpenCode, Codex) was integrated by first observing what the CLI actually renders in a tmux pane. This phase eliminates the single unknown: what text patterns to match against.

> **⚠️ HARD BLOCKER — Session ID Detection**
>
> Session ID extraction is the **#1 open risk** for Copilot CLI integration. If Phase 0 cannot determine a reliable method to detect or extract session IDs from the Copilot CLI, **Phase 3 (Session Detection + Resume) is blocked entirely.** Phases 1 and 2 may proceed (config surface and command builder don't require session IDs), but any functionality depending on session identity — resume, restart, session history — cannot be built until this is resolved.
>
> Task 3 below is the **gating deliverable** for this question. Treat it with the highest priority during live capture.

## Tasks

### 1. ✅ Install & authenticate

```bash
brew install copilot-cli@prerelease  # or: npm install -g @github/copilot
copilot                     # launches interactive TUI; follow /login flow
```

### 2a. ✅ Capture CLI metadata

These are one-time captures (not state-dependent) that feed directly into later phases.

```bash
# Source of truth for all CLI flags — feeds command builder (Phase 2).
# Confirm --model, --agent, --yolo, --resume, --continue, --config-dir,
# --additional-mcp-config, -i all exist; discover any new flags.
copilot --help > captures/cli-help.txt

# Pin the version we validated against. The CLI is in public preview;
# flags/patterns may change across releases. Lets us detect regressions.
copilot --version > captures/cli-version.txt

# Reveals session state file locations & config-dir defaults.
# Directly informs session ID detection strategy (Task 3) — if session
# state lives on the filesystem we can read IDs programmatically
# instead of parsing the TUI.
ls -laR ~/.copilot/ > captures/copilot-dir-structure.txt 2>&1
```

### 2b. ✅ Capture terminal content in each state

Launch `copilot` inside a tmux pane, then capture output at each stage.

> **Completed 2026-02-25.** All 9 states captured in dual mode (plain text + ANSI). Files committed to `docs/plans/copilot-cli-captures/`. See `findings.md` §5 for full analysis.

States captured: Startup, Idle, Thinking, Tool, Plan, Error, MCP, PaneTitle, PaneTitle-renamed (× 2 modes = 18 files).

### 3. ✅ Determine session ID detection strategy (**GATING DELIVERABLE — RESOLVED**)

> This task is a **hard gate**. Its outcome determines whether Phase 3 (Session Detection + Resume) can proceed. If no reliable session ID extraction method is found, document that finding explicitly and record the fallback strategy.

Answer these questions from the live session:

- Is a session ID printed in the **welcome banner**? **→ No.** Welcome banner shows `Welcome {username}!` only.
- Can session IDs be read from `~/.copilot/` storage files? What format? **→ Yes.** `~/.copilot/session-state/{workspace-uuid}/workspace.yaml` contains workspace ID, `cwd`, `git_root`, `repository`, `branch`. Session ID found as directory name under `session-state/` with `events.jsonl`. Both are UUID v4.
- Does `--continue` reliably resume the last session without an explicit ID? **→ Not yet confirmed** (requires restart test). Serves as fallback.
- Is the session ID visible in `/usage` output? **→ Not tested.** The `/session` modal shows it, but that modal is a blocking TUI overlay not suitable for scraping.
- What is the session ID format? (UUID, hash, incremental?) **→ UUID v4** (standard 8-4-4-4-12 hex).

**Strategy chosen: Option A (filesystem-based) + Option C (`--continue`) fallback.** See `docs/plans/copilot-cli-captures/findings.md` for full rationale. Decision recorded in `.squad/decisions/inbox/ripley-phase0-session-id-strategy.md`.

**Key architectural note:** Copilot uses TWO distinct UUIDs — a workspace ID (process-scoped, in `workspace.yaml`) and a session ID (conversational, in `events.jsonl` directory). Agent Deck will track both in the Instance struct, following the dual-ID pattern.

**Fallback strategy if no session ID is discoverable:**

1. **Synthetic IDs** — Generate an Agent Deck–managed session ID (UUID) at launch time. Store the mapping in SQLite. Resume/restart would rely on Agent Deck's own tracking rather than the CLI's native session concept.
2. **`--continue` only** — If the CLI supports `--continue` (resume last session) but exposes no explicit session ID, Agent Deck can use that flag without needing to know the ID. Multi-session resume would not be possible.
3. **Defer Phase 3** — If neither option is viable, Phase 3 is deferred until ACP (Phase 7+) provides programmatic session access. Document this as a known limitation.

### 4. ✅ Draft preliminary patterns

> **Completed 2026-02-25.** Full analysis in `findings.md` §5–§6. Pattern confidence assessed per-signal.

Based on all captures, the drafted `DefaultRawPatterns("copilot")` block:

```go
case "copilot":
    return &RawPatterns{
        BusyPatterns: []string{
            "Esc to cancel",       // PRIMARY: universal busy indicator across all active states
            `re:(?m)^[◉◐◎∙]\s`,   // SECONDARY: state icon at line start during processing
            "ctrl+q enqueue",      // TERTIARY: status bar hint, only present during active processing
        },
        PromptPatterns: []string{
            "Type @ to mention files", // PRIMARY: stable prompt hint in idle state (normal + plan mode)
            "Describe a task to get started", // SECONDARY: welcome banner initial state
        },
    }
```

**Key design choices:** No `SpinnerChars` — Copilot uses state icons, not cycling spinners. No `ErrorPatterns` — struct doesn't have that field; errors return to idle. Pattern mirrors the Codex block structure (strings + one regex, no whimsical words).

### 5. Commit findings

Create `docs/plans/copilot-cli-captures/` with:
- `cli-help.txt` — full `copilot --help` output (CLI flags source of truth)
- `cli-version.txt` — `copilot --version` output (pinned validation baseline)
- `copilot-dir-structure.txt` — `ls -laR ~/.copilot/` output (session storage discovery)
- One `.txt` file per captured state (e.g., `startup.txt`, `idle.txt`, `busy.txt`, `approval.txt`)
- Corresponding `-ansi.txt` files for each state (escape-sequence captures)
- A `findings.md` summarizing:
  - Session ID detection strategy chosen (Option A/B/C from design doc)
  - Preliminary busy patterns
  - Preliminary prompt patterns
  - Any surprises or concerns

## Exit Criteria

- [x] CLI metadata captured and committed (`cli-help.txt`, `cli-version.txt`, `copilot-dir-structure.txt`)
- [x] Captured terminal content files committed to `docs/plans/copilot-cli-captures/` (plain text + ANSI pairs) — 9 states × 2 modes = 18 files
- [x] **HARD GATE:** Session ID detection strategy documented in `findings.md` — filesystem-based via `workspace.yaml` (Option A) with `--continue` fallback (Option C). Phase 3 is unblocked.
- [x] Preliminary `DefaultRawPatterns("copilot")` drafted — see `findings.md` §6. Includes 3 busy patterns, 2 prompt patterns, confidence assessment per signal.
- [x] New observations documented: no pane title support, state icons (not cycling spinners), `✗` error marker, plan mode covered by shared prompt pattern. No undocumented CLI flags found.

## Notes

- This phase is entirely manual — no code changes, no tests.
- If the Copilot CLI is unavailable or auth fails, document the blockers and defer.
- Pay attention to locale/theme variations in output — patterns should be language-agnostic if possible.
