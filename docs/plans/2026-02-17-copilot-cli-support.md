# GitHub Copilot CLI Support — Design Document

> Generated: 2026-02-17  
> Brainstorm perspectives: Architect, Implementer, Devil's Advocate, Release/Support  
> Chosen approach: First-class Copilot tool (built-in parity with Claude/Gemini/OpenCode/Codex)

> **Implementation guide:** The phased implementation plan has been broken out into self-contained docs for agent-friendly execution. See [copilot-cli/README.md](copilot-cli/README.md) for the index and individual phase instructions.

## Summary

Add GitHub Copilot CLI (`copilot`) as a first-class agent-deck tool with the same UX/features as Claude, Gemini, OpenCode, and Codex: selectable in New Session/Setup Wizard, persisted session tracking/resume, status detection, configuration, and guardrails. The Copilot CLI is a standalone binary (`github/copilot-cli`, public preview) installed via `brew install copilot-cli` or `npm install -g @github/copilot`; the integration must handle install preflights and map Copilot's interactive TUI into tmux-managed sessions.

## Acceptance Criteria

- [ ] Copilot appears as a selectable tool (icon + description) in New Session, Fork Session, Setup Wizard, and settings with default_tool support.
- [ ] `[copilot]` config section added with defaults (command `copilot`, config dir, env file, model/mode/agent flags, yolo mode) and documented in sample `config.toml`.
- [ ] Session start builds the correct command for chat mode, supports per-session overrides, and sets tmux env vars needed for resume/restart.
- [ ] Resume flow works: agent-deck can reattach to an existing Copilot conversation after restart or crash using `--resume SESSION-ID` or `--continue`; resume flag/args are passed when available.
- [ ] Status detection matches other tools: prompt/busy/spinner patterns captured; transitions drive UI state and tests in `tmux`/`patterns` packages.
- [ ] Preflight errors are user-friendly: missing `copilot` binary surfaces actionable toasts/tooltips with install instructions.
- [ ] Storage/state additions (session ID + detected timestamps) are persisted and shown in session details alongside other tools.
- [ ] Unit tests cover command construction, resume argument logic, status pattern detection, and config parsing; manual smoke steps documented.

## Non-Goals

- Building a Copilot API proxy or bypassing the CLI's built-in auth flow.  
- Extending Copilot features beyond what the official CLI exposes.  
- Managing Copilot's MCP configuration (users configure via `~/.copilot/mcp-config.json` or `/mcp add`).  
- Adding new MCPs or altering existing MCP pooling behavior.

## Context & Constraints

- Copilot CLI is a standalone binary (public preview); users may not have it installed. Auth is handled by the CLI itself via `/login` or `GH_TOKEN`/`GITHUB_TOKEN` env vars.  
- Sessions are resumable via `--resume SESSION-ID` and `--continue`; session storage is built into the CLI.  
- The CLI ships with a built-in GitHub MCP server and supports custom MCP servers via `--additional-mcp-config` or `~/.copilot/mcp-config.json`.  
- Output is a rich TUI; status must be inferred from terminal content similar to OpenCode/Codex.  
- Copilot may stream partial tokens; busy/prompt detection must avoid flicker and false-ready states.

## Approaches Considered

1) **First-class built-in tool (Selected):** Mirror the pattern used for Codex/OpenCode: dedicated config struct, command builder, status patterns, persisted session metadata, UI wiring.  
2) **Custom tool template only:** Rely on user-defined `[tools.copilot]` TOML entry. Rejected because it lacks resume/state, status integration, and install guidance.  
3) **ACP protocol integration:** Use `copilot --acp --stdio` for structured programmatic control instead of tmux scraping. Deferred — would be a different architecture than all other tools; better suited as a future enhancement.

## Design

### Tool surface & configuration
- Add `CopilotSettings` in `userconfig.go`: `command` (default `copilot`), `default_model` (maps to `--model`), `default_agent` (maps to `--agent`), `config_dir` (maps to `--config-dir`, default `~/.copilot`), `yolo_mode` (maps to `--yolo`/`--allow-all`), `env_file`, inline `env`.  
- Extend sample `config.toml` with `[copilot]` and `tools.copilot` blocks, including busy/prompt pattern overrides.  
- Add `IconCopilot` and description strings for menus, Setup Wizard, and settings; update default tool mapping.

### Command building & session lifecycle
- New `CopilotOptions` struct + JSON (similar to `CodexOptions`) for per-session overrides (model, agent, config dir, optional context file).  
- `Instance.buildCopilotCommand` composes: base command + `--model` + `--agent` + `--yolo`/`--allow-all` + `--resume SESSION-ID` or `--continue` when session ID known + `--additional-mcp-config` if needed + working directory env.  
- Introduce `CopilotSessionID` fields in `Instance`, `storage`, and migrations; track detection timestamp for UI.  
- Start flow: after tmux pane spawn, store session metadata; set tmux env (e.g., `COPILOT_SESSION_ID`) for resume tracking. Use `-i PROMPT` flag to auto-execute an initial prompt when provided.  
- Restart flow: if session ID exists, respawn with `--resume SESSION-ID`; if unknown, use `--continue` to resume most recent session; otherwise start fresh.

### Status detection
- Add default patterns to `tmux` detector: busy candidates ("esc to interrupt", thinking spinner, tool execution text), prompt markers (input prompt line, approval prompts like "1. Yes / 2. Yes, and approve..."), plan mode indicator.  
- **Action needed:** Capture actual terminal content from a live Copilot CLI session in tmux during thinking/idle/approval states before finalizing patterns.  
- Mirror Codex/OpenCode tests in `internal/tmux/patterns_test.go` to cover Copilot patterns and avoid collisions with generic `>` prompts.

### Preflight & validation
- Add install check in session start: `copilot version` or `which copilot` to confirm the binary is in PATH.  
- Auth is handled by the CLI itself (`/login` on first launch, or `GH_TOKEN`/`GITHUB_TOKEN` env vars) — no external auth check needed.  
- UI toasts for missing binary; link to install instructions (`brew install copilot-cli` / `npm install -g @github/copilot`).

### UI wiring
- New session dialog: Copilot card with icon and description; model/agent overrides shown when tool selected.  
- Settings panel: `[copilot]` toggles/fields; default tool radio includes Copilot.  
- Session details view: show Copilot session ID + detected-at timestamps similar to Codex.  
- New session picker/state search includes Copilot tool filtering.

### Testing & rollout
- Unit tests: config parsing defaults, command builder (with/without resume), status pattern detection, state persistence migration.  
- Manual smoke script: install `copilot` binary, authenticate via `/login`, start Copilot session from agent-deck, verify busy→prompt transitions, restart app, confirm resume works.  
- Feature flag via config (`[copilot].enabled` or UI toggle) for initial rollout if needed.

## Risks & Open Questions

- Copilot CLI is in **public preview** — API surface, flags, and TUI output may change between releases.  
- The `--resume` session ID format is not yet documented; need to capture from a live session to confirm tracking approach.  
- Output patterns could be localized or change across versions; busy/prompt patterns should be configurable in `config.toml`.  
- Session resume (`--resume`/`--continue`) behavior on version upgrades is untested.

## Phased Implementation Plan

### Phase 0 — Live capture (BLOCKING prerequisite)

**Goal:** Eliminate the single unknown — what the Copilot CLI actually renders in tmux — and determine the session ID detection strategy before writing any code.

**Tasks:**
1. Install Copilot CLI (`brew install copilot-cli`), authenticate via `/login`.
2. Launch `copilot` inside a tmux pane; capture terminal content (`tmux capture-pane -p`) during each of these states:
   - Startup / welcome banner (note any session ID displayed)
   - Idle / awaiting input (the prompt line)
   - Thinking / busy (spinner, "esc to interrupt", etc.)
   - Tool approval prompt ("1. Yes / 2. Yes, and approve TOOL... / 3. No...")
   - Plan mode prompt (after Shift+Tab)
   - `/compact` auto-compaction in progress
   - Error states (network failure, auth expiry)
3. Determine how session IDs are surfaced:
   - Is a session ID printed in the welcome banner or `/usage` output?
   - Can session IDs be read from `~/.copilot/` storage files?
   - Does `--continue` reliably resume without an explicit ID?
4. Commit text captures and findings to `docs/plans/copilot-cli-captures/`.

**Exit criteria:** Captured terminal content committed; session ID detection strategy documented; `DefaultRawPatterns("copilot")` drafted (even if preliminary).

---

### Phase 1 — Config surface

**Goal:** Copilot appears as a selectable tool everywhere in the UI/config, with no functional backend yet.

**Tasks:**
1. Add `CopilotSettings` struct to `internal/session/userconfig.go`:
   - `command` (default `"copilot"`)
   - `yolo_mode` (bool → `--yolo` / `--allow-all`)
   - `default_model` (string → `--model`)
   - `default_agent` (string → `--agent`)
   - `config_dir` (string → `--config-dir`, default `~/.copilot`)
   - `env_file` (string, sourced before launch)
2. Add `Copilot CopilotSettings` field to the top-level `UserConfig` struct.
3. Add `[copilot]` section to sample `config.toml` with commented defaults.
4. Add `IconCopilot` constant and tool description string.
5. Wire `"copilot"` into the tool list: New Session dialog, Fork Session, Setup Wizard, Settings panel, `default_tool` radio.

**Exit criteria:** UI pickers show Copilot; `config.toml` parses `[copilot]` with defaults; no runtime errors when Copilot is selected (session start can be a no-op or error stub).

**Tests:** Config parsing round-trip; tool list includes `"copilot"`.

---

### Phase 2 — Command builder, options, storage, initial prompt

**Goal:** `buildCopilotCommand` produces the correct shell command; session metadata is persisted; `-i PROMPT` delivers initial messages.

**Tasks:**
1. Add `CopilotOptions` struct to `internal/session/tooloptions.go`:
   - Fields: `SessionMode` (new/continue/resume), `ResumeSessionID`, `Model`, `Agent`, `YoloMode *bool`, `ConfigDir`.
   - Methods: `ToolName() string`, `ToArgs() []string`, `ToArgsForFork() []string`.
2. Implement `Instance.buildCopilotCommand(baseCommand string) string` in `internal/session/instance.go`:
   - Compose: env prefix + `AGENTDECK_INSTANCE_ID` + copilot binary + `--model` + `--agent` + `--yolo` + `--config-dir` + `--resume SESSION-ID` or `--continue` + `-i PROMPT` for initial message.
   - Follow `buildGeminiCommand` / `buildCodexCommand` patterns.
3. Add helper `buildCopilotExtraFlags()` for options→flags conversion.
4. Add fields to `Instance`:
   - `CopilotSessionID string`
   - `CopilotSessionDetectedAt *time.Time`
   - `CopilotYoloMode *bool` (per-session override)
   - `CopilotModel string` (per-session override)
   - `CopilotAgent string` (per-session override)
5. Add storage migration (new version) adding `copilot_session_id` and `copilot_session_detected_at` columns.
6. Wire `buildCopilotCommand` into the session start path (the tool dispatch in `Instance.Start` or equivalent).

**Exit criteria:** Unit tests pass for command construction covering: new session, resume by ID, continue most recent, with model/agent/yolo overrides, with `-i PROMPT` initial message, `ToArgsForFork`. Session details view shows stored Copilot metadata.

**Tests:**
- `TestBuildCopilotCommand_New`
- `TestBuildCopilotCommand_Resume`
- `TestBuildCopilotCommand_Continue`
- `TestBuildCopilotCommand_WithOptions`
- `TestCopilotOptions_ToArgs`
- `TestCopilotOptions_ToArgsForFork`
- Config→options defaults fallback

---

### Phase 3 — Session detection + resume

**Goal:** agent-deck can detect the Copilot session ID after spawn and resume sessions across restarts.

**Tasks:**
1. Implement session ID detection based on Phase 0 findings. Likely approaches (choose one):
   - **Option A:** Parse `~/.copilot/` storage directory for the most recently modified session file matching the working directory.
   - **Option B:** Scrape the tmux pane content for session ID in the welcome banner.
   - **Option C:** Rely on `--continue` exclusively (no explicit ID needed; simplest fallback).
2. If using Option A/B: implement `detectCopilotSessionAsync()` following the `detectOpenCodeSessionAsync()` pattern (quick poll loop → background watcher).
3. Set tmux environment variable `COPILOT_SESSION_ID` when detected (for resume tracking across restarts).
4. Restart flow logic:
   - If `CopilotSessionID` is set → `copilot --resume SESSION-ID`
   - If `CopilotSessionID` is empty but session existed → `copilot --continue`
   - Otherwise → fresh `copilot`
5. Implement `NewCopilotOptions(config)` factory with global config defaults.

**Exit criteria:** Tests cover resume argument logic and storage migration. Manual smoke: start a Copilot session in agent-deck, quit agent-deck, restart, session resumes without re-prompting.

**Tests:**
- `TestCopilotResume_WithSessionID`
- `TestCopilotResume_ContinueFallback`
- `TestCopilotResume_FreshStart`
- Storage migration adds fields correctly

---

### Phase 4 — Status detection

**Goal:** agent-deck's status bar accurately reflects Copilot's busy/prompt state.

**Tasks:**
1. Add `case "copilot"` to `DefaultRawPatterns()` in `internal/tmux/patterns.go` using captures from Phase 0. Expected shape:
   ```go
   case "copilot":
       return &RawPatterns{
           BusyPatterns:   []string{"esc to interrupt", /* other patterns from captures */},
           PromptPatterns: []string{/* prompt line text */, "1. Yes", "Yes, and approve"},
       }
   ```
2. Add Copilot-specific test cases to `internal/tmux/patterns_test.go`:
   - Verify busy patterns match captured thinking/spinner content.
   - Verify prompt patterns match captured idle/approval content.
   - Verify no false positives from other tools' patterns.
3. If Copilot uses unique spinner characters, add them to `SpinnerRuneSet()` for content normalization.
4. Manual smoke: start a Copilot session, verify status bar transitions between busy and prompt states during a real conversation.

**Exit criteria:** `patterns_test.go` covers Copilot patterns with no regressions to Claude/Gemini/OpenCode/Codex. Manual smoke confirms busy→prompt transitions are stable (no flicker).

**Tests:**
- `TestCopilotBusyPatterns`
- `TestCopilotPromptPatterns`
- `TestCopilotPatternsNoCollision`

---

### Phase 5 — Preflight checks + error UX

**Goal:** Users without the Copilot CLI installed get actionable guidance instead of a cryptic failure.

**Tasks:**
1. Add install check in session start path: `which copilot` or `copilot version`.
2. If binary is missing, surface a toast/tooltip with install instructions:
   - macOS: `brew install copilot-cli`
   - npm: `npm install -g @github/copilot`
   - Script: `curl -fsSL https://gh.io/copilot-install | bash`
3. No auth check needed — the CLI handles auth interactively on first launch via `/login`, or via `GH_TOKEN`/`GITHUB_TOKEN`/`COPILOT_GITHUB_TOKEN` env vars.
4. Add prerequisite mention in Settings panel alongside the tool entry.

**Exit criteria:** Simulated missing binary (e.g., renamed PATH) shows friendly error with install instructions. No crash or hang.

**Tests:** Mock/stub binary-not-found scenario triggers correct error path.

---

### Phase 6 — Documentation, polish, enable by default

**Goal:** Copilot integration is production-ready and documented.

**Tasks:**
1. Update README: add Copilot to supported tools list, mention install prerequisites.
2. Update sample `config.toml` with finalized `[copilot]` section (uncommented).
3. Add Copilot troubleshooting section (common issues: binary not found, auth expired, version mismatch).
4. Ensure session details view shows `CopilotSessionID` + `CopilotSessionDetectedAt` timestamps.
5. Surface `--experimental` (autopilot mode) in `CopilotOptions` and settings UI if desired.
6. Enable Copilot by default in tool list (remove feature flag if one was used).
7. Run full smoke checklist:
   - [ ] New session starts Copilot CLI in tmux
   - [ ] Initial prompt delivered via `-i`
   - [ ] Status bar shows busy during thinking, prompt when idle
   - [ ] Tool approval prompts detected correctly
   - [ ] Session ID detected and stored
   - [ ] Restart resumes session (--resume or --continue)
   - [ ] Fork session works (ToArgsForFork)
   - [ ] Missing binary shows install guidance
   - [ ] Config overrides (model, agent, yolo) applied correctly
   - [ ] All existing tool tests still pass (no regressions)

**Exit criteria:** Docs merged. Smoke checklist completed. Copilot at feature parity with Codex/OpenCode integrations.

---

### Future enhancements (not blocking v1)

- **ACP protocol integration:** Spawn `copilot --acp --stdio` for structured programmatic status detection instead of tmux scraping. Would require a new session management architecture; evaluate after v1 is stable.
- **MCP injection:** Use `--additional-mcp-config` to inject agent-deck's MCP servers into Copilot sessions.
- **`/delegate` tracking:** Monitor delegation to Copilot coding agent on GitHub; link resulting PRs back to agent-deck sessions.
- **`/share` integration:** Surface session export (markdown/gist) in agent-deck's session actions menu.
- **Plan mode toggle:** Expose plan mode (Shift+Tab) as a session-level option in the new session dialog.
- **Copilot Memory surfacing:** Show Copilot's persistent memory entries in session details if accessible.

## Viability Assessment (2026-02-18)

> **Note:** This document has been updated to target the current GitHub Copilot CLI
> (`github/copilot-cli`) -- a standalone binary installed via `brew install copilot-cli`
> or `npm install -g @github/copilot`. The previously archived `gh copilot` extension
> (archived Oct 30, 2025) is no longer referenced.

---

### Verdict by area

#### 1. Resume/Fork -- VIABLE (first-class support)
Resume is a documented, first-class feature:
- `copilot --resume [SESSION-ID]` -- resume a specific session by ID
- `copilot --continue` -- resume the most recently closed local session
- `/resume [SESSION-ID]` -- slash command in interactive mode cycles through local and remote sessions
- Sessions include both local and Copilot coding agent (remote) sessions

This maps perfectly to agent-deck's `buildXCommand` pattern. `CopilotSessionID` can be
stored in the `Instance` and used on restart exactly like `GeminiSessionID` or `OpenCodeSessionID`.
The `--continue` flag gives a graceful fallback when session ID is unknown.

#### 2. MCP Integration -- VIABLE (built-in + extensible)
MCP is a core feature of the Copilot CLI:
- Ships with the **GitHub MCP server** built in (issues, PRs, code search, etc.)
- Custom MCP servers added via `/mcp add` or `~/.copilot/mcp-config.json`
- `--additional-mcp-config JSON` flag for per-session MCP server injection
- `--add-github-mcp-tool`, `--add-github-mcp-toolset`, `--enable-all-github-mcp-tools` for fine-grained GitHub MCP tool control
- `--disable-builtin-mcps`, `--disable-mcp-server SERVER-NAME` for disabling
- Organization-level MCP policies partially supported (some limitations noted)

agent-deck could potentially inject its own MCP servers into Copilot sessions via `--additional-mcp-config`.

#### 3. Status Detection -- VIABLE (feasible with tmux scraping)
The Copilot CLI is a rich TUI application. Based on docs and the
CLI description, expected patterns include:
- **Busy/thinking:** "esc to interrupt" (analogous to Codex's "ctrl+c to interrupt"),
  "Thinking" spinner text, tool execution approval prompts
- **Prompt/ready:** The input prompt line (awaiting user input), `/` for slash commands
- **Approval prompts:** "1. Yes / 2. Yes, and approve TOOL... / 3. No..." --
  these could be detected as a distinct "waiting for approval" state
- **Plan mode indicator:** Different prompt when in plan mode vs ask/execute mode

Actual patterns need to be captured from a live session, but the architecture
(text-based TUI in tmux) is identical to how Claude/Gemini/OpenCode/Codex are detected.
The `/context` and `/usage` commands suggest rich terminal output suitable for scraping.

**Action needed:** Run a real Copilot CLI session in tmux, capture terminal content
during thinking/idle/approval states, and codify patterns in `DefaultRawPatterns("copilot")`.

#### 4. Preflight Checks -- VIABLE (simple)
Since `copilot` is a standalone binary, preflights are straightforward:
- **Install check:** `copilot version` (or `which copilot`) -- single binary, no `gh` dependency
- **Auth check:** The CLI itself handles auth via `/login` on first launch.
  Alternatively, users can set `GH_TOKEN` or `GITHUB_TOKEN` env vars with a
  fine-grained PAT that has "Copilot Requests" permission.
- **No `gh` dependency** -- single standalone binary

Preflight is: does `copilot` exist in PATH? Everything else
(auth, trust, config) is handled by the CLI's own interactive flows on first launch.

#### 5. Command Building -- VIABLE (well-documented flags)
Full command reference is documented. Key flags for `buildCopilotCommand`:

| Flag | Purpose |
|------|---------|
| `copilot` | Launch interactive session |
| `--resume [SESSION-ID]` | Resume a specific session |
| `--continue` | Resume most recent session |
| `-p PROMPT` / `--prompt PROMPT` | Programmatic one-shot mode |
| `-i PROMPT` / `--interactive PROMPT` | Interactive session with auto-executed first prompt |
| `--model MODEL` | Set AI model (default: Claude Sonnet 4.5) |
| `--agent AGENT` | Use a custom agent profile |
| `--allow-all` / `--yolo` | Skip all approval prompts (tools + paths + URLs) |
| `--allow-all-tools` | Skip tool approval only |
| `--allow-tool TOOL` | Allow specific tools |
| `--deny-tool TOOL` | Deny specific tools |
| `--config-dir PATH` | Custom config directory (default: `~/.copilot`) |
| `--additional-mcp-config JSON` | Inject MCP servers for this session |
| `--add-dir PATH` | Add trusted directory for file access |
| `--no-custom-instructions` | Disable AGENTS.md/instructions loading |
| `--acp [--stdio or --port N]` | Start as ACP (Agent Client Protocol) server |
| `-s` / `--silent` | Suppress usage stats (for scripting) |
| `--share [PATH]` / `--share-gist` | Export session to markdown/gist |

The command builder would compose: `copilot` + model flag + yolo/allow flags +
resume flag + additional MCP config + env prefix. This follows the exact same
pattern as `buildGeminiCommand`, `buildCodexCommand`, etc.

#### 6. ACP Protocol -- BONUS OPPORTUNITY
The Copilot CLI supports ACP (Agent Client Protocol):
- `copilot --acp --stdio` -- NDJSON-over-stdio programmatic interface
- `copilot --acp --port 3000` -- TCP mode
- Full TypeScript SDK available (`@agentclientprotocol/sdk`)
- Enables: `newSession`, `prompt`, `requestPermission`, `sessionUpdate` callbacks

This is a potential **alternative to tmux scraping** for status detection.
agent-deck could spawn `copilot --acp --stdio` and get structured session updates
instead of parsing terminal content. However, this would be a significantly different
architecture than the tmux-based approach used for all other tools. Recommend
treating ACP as a future enhancement, not blocking v1.

#### 7. Config and Auth -- VIABLE

| Aspect | Detail |
|--------|--------|
| Binary | `copilot` |
| Install | `brew install copilot-cli` / `npm i -g @github/copilot` |
| Auth | `/login` in CLI, or `GH_TOKEN`/`GITHUB_TOKEN` env var |
| Config dir | `~/.copilot/` (override via `--config-dir` or `XDG_CONFIG_HOME`) |
| Config file | `~/.copilot/config.json` |
| MCP config | `~/.copilot/mcp-config.json` |
| Session storage | Built-in (accessible via `--resume`, `--continue`) |

`CopilotSettings` struct should include:
- `command` (default `"copilot"`)
- `yolo_mode` (bool, maps to `--yolo` / `--allow-all`)
- `default_model` (string, maps to `--model`)
- `default_agent` (string, maps to `--agent`)
- `config_dir` (string, maps to `--config-dir`)
- `env_file` (string, sourced before launch)

---

### New Opportunities to Consider

- **`-i PROMPT` flag:** Start interactive session with an auto-executed first prompt -- could replace the Gemini-style "wait-and-send" hack for initial messages.
- **Custom agents:** `--agent AGENT` allows specifying agent profiles. agent-deck could surface this in the new session dialog.
- **Plan mode:** Shift+Tab cycles into plan mode. Could be exposed as a session option.
- **`/delegate`:** Hands off work to Copilot coding agent on GitHub. Could integrate with agent-deck's session tracking.
- **`/share`:** Export session to markdown or gist. Could integrate with session export features.
- **ACP server:** Long-term, `copilot --acp --stdio` enables structured programmatic control without tmux scraping.
- **`--available-tools`:** Restricts which tools Copilot can use -- could be a session-level option.
- **Autopilot mode:** Experimental mode (`--experimental`) that continues working until task is complete -- analogous to Claude's `--dangerously-skip-permissions`.

### Bottom Line

**The goal is VIABLE and arguably EASIER than originally anticipated.** The new Copilot CLI
is a modern, well-documented agentic tool with first-class support for everything the design
doc hoped for (resume, MCP, model selection, approval controls) plus features that weren't
imagined (ACP protocol, custom agents, plan mode, delegation). The architecture maps cleanly
to agent-deck's existing `buildXCommand` / `DefaultRawPatterns` / `XSettings` patterns.

The architecture maps cleanly and estimated scope of code changes is comparable to
the Codex or OpenCode integrations.

### Blocking Next Step

Install Copilot CLI (`brew install copilot-cli`), authenticate, run a real session
inside tmux, and capture terminal content during idle/thinking/approval states to
finalize status detection patterns before implementation begins.
