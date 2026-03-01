# Phase 6 — Execution Plan

> **Prepared by:** Ripley (Lead)  
> **Date:** 2026-03-01T03:30:49Z  
> **Status:** Ready for execution  

---

## Task 1 — README Updates

### 1A. Multi-Tool Support table (~line 265)

**File:** `README.md` line 268 area  
**Current state:** Table lists Claude Code, Gemini CLI, OpenCode, Codex, Cursor, Custom tools. **Copilot is missing.**

**Change:** Add a row for Copilot between Codex and Cursor:

```
| **Copilot CLI** | Status detection, session resume, organization |
```

Integration level matches Codex/OpenCode (status detection + organization) plus session resume (which we built in Phases 2-4). It does NOT have MCP injection or fork yet, so it shouldn't claim "Full."

**Assigned to:** Parker  
**Parallelizable:** Yes (independent of all other tasks)

### 1B. "The Problem" section (~line 59)

**File:** `README.md` line 59  
**Current state:** `"Running Claude Code on 10 projects? OpenCode on 5 more? Another agent somewhere in the background?"`

**Change:** Mention Copilot. Suggested:  
`"Running Claude Code on 10 projects? Copilot on 5 more? OpenCode and Codex in the background?"`

**Assigned to:** Parker

### 1C. Install prerequisite mention

**File:** `README.md` (Installation section, ~line 280+)  
**Current state:** No prerequisites section exists. Agent Deck installs via curl/brew/go. Individual tool install is the user's responsibility.  
**Finding:** No other tool (Claude, Gemini, OpenCode, Codex) has an install prerequisite mentioned in README. The troubleshooting doc is the appropriate place (see Task 3).

**Decision:** Do NOT add a prerequisite to the README install section — it would be inconsistent. The preflight check (Phase 5) already gives actionable install guidance at runtime. Troubleshooting doc (Task 3) covers it.

**Assigned to:** N/A (no change needed)

---

## Task 2 — Sample Config Finalization

**File:** `internal/session/userconfig.go` lines 1460–1472 (`CreateExampleConfig()`)  
**Current state:** `[copilot]` section is **present and complete**. All 6 `CopilotSettings` fields are represented:

| CopilotSettings field | Sample config line | Present? |
|---|---|---|
| `Command` | `command = "copilot"` | ✅ |
| `YoloMode` | `yolo_mode = true` | ✅ |
| `DefaultModel` | `default_model = ""` | ✅ |
| `DefaultAgent` | `default_agent = ""` | ✅ |
| `ConfigDir` | `config_dir = "~/.copilot"` | ✅ |
| `EnvFile` | `env_file = ""` | ✅ |

Cross-checked against `CopilotSettings` struct (lines 519-540): **perfect 1:1 match.** Also verified `default_tool` comment includes `"copilot"` in the valid values list (line 1429 area).

**Decision:** No changes needed. Task 2 is already complete from Phase 1.

**Assigned to:** N/A (verified complete)

---

## Task 3 — Troubleshooting Section

### Precedent analysis

**Existing location:** `skills/agent-deck/references/troubleshooting.md`  
This is the canonical troubleshooting doc. It covers: Quick Fixes table, Flags Ignored, MCP Not Available, Session ID Not Detected, Conductor Permissions, High CPU, Log Files, Global Search, Debugging, Bug Reporting.

All troubleshooting content is tool-agnostic or Claude-specific (session ID, conductor permissions). **No per-tool troubleshooting sections exist.**

**Decision:** Add a `### Copilot CLI Issues` section to `skills/agent-deck/references/troubleshooting.md`, placed after the "Conductor Keeps Asking for Permissions" section (~line 70). This follows the existing pattern of tool-specific subsections within the shared troubleshooting doc.

Additionally, add a Copilot row to the Quick Fixes table at line 8.

**Content (from Phase 6 design doc):**

Quick Fixes table row:
```
| Copilot not found | Install: `brew install copilot-cli@prerelease` or `npm install -g @github/copilot` |
```

New section:
```markdown
### Copilot CLI Issues

| Issue | Solution |
|-------|----------|
| `copilot: command not found` | Install: `brew install copilot-cli@prerelease` or `npm install -g @github/copilot` |
| Auth expired / not logged in | Run `copilot` and use `/login`, or set `GH_TOKEN` env var |
| Session resume fails | Try `copilot --continue` manually; check `~/.copilot/` for session data |
| Status bar not updating | Verify patterns match Copilot's output; check `config.toml` for pattern overrides |
| Model not available | Check `copilot /usage` for available models; verify `--model` flag spelling |
```

**Assigned to:** Parker  
**Parallelizable:** Yes (independent)

---

## Task 4 — Session Details View

**File:** `internal/ui/home.go` lines 7835–7855  
**Current state:** The session details panel has a **Claude-only** section gated by `if selected.Tool == "claude"`. It renders:
- Section divider: `renderSectionDivider("Claude", ...)`
- Status: Connected/Not connected based on `ClaudeSessionID`
- Session ID: `selected.ClaudeSessionID`
- MCP servers

**Finding:** **No equivalent Copilot section exists.** CopilotSessionID is populated by session detection (Phase 3), but it's never rendered in the details panel.

**Change:** Add a Copilot section after the Claude section, gated by `selected.Tool == "copilot"`. It should render:
- Section divider: `renderSectionDivider("Copilot", ...)`
- Status: Connected/Not connected based on `CopilotSessionID`
- Session ID: `selected.CopilotSessionID`
- Detected At: `selected.CopilotDetectedAt` (formatted)
- Model: from `CopilotOptions.Model` if extracted from tool data
- Agent: from `CopilotOptions.Agent` if extracted from tool data

Follow the exact Claude pattern. No MCP sub-section (Agent Deck doesn't manage Copilot MCP yet).

**Assigned to:** Parker (implementation), Lambert (test verification)  
**Parallelizable:** Yes (independent)

---

## Task 5 — Experimental Features (Optional)

**Design doc says:** Surface `--experimental` (autopilot mode) and `--available-tools` in CopilotOptions.

**Current state of CopilotOptions** (tooloptions.go lines 299-315): Has SessionMode, ResumeSessionID, Model, Agent, YoloMode, ConfigDir. No Experimental or AvailableTools fields.

**Assessment:**
- `--experimental` is Copilot's autopilot mode. It's analogous to Claude's `--dangerously-skip-permissions`. Copilot already has `--yolo` which we support. `--experimental` is a separate, more aggressive flag. Adding it is low-effort (~10 lines in CopilotOptions + ToArgs + ToArgsForFork + NewCopilotOptions, plus CopilotSettings struct).
- `--available-tools` restricts which tools Copilot can use. This is a string list, more complex to surface in TUI.

**Decision:** Defer both to a follow-up. The Phase 6 doc correctly marks these as "optional" and "additive." They don't block the core integration. The `--yolo` flag is already supported for the primary auto-approve use case.

**Assigned to:** N/A (deferred)

---

## Task 6 — Remove Feature Flag

**Current state:** Searched the entire codebase for `copilot.enabled`, `copilotEnabled`, `CopilotEnabled`, and any experiment flag in `internal/experiments/`. 

**Finding:** **No feature flag exists.** The experiments package (`internal/experiments/experiments.go`) is unrelated — it manages the `agent-deck try` command for experiment folders, not feature flags. Copilot was never gated behind a flag; it was always additive (added to tool lists, config struct, pattern switch cases).

**Decision:** No changes needed. Task 6 is already satisfied.

**Assigned to:** N/A (verified complete)

---

## Task 7 — CHANGELOG Entry

**File:** `CHANGELOG.md`  
**Current state:** Latest entry is `[0.19.19] - 2026-02-26`. The Copilot integration spans Phases 1-5 and should be released as a feature in the next version.

**Change:** Add a new version section at the top (after the header, before `[0.19.19]`). Version should be `0.20.0` — this is a significant feature addition (new tool) warranting a minor version bump per semver.

```markdown
## [0.20.0] - 2026-03-01

### Added

- GitHub Copilot CLI (`copilot`) as a first-class tool: selectable in New Session / Fork / Setup Wizard, session resume via `--resume` / `--continue`, status detection (busy/idle patterns), config support (`[copilot]` section in config.toml), and preflight binary checks with install guidance.
- Copilot session details in TUI: session ID, detection timestamp, connected status.
- Copilot troubleshooting section in troubleshooting guide.
```

**Assigned to:** Parker  
**Parallelizable:** Yes (independent)

---

## Additional Verification Results

### `internal/session/instance.go` — TODOs/FIXMEs

**Result:** No TODO or FIXME comments found related to Copilot (or at all). Clean.

### `internal/session/tooloptions.go` — CopilotOptions completeness

**Verified fields:** SessionMode, ResumeSessionID, Model, Agent, YoloMode, ConfigDir.  
**Methods:** ToolName(), ToArgs(), ToArgsForFork(), NewCopilotOptions(), UnmarshalCopilotOptions().  
**Assessment:** Complete for v1. All fields match what `buildCopilotCommand()` and `buildCopilotExtraFlags()` consume.

### `internal/tmux/patterns.go` — Copilot patterns

**Present at line 78.** BusyPatterns: `"Esc to cancel"`, `re:(?m)^[◉◐◎∙]\s`. PromptPatterns: `"Type @ to mention files"`. Matches Phase 0 capture findings exactly. Complete.

### `internal/ui/styles.go` — Copilot icon/color

**Icon:** `IconCopilot = "🛸"` (line 193). Present in `ToolIcon()` switch (line 599).  
**Color:** `lipgloss.Color("#6e40c9")` (GitHub purple) in `ToolColor()` switch (line 620). Complete.

### `internal/ui/` — Full copilot reference audit

All expected UI sites are wired:
- `setup_wizard.go:56` — tool options list includes "copilot" ✅
- `newdialog.go:97` — command presets include "copilot" ✅
- `settings_panel.go:81-82` — tool names/values include "Copilot"/"copilot" ✅
- `home.go:5151-5152` — `createSessionInGroupWithWorktreeAndOptions()` has `case "copilot"` ✅ (Phase 5 fix)
- `styles.go` — icon + color ✅

### Missing: Session details panel (Task 4 above — needs implementation)

---

## Execution Summary

| Task | Status | Agent | Effort | Parallel? |
|------|--------|-------|--------|-----------|
| 1A — README tool table | **Needs work** | Parker | 5 min | ✅ |
| 1B — README "The Problem" | **Needs work** | Parker | 2 min | ✅ |
| 1C — README prerequisites | **No change** | — | — | — |
| 2 — Sample config | **Already complete** | — | — | — |
| 3 — Troubleshooting | **Needs work** | Parker | 10 min | ✅ |
| 4 — Session details view | **Needs work** | Parker + Lambert | 30 min | ✅ |
| 5 — Experimental features | **Deferred** | — | — | — |
| 6 — Remove feature flag | **No flag exists** | — | — | — |
| 7 — CHANGELOG entry | **Needs work** | Parker | 5 min | ✅ |

**All 5 actionable tasks (1A, 1B, 3, 4, 7) are independent and can be parallelized.**

Task 4 (session details panel) is the only one requiring both Parker (implementation) and Lambert (test). All others are documentation-only changes by Parker.

---

## Issues Discovered

1. **Session details panel gap (Task 4):** This is the only functional code change in Phase 6. CopilotSessionID is detected and stored but never rendered in the TUI details panel. This is a visibility gap — users can't see the session ID they need for manual `--resume` operations.

2. **No Gemini/OpenCode/Codex details panels either:** The session details panel is Claude-only. A Copilot-specific section would be the second tool-specific panel. This is fine — it establishes the precedent for eventually adding panels for other tools.

3. **Version number decision:** Recommending `0.20.0` for the Copilot feature release. This is a judgment call — `0.19.20` would also be defensible if the project prefers to stay in the 0.19.x series. Matthew should confirm.
