# Phase 0 — Live Capture Findings

> **Analyst:** Ripley (Lead)  
> **Date:** 2026-02-25  
> **Captures reviewed:** `/session` TUI modal, log file, `workspace.yaml`, `~/.copilot/` directory structure  
> **Copilot CLI version:** 0.0.417

---

## 1. Session ID Detection Strategy — GATING DELIVERABLE RESOLVED

### Task 3 Answers

**Q1: Is a session ID printed in the welcome banner?**  
No. The welcome banner shows `Welcome matthewcorven!` — no session ID or workspace ID is displayed at startup. The session ID only appears in the `/session` command modal (which is a blocking TUI overlay, not scrapeable from the normal pane content).

**Q2: Can session IDs be read from `~/.copilot/` storage files? What format?**  
Yes. Two distinct ID types exist on the filesystem:

- **Workspace ID** (UUID v4): Found in `~/.copilot/session-state/<workspace-uuid>/workspace.yaml`. Contains `id`, `cwd`, `git_root`, `repository`, `branch`. Example: `8713b307-9b88-407f-a56b-c8ab7ec25b7b`.
- **Session ID** (UUID v4): Found in the `/session` modal output and as a directory name under `~/.copilot/session-state/`. Example: `155f69ab-8c0f-4b4a-a0ae-87ba6f518176`. The session state directory contains `events.jsonl`.

The workspace ID is the process-scoped identifier (created at startup, logged as `Workspace initialized: {id}`). The session ID appears to be the conversational session within that workspace, and its directory also lives under `~/.copilot/session-state/`.

**Q3: Does `--continue` reliably resume the last session without an explicit ID?**  
Not yet confirmed from captures (requires a restart test). However, the CLI's `--help` output (captured separately) should document this flag. The `/session` modal mentions the session directory path — the CLI likely uses this to locate continue state.

**Q4: Is the session ID visible in `/usage` output?**  
Not directly tested. The `/session` modal shows both the session ID and session directory path. `/usage` may show similar data but was not captured in this batch.

**Q5: What is the session ID format?**  
UUID v4 (standard 8-4-4-4-12 hex format): `155f69ab-8c0f-4b4a-a0ae-87ba6f518176`. Same format for workspace IDs.

### Chosen Strategy: **Option A — Filesystem-based detection (primary) + `--continue` fallback**

**Detection algorithm:**

1. At Copilot launch, record `CopilotStartedAt` (unix millis) on the Instance — same pattern as Codex.
2. After a 1–2 second delay, scan `~/.copilot/session-state/*/workspace.yaml`.
3. For each `workspace.yaml`:
   - Parse YAML for `cwd` / `git_root` / `repository` fields.
   - Match against the Instance's `ProjectPath`.
   - Check `created_at` is after `CopilotStartedAt` (for new sessions) OR `updated_at` is recent (for resumed sessions).
4. The `id` from the matching `workspace.yaml` is the **workspace ID**. Store it as `CopilotWorkspaceID`.
5. To find the **session ID**, scan for `events.jsonl` files under `~/.copilot/session-state/` with modification times after `CopilotStartedAt`, correlating with the workspace directory.
6. Store session ID as `CopilotSessionID`; store workspace ID separately — both are useful for different operations.
7. Fallback: if filesystem discovery fails, use `--continue` for resume (Option C).

**Why this strategy:**

- **Mirrors the Codex pattern exactly** — filesystem walk, YAML/JSONL parsing, project path matching, time-window filtering. The codebase already has `queryCodexSession()` doing nearly the same thing with `.jsonl` files.
- **No tmux scraping required** — avoids the fragility of parsing the `/session` TUI modal (which is a blocking overlay, not regular pane content).
- **Two-ID model is robust** — workspace ID anchors to the project directory; session ID anchors to the conversational session. Agent Deck can track both in the Instance struct, using workspace ID for directory matching and session ID for resume/history.
- **`workspace.yaml` is YAML** — trivially parseable (Go `gopkg.in/yaml.v3` or even simple regex for these flat fields). Much more stable than TUI scraping.
- **Filesystem is always available** — no dependency on CLI flags, network state, or TUI rendering.

### Dual-ID Mapping (Important Architectural Note)

The Copilot CLI uses TWO identifiers:

| ID | Source | Purpose | Lifecycle |
|----|--------|---------|-----------|
| Workspace ID | `workspace.yaml` → `id` field | Anchors to a working directory. Created at process start. | Process-scoped (new per `copilot` invocation) |
| Session ID | `/session` modal, `events.jsonl` directory name | Identifies the conversational session | May persist across `--continue` |

Agent Deck should store BOTH: `CopilotWorkspaceID` for filesystem matching and `CopilotSessionID` for resume operations. The workspace ID is the reliable detection anchor (it's in a YAML file with the `cwd` right there). Session ID is needed for `--resume` if it takes an explicit ID.

---

## 2. Log File Pattern Analysis

### Startup Sequence

The log reveals a deterministic startup sequence:

```
1. [ERROR] PAT check                           → t+0ms
2. [INFO]  Workspace initialized: {uuid}        → t+296ms
3. [INFO]  Starting Copilot CLI: {version}       → t+317ms  
4. [INFO]  Node.js version: {version}            → t+317ms
5. [INFO]  Login status unknown                  → t+364ms
6. [INFO]  Remote session access enabled         → t+368ms
7. [INFO]  Welcome {username}!                   → t+376ms
8. [ERROR] MCP client startups (parallel)        → t+396ms–876ms
9. [ERROR] IDE MCP server connect               → t+1033ms
10. [INFO] Using default model: {model}          → t+1044ms–1156ms (logged 3x)
11. [INFO] Memory enablement check: disabled     → t+1411ms
```

**Total startup: ~1.4 seconds** from first log line to ready state.

### MCP Client Events — NOT Actual Errors

The `[ERROR]` log level on MCP events is **misleading — these are verbose/debug messages**, not actual errors. Evidence:

- Every MCP client logs `[ERROR] Starting MCP client for {name}...` followed by `[ERROR] MCP client for {name} connected, took Xms` and `[ERROR] Started MCP client for {name}` — this is a successful lifecycle.
- The pattern applies uniformly to stdio servers (aspire), npx-launched servers (playwright), and remote HTTP servers (microsoft-learn, github-mcp-server).
- The IDE MCP server connection also logs at `[ERROR]` level despite succeeding.

**Implication for Phase 4:** Do NOT treat `[ERROR]` lines in the log as error indicators for status detection. The log level is unreliable. Instead, match on message content.

### Key Log Patterns for Status Detection

| Pattern | Signal | Use |
|---------|--------|-----|
| `Workspace initialized: {uuid}` | Startup, workspace ID | Session detection (INFO level) |
| `Starting Copilot CLI: {version}` | Startup, version | Version pinning (INFO level) |
| `Welcome {username}!` | Auth complete, startup done | Ready state indicator (INFO level) |
| `MCP client for {name} connected, took {N}ms` | MCP ready | MCP pool status (ERROR level, actually informational) |
| `Using default model: {model}` | Model selected | Analytics (INFO level, logged 3x during startup — dedup needed) |
| `Connected to IDE MCP server: {name}` | IDE link established | IDE integration status (ERROR level, actually informational) |

### MCP Server Types Observed

| Server | Type | Transport | Startup Time |
|--------|------|-----------|-------------|
| aspire | Local stdio | `aspire mcp start` | 23ms |
| playwright | Local via npx | `npx -y @playwright/mcp@latest` | 627ms |
| microsoft-learn | Remote HTTP | `https://learn.microsoft.com/api/mcp` | 310ms |
| github-mcp-server | Remote HTTP | `https://api.individual.githubcopilot.com/mcp/readonly` | 128ms |
| VS Code Insiders | IDE stdin/stdout | Socket to running IDE | ~7ms |

---

## 3. Surprises and Concerns

### 3a. Workspace UUID ≠ Session UUID

The log shows `Workspace initialized: 8713b307-...` while the `/session` modal shows `ID: 155f69ab-...`. These are different UUIDs for different concepts. The `workspace.yaml` uses the workspace UUID as its `id` field. The session UUID is the conversational session identifier. Agent Deck must track both — see Dual-ID Mapping above.

### 3b. `/session` Modal is a Blocking TUI Overlay

The `/session` command renders a full-screen modal with a live clock and requires `Enter` to dismiss. This means:

- **Cannot scrape session ID from normal `tmux capture-pane`** during the modal — it replaces pane content.
- **Cannot use `/session` programmatically** — it blocks the CLI's input loop.
- **Not viable for automated session ID extraction** — confirms that filesystem-based detection (Option A) is the right call.

### 3c. MCP Transport Diversity

The Copilot CLI connects to MCP servers via three distinct transport mechanisms:

1. **Stdio** (aspire) — local process, `command` + `args`
2. **npx-launched stdio** (playwright) — spawns via npx, then stdio
3. **Remote HTTP/SSE** (microsoft-learn, github-mcp-server) — server-sent events over HTTPS
4. **IDE socket** (VS Code Insiders) — connects to the running IDE's MCP pipe

**Implication for Agent Deck's MCP pooling:** The `--additional-mcp-config` flag can inject Agent Deck's pool MCP servers into the Copilot session. However, Agent Deck must NOT interfere with the built-in GitHub MCP server (which appears to be auto-configured after authentication — `GitHub MCP server configured after authentication`). Phase 5 (Preflight) should account for this.

### 3d. Model Logged 3 Times

`Using default model: claude-sonnet-4.6` appears three times in the log during startup. This suggests the model is resolved at multiple stages (initial, post-auth, post-MCP). For analytics (model tracking in the Instance struct), deduplicate by taking the last occurrence.

### 3e. Memory System Exists But Disabled

`Memory enablement check: disabled` — Copilot CLI has a memory subsystem (disabled for this user). If enabled, it may create additional files under `~/.copilot/` that could serve as a secondary signal for session detection. Worth noting for future phases.

### 3f. `cwd: undefined` in MCP Config

Several MCP clients log `cwd: undefined` at startup. This is the MCP config's working directory, not the session's. It means the MCP servers are spawned without a specific working directory. Not a concern for Agent Deck — our MCP pool servers have explicit working directories.

---

## 4. Preliminary Observations for Pattern Drafting (Phase 4)

Based on the `/session` modal and log observations:

### Candidate Prompt Patterns

The `/session` modal shows the input prompt as just `Enter to continue` when in the modal view. The actual input prompt for the conversational interface was not captured in this batch — **Task 2b captures are still needed** for idle/busy/approval states.

### Candidate Busy Patterns

Not yet captured — requires Task 2b tmux captures during active tool use.

### What We Know Will Work

From the log timing, Copilot reaches "ready" state in ~1.4 seconds. The `Welcome {username}!` message in the log (and likely in the TUI) marks the transition from startup to ready. This is relevant for `StatusStarting` → `StatusIdle` transition timing.

### Next Captures Needed

To draft `DefaultRawPatterns("copilot")`, we still need:
- **Idle state** — what the prompt looks like when waiting for input
- **Busy state** — spinner/progress text during tool execution
- **Tool approval** — the permission prompt format
- **Error state** — what shows when something fails
- **Pane title** — `tmux display-message -p '#{pane_title}'` while active

---

## 5. TUI State Capture Analysis (Task 2b — Complete)

> **Captures analyzed:** Startup, Idle, Thinking, Tool, Plan, Error, MCP, PaneTitle, PaneTitle-renamed  
> **Each state captured in dual mode:** plain text (`.txt`) + ANSI escape sequences (`-ansi.txt`)

### 5a. State Catalog

| State | Capture File | Key Visual Indicators | Status |
|-------|-------------|----------------------|--------|
| **Startup** | `Startup.txt` | Welcome banner, `◉ Loading environment:`, version, tip line | Loading → Idle transition |
| **Idle (normal)** | `Idle.txt` | `❯  Type @ to mention files`, `shift+tab switch mode` | Awaiting input |
| **Thinking** | `Thinking.txt` | `◉ Thinking (Esc to cancel)`, `ctrl+q enqueue` in status bar | Active processing |
| **Tool execution** | `Tool.txt` | `◐` / `◎` state icons, tool call results with `●` prefix, `Esc to cancel` | Active processing |
| **Plan mode** | `Plan.txt` | `Describe a plan.` prompt, `plan ·` mode indicator, `∙ Thinking (Esc to cancel · 183 B)` | Plan mode variant |
| **Error** | `Error.txt` | `✗ Execution failed:` (bold red in ANSI), error message follows | Error state, returns to idle |
| **MCP loading** | `MCP.txt` | `● Environment loaded: 3 MCP servers, 1 plugin, 6 skills, 1 agent` | Startup complete |
| **Pane title** | `PaneTitle.txt` | No custom tmux pane title detected; standard TUI | Normal |
| **Pane title (renamed)** | `PaneTitle-renamed.txt` | `● Session renamed to: boo` displayed in TUI | After `/session rename` |

### 5b. Spinner / State Icon Analysis

Copilot CLI uses **state icons** (not cycling spinners like Claude's braille characters). Each icon indicates a specific processing phase:

| Icon | Meaning | ANSI Color | Found In |
|------|---------|------------|----------|
| `◉` | Loading / initial thinking | Purple/magenta | Startup.txt, Thinking.txt |
| `◐` | Mid-processing / streaming | — (with italic text) | Tool.txt, Plan.txt |
| `◎` | Active tool execution | — | Tool.txt |
| `∙` | Thinking variant | — | Plan.txt |
| `●` | Completed item | Green (tool calls), Blue (info), Purple (response) | Tool.txt, MCP.txt, Idle.txt |
| `✗` | Error / failure | Bold red | Error.txt |
| `❯` | Prompt indicator | — | All idle states |

**Key distinction from Claude:** These are **state indicators**, not cycling animation characters. They don't rotate — each icon maps to a specific processing phase. For status detection, the icons `◉`, `◐`, `◎`, `∙` at line start reliably indicate active processing.

**`●` is NOT a busy indicator.** It marks completed items (finished tool calls, loaded environment, info messages). Must be excluded from busy detection patterns.

### 5c. Busy State Patterns (Copilot is Working)

Three reliable signals that Copilot is actively processing:

| Pattern | Reliability | Rationale |
|---------|-------------|-----------|
| `Esc to cancel` | ⭐⭐⭐ HIGH | Appears in ALL busy states: thinking, tool execution, plan-mode thinking. Universal interrupt hint. Consistent across all captures. |
| `re:(?m)^[◉◐◎∙]\s` | ⭐⭐ MEDIUM | State icon at line start followed by whitespace. Catches processing lines directly. May be fragile if icons change across versions. |
| `ctrl+q enqueue` | ⭐⭐ MEDIUM | Status bar hint only present during active processing. Disappears when idle. Could change if keyboard shortcuts are remapped. |

**Rejected busy candidates:**
- `● ` (completed item prefix) — false positive; appears in idle state for finished items.
- `◉ Loading environment` — too specific; only appears at startup, not during normal processing. The regex `^[◉◐◎∙]\s` already covers `◉` at line start.

### 5d. Idle / Prompt Patterns (Copilot is Ready for Input)

| Pattern | Reliability | Rationale |
|---------|-------------|-----------|
| `Type @ to mention files` | ⭐⭐⭐ HIGH | Present in BOTH normal mode and plan mode idle states. Part of the input hint line. Very stable — it's a core UX element unlikely to change. |
| `Describe a task to get started` | ⭐⭐ MEDIUM | Welcome banner only — appears once at first launch. Good for detecting the initial state before any interaction. |

**Rejected prompt candidates:**
- `Describe a plan` — redundant; `Type @ to mention files` already covers plan mode idle.
- `shift+tab switch mode` — status bar text, appears in all states (busy too?), testing needed.
- `❯` — the prompt character itself; appears always, not a useful idle-vs-busy discriminator.

### 5e. Error Pattern Observations

| Pattern | Meaning |
|---------|---------|
| `✗ Execution failed:` | Fatal execution error (bold red in ANSI). Followed by error details. |
| `● Asked user:` | User clarification prompt (not an error, but a branching interaction). |

The current `RawPatterns` struct has no `ErrorPatterns` field. Error detection is a Phase 4 concern — the TUI returns to idle state after errors, so the prompt pattern still catches the transition back.

### 5f. Plan Mode Detection

Plan mode is visually distinct:
- Prompt changes to: `Describe a plan. Type @ to mention files, / for commands, or ? for shortcuts`
- Status bar shows: `plan · shift+tab switch mode`
- Thinking shows: `∙ Thinking (Esc to cancel · 183 B)` (uses `∙` instead of `◉`)

Since `Type @ to mention files` appears in BOTH normal and plan mode, a single prompt pattern covers both. If plan-mode-specific detection is later needed, `Describe a plan` can be added.

### 5g. Pane Title Behavior

**Copilot CLI does NOT set a custom tmux pane title.** The `PaneTitle.txt` capture shows standard TUI output with no `\033]2;...\007` title escape sequences in the ANSI capture. After `/session rename boo`, the rename confirmation appears in-TUI (`● Session renamed to: boo`) but still no pane title change.

**Implication:** Pane title cannot be used as an alternative detection signal for Copilot. Status detection must rely entirely on pane content scraping.

---

## 6. Drafted `DefaultRawPatterns("copilot")` (Task 4)

Based on complete capture analysis, here is the recommended pattern block for `internal/tmux/patterns.go`:

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

### Design Rationale

**Why no `SpinnerChars` or `WhimsicalWords`:** Copilot's state icons (`◉`, `◐`, `◎`, `∙`) are static per-state indicators, not cycling animation characters like Claude's braille spinners. They don't benefit from the `ThinkingPattern` / `SpinnerActivePattern` combo machinery. The regex `^[◉◐◎∙]\s` handles them directly as a busy pattern.

**Why `Esc to cancel` (capital E):** The captures consistently render `Esc` with a capital E. This differs from Gemini's lowercase `esc to cancel`. Since pattern matching uses `strings.Contains` (case-sensitive), we match the actual rendering. If case changes across versions, this can be upgraded to `re:(?i)esc to cancel`.

**Why not include error patterns:** The `RawPatterns` struct has no `ErrorPatterns` field. The `✗ Execution failed:` marker is valuable but would need a struct extension. After errors, the TUI returns to idle state, so the prompt pattern catches the transition. Error-specific detection can be deferred to Phase 4 if a `StatusError` state needs distinct handling.

**Similarity to Codex pattern:** The structure mirrors the `case "codex"` block — string-based busy patterns + string-based prompt patterns, no spinners. This is the right level of complexity for a non-Claude tool.

### Confidence Assessment

| Pattern | Confidence | Risk |
|---------|-----------|------|
| `Esc to cancel` | HIGH | Core UX element — removal would break the user's cancel affordance |
| `re:(?m)^[◉◐◎∙]\s` | MEDIUM | Unicode chars could change across versions; regex handles the current set |
| `ctrl+q enqueue` | LOW-MEDIUM | Keyboard shortcut could be remapped or removed |
| `Type @ to mention files` | HIGH | Core input affordance — fundamental to the TUI's usability |
| `Describe a task to get started` | MEDIUM | Welcome banner text could change; only seen once per session |

---

## 7. Summary

**All Phase 0 captures are complete.** Task 2b (TUI state captures) produced 9 state pairs (18 files total). Task 3 (session ID detection) was resolved in the previous session. Task 4 (preliminary `DefaultRawPatterns`) is drafted above.

**Key findings:**
1. **`Esc to cancel`** is the single most reliable busy indicator — appears in every active processing state.
2. **`Type @ to mention files`** is the single most reliable idle indicator — appears in both normal and plan mode.
3. Copilot uses **state icons** (`◉◐◎∙`), not cycling spinners. They serve as a secondary busy signal via regex.
4. **No custom pane title** is set by Copilot CLI — detection must rely on content scraping only.
5. **Error state** (`✗`) is observable but not captured by the current pattern struct; deferred to Phase 4.
6. **Plan mode** is covered by the same prompt pattern (`Type @ to mention files`); no special handling needed.

**Phase 0 is effectively complete.** The exit criteria are met: captures committed, session ID strategy resolved, preliminary patterns drafted. Phase 1 (config surface) and Phase 2 (command builder) are unblocked.
