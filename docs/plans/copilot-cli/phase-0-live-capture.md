# Phase 0 — Live Capture

> **Goal:** Capture terminal output from a real Copilot CLI session in tmux to finalize status detection patterns and session ID strategy.  
> **Depends on:** Nothing (BLOCKING prerequisite for all other phases).  
> **Estimated scope:** Manual exploration + commit captures. No code changes.

## Why This Phase Exists

Every other tool (Claude, Gemini, OpenCode, Codex) was integrated by first observing what the CLI actually renders in a tmux pane. This phase eliminates the single unknown: what text patterns to match against.

## Tasks

### 1. Install & authenticate

```bash
brew install copilot-cli@prerelease  # or: npm install -g @github/copilot
copilot                     # launches interactive TUI; follow /login flow
```

### 2. Capture terminal content in each state

Launch `copilot` inside a tmux pane, then capture output at each stage:

```bash
tmux capture-pane -p -t <pane> > captures/<state>.txt
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

### 3. Determine session ID detection strategy

Answer these questions from the live session:

- Is a session ID printed in the **welcome banner**?
- Can session IDs be read from `~/.copilot/` storage files? What format?
- Does `--continue` reliably resume the last session without an explicit ID?
- Is the session ID visible in `/usage` output?
- What is the session ID format? (UUID, hash, incremental?)

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
- One `.txt` file per captured state (e.g., `startup.txt`, `idle.txt`, `busy.txt`, `approval.txt`)
- A `findings.md` summarizing:
  - Session ID detection strategy chosen (Option A/B/C from design doc)
  - Preliminary busy patterns
  - Preliminary prompt patterns
  - Any surprises or concerns

## Exit Criteria

- [ ] Captured terminal content files committed to `docs/plans/copilot-cli-captures/`
- [ ] Session ID detection strategy documented in `findings.md`
- [ ] Preliminary `DefaultRawPatterns("copilot")` drafted (even if approximate)
- [ ] Any new CLI flags or behaviors not in the design doc are noted

## Notes

- This phase is entirely manual — no code changes, no tests.
- If the Copilot CLI is unavailable or auth fails, document the blockers and defer.
- Pay attention to locale/theme variations in output — patterns should be language-agnostic if possible.
