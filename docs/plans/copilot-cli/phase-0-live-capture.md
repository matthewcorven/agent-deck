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

### 1. Install & authenticate

```bash
brew install copilot-cli@prerelease  # or: npm install -g @github/copilot
copilot                     # launches interactive TUI; follow /login flow
```

### 2a. Capture CLI metadata

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

### 2b. Capture terminal content in each state

Launch `copilot` inside a tmux pane, then capture output at each stage.

**Dual-mode capture:** For every state, capture BOTH plain text and ANSI escape sequences. Plain text is what patterns match against; the escape-sequence version helps distinguish real text from TUI rendering artifacts (spinners, cursor positioning, color codes).

```bash
# Plain text (pattern matching)
tmux capture-pane -p -t <pane> > captures/<state>.txt
# With ANSI escapes (artifact analysis)
tmux capture-pane -p -e -t <pane> > captures/<state>-ansi.txt
```

States to capture:

| State | What to look for |
|-------|-----------------|
| **Startup / welcome banner** | Any session ID displayed, version info, greeting |
| **Idle / awaiting input** | The prompt line (e.g., `>`, `copilot>`, or freeform input area) |
| **Thinking / busy** | Spinner text, "esc to interrupt", tool execution progress |
| **Tool approval prompt** | "1. Yes / 2. Yes, and approve TOOL... / 3. No..." |
| **Plan mode prompt** | After Shift+Tab – different prompt indicator? |
| **`/compact` auto-compaction** | Any visible progress text during compaction |
| **Error states** | Network failure, auth expiry, permission denied |
| **MCP server output** | Look for lines like "Connected to GitHub MCP server" or custom MCP config announcements. The Copilot CLI ships with a built-in GitHub MCP server — if it announces in the TUI, this affects status detection patterns and MCP pooling integration in later phases. |
| **Pane title** | Run `tmux display-message -p '#{pane_title}'` while Copilot is active. Some CLIs set the terminal/pane title dynamically — if Copilot does this, it could serve as an alternative detection signal alongside content scraping (potentially more reliable for state detection). |

### 3. Determine session ID detection strategy (**GATING DELIVERABLE**)

> This task is a **hard gate**. Its outcome determines whether Phase 3 (Session Detection + Resume) can proceed. If no reliable session ID extraction method is found, document that finding explicitly and record the fallback strategy.

Answer these questions from the live session:

- Is a session ID printed in the **welcome banner**?
- Can session IDs be read from `~/.copilot/` storage files? What format?
- Does `--continue` reliably resume the last session without an explicit ID?
- Is the session ID visible in `/usage` output?
- What is the session ID format? (UUID, hash, incremental?)

**Fallback strategy if no session ID is discoverable:**

1. **Synthetic IDs** — Generate an Agent Deck–managed session ID (UUID) at launch time. Store the mapping in SQLite. Resume/restart would rely on Agent Deck's own tracking rather than the CLI's native session concept.
2. **`--continue` only** — If the CLI supports `--continue` (resume last session) but exposes no explicit session ID, Agent Deck can use that flag without needing to know the ID. Multi-session resume would not be possible.
3. **Defer Phase 3** — If neither option is viable, Phase 3 is deferred until ACP (Phase 7+) provides programmatic session access. Document this as a known limitation.

### 4. Draft preliminary patterns

Based on captures, draft the `DefaultRawPatterns("copilot")` block:

```go
case "copilot":
    return &RawPatterns{
        BusyPatterns:   []string{/* fill from captures */},
        PromptPatterns: []string{/* fill from captures */},
    }
```

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

- [ ] CLI metadata captured and committed (`cli-help.txt`, `cli-version.txt`, `copilot-dir-structure.txt`)
- [ ] Captured terminal content files committed to `docs/plans/copilot-cli-captures/` (plain text + ANSI pairs)
- [ ] **HARD GATE:** Session ID detection strategy documented in `findings.md` — must explicitly state whether a reliable extraction method exists. If not, the chosen fallback strategy must be recorded. Phase 3 is blocked until this is resolved.
- [ ] Preliminary `DefaultRawPatterns("copilot")` drafted (even if approximate)
- [ ] Any new CLI flags or behaviors not in the design doc are noted

## Notes

- This phase is entirely manual — no code changes, no tests.
- If the Copilot CLI is unavailable or auth fails, document the blockers and defer.
- Pay attention to locale/theme variations in output — patterns should be language-agnostic if possible.
