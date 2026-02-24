### 2026-02-24: Upstream Review — 51 Commits (v0.19.2 → v0.19.14)

**By:** Ripley (Lead)
**Requested by:** Matthew Corven
**Scope:** 59 files, +5873/−807 lines across v0.19.2 through v0.19.14

---

## Executive Summary

The upstream has evolved significantly since our fork. The changes are overwhelmingly **positive** for our Copilot CLI integration — they introduce patterns, infrastructure, and reliability improvements that map directly onto phases we planned to build. However, they also change APIs and file structures our plan assumed would be stable, requiring **phase doc updates and merge conflict resolution** before we can proceed with implementation.

**Overall verdict:** Merge upstream ASAP. The longer we wait, the worse the divergence. Every change area either helps us or is neutral — nothing blocks our Copilot integration.

---

## 1. Session Lifecycle — `instance.go` (+351 lines)

### Changes
- **Codex scan rate-limiting:** New `shouldScanCodexSession()` with bootstrap/rotation intervals prevents expensive filesystem scans. Adds `lastCodexScanAt` field to Instance struct.
- **`sendMessageWhenReady()` overhaul:** Major reliability hardening. Now tracks `readyCount` (was `waitingCount`), accepts `"idle"` as a ready state alongside `"waiting"`, adds Claude-specific composer prompt detection (`hasCurrentComposerPrompt`, `hasUnsentComposerPrompt`, `hasUnsentPastedPrompt`), and implements post-send verification with retry logic (50 retries, 300ms delay, periodic Enter nudges).
- **Shell status fix:** `UpdateStatus()` now maps `"waiting"` → `StatusIdle` for shell-type sessions, preventing false "waiting" flags.
- **`GetLastResponseBestEffort()`:** New method with Claude-specific recovery path (refresh from tmux env → disk scan → terminal fallback → empty response). Used by `session send` and `session output`.
- **Composer prompt parsing:** ~170 lines of new prompt detection logic (`currentComposerPrompt`, `parsePromptFromComposerBlock`, `isComposerDividerLine`, `normalizePromptText`) for Claude's TUI composer.

### Goal Impact: **HELPS** (Phase 2, 3, 4)
- The `sendMessageWhenReady()` pattern is exactly what we need for Copilot CLI. The `"idle"` status recognition means Copilot sessions that land in idle state will be correctly handled.
- `GetLastResponseBestEffort()` provides the fallback chain pattern we should follow for Copilot's output retrieval.
- The Codex scan rate-limiter is the pattern to follow if Copilot CLI has a similar filesystem session detection path.

### Implementation Plan Impact
- **Phase 2 (Command Builder):** No changes needed.
- **Phase 3 (Session Detection):** Our plan assumed `ClaudeSessionID` pattern for session tracking. Upstream shows the full recovery chain (tmux env → disk → terminal). Copilot likely needs a similar multi-fallback strategy. **Update phase-3 doc.**
- **Phase 4 (Status Detection):** The `"idle"` state acceptance in `sendMessageWhenReady()` means our Copilot status patterns should distinguish between idle and waiting states. The composer prompt detection is Claude-specific but the *pattern* (parse TUI output to verify send success) may apply to Copilot. **Update phase-4 doc.**

### Merge Conflict Risk: **LOW-MEDIUM**
- Our fork hasn't modified `instance.go`. Clean merge expected.
- However, when we add Copilot-specific session detection (Phase 3), we'll be adding alongside the existing `UpdateCodexSession()` pattern — no conflict, just adjacent adds.

### API Changes
- `Instance` struct gains `lastCodexScanAt` field (unexported, no impact on us).
- `GetLastResponseBestEffort()` is new API — we should use it instead of `GetLastResponse()` for Copilot.
- `sendMessageWhenReady()` signature unchanged but behavior significantly different (idle acceptance, verification loop).

### New Patterns to Adopt
- **Scan rate-limiting:** Add similar `lastCopilotScanAt` if Copilot has filesystem-based session detection.
- **Best-effort response retrieval:** Use `GetLastResponseBestEffort()` for Copilot CLI output.
- **Post-send verification:** The Enter-nudge retry pattern is critical for any TUI tool.

---

## 2. Conductor Subsystem — `conductor.go` (+667 lines) + `conductor_templates.go` (+343 lines)

### Changes
- **Policy/Learnings split:** Auto-response rules moved from shared `CLAUDE.md` into separate `POLICY.md`. New `LEARNINGS.md` for conductor self-improvement patterns.
- **`SetupConductor()` signature change:** Now takes `customPolicyMD string` as 6th parameter (was 5-param).
- **`InstallPolicyMD()`, `InstallLearningsMD()`:** New functions for the policy/learnings files.
- **Migration functions:** `MigrateConductorPolicySplit()`, `MigrateConductorLearnings()`, `MigrateConductorHeartbeatScripts()` — safe template migrations that only touch exact-match generated files.
- **PATH management:** `buildDaemonPath()` prepends agent-deck binary dir to daemon PATH. Heartbeat scripts now use `--no-wait -q` flags.
- **Transition notifier daemon:** New `TransitionNotifierLaunchdPlistName`, plist templates, systemd templates for the transition notification system.
- **`findPython3()` rewrite:** Now prefers `exec.LookPath("python3")` (respects pyenv/asdf) before hardcoded paths.
- **`ExpandPath()` adoption:** `createSymlinkWithExpansion()` now uses `ExpandPath()` instead of inline tilde expansion.

### Goal Impact: **NEUTRAL** (conductor is orthogonal to Copilot CLI integration)
- No direct impact — Copilot sessions don't interact with the conductor subsystem differently than any other tool.
- Indirectly helpful: if Copilot is used inside a conductor-orchestrated workflow, the transition notifier will correctly detect Copilot sessions (it's tool-agnostic for status polling).

### Implementation Plan Impact: None

### Merge Conflict Risk: **LOW**
- Our fork hasn't modified conductor files. The `.beads/` cleanup in upstream will conflict with our local `.beads/` additions but that's in a different domain.

### API Changes
- `SetupConductor()` now has 6 parameters (added `customPolicyMD`). Not called by our Copilot code.
- `SetupConductorProfile()` calls updated signature — backward compat wrapper exists.

### New Patterns to Adopt
- **`ExpandPath()` as canonical path expander:** Any path from user config should go through `ExpandPath()`. Our Copilot config paths must follow this pattern.

---

## 3. New Subsystems — Transition Daemon + Notifier (+732 lines combined)

### Changes
- **`transition_daemon.go` (395 lines):** Long-running daemon that polls all profiles for status transitions (running→waiting, running→idle, running→error). Adaptive polling intervals (1s/2s/3s based on active session count). Reads hook status files for Claude/Codex fast-path detection.
- **`transition_notifier.go` (337 lines):** Delivers transition notifications to parent sessions or fallback conductor. Implements dedup (90s window), structured logging, state persistence (`transition-notify-state.json`).
- **`send_helper.go` (61 lines):** `SendSessionMessageReliable()` — wraps `agent-deck session send` CLI call for programmatic use. Uses `agentDeckBinaryPath()` to locate the binary.
- **`notify_daemon_cmd.go` (42 lines):** New CLI command `agent-deck notify-daemon` with `--once` flag.

### Goal Impact: **HELPS** (Phase 4, Phase 5)
- The transition daemon is **tool-agnostic for status polling**. When we add Copilot status detection, the transition daemon will automatically pick up Copilot sessions and deliver notifications — zero extra work needed.
- The `SendSessionMessageReliable()` helper is available for any Copilot-specific automation.
- The `notify-daemon` command pattern shows how upstream adds new subcommands — useful reference for any Copilot-specific CLI additions.

### Implementation Plan Impact
- **Phase 4 (Status Detection):** Our Copilot status patterns will be consumed by the transition daemon automatically. No extra integration needed. **Note this in phase-4 doc.**

### API Changes
- New exported types: `TransitionDaemon`, `TransitionNotifier`, `TransitionNotificationEvent`, `ShouldNotifyTransition()`.
- New exported function: `SendSessionMessageReliable()`.
- None of these conflict with our planned additions.

### New Capabilities to Leverage
- **Transition notifications for Copilot:** Once Copilot status detection works (Phase 4), the transition daemon automatically routes Copilot waiting/error states to parent/conductor sessions.
- **`SendSessionMessageReliable()`:** Can be used in any Copilot-specific automation scripting.

---

## 4. Config & Environment — `userconfig.go` (+40 lines) + `env.go` (+34 lines)

### Changes
- **`ManageMCPJson *bool`:** New config field to disable `.mcp.json` writes. Default true (nil=true pattern).
- **`ExpandPath()` replaces `expandHomePath()`:** Now handles `$HOME`, `${VAR}`, `~/` — environment variable expansion before tilde expansion.
- **`resolvePath()` replaces `resolveEnvFilePath()`:** Same signature, uses `ExpandPath()` internally.
- **Documentation updates:** All `env_file` comments now mention `$HOME/${VAR}` support.
- **`GetManageMCPJson()` function:** New accessor with nil-default-true pattern.
- **Import cleanup:** Removed unused `strings` import from `userconfig.go`.

### Goal Impact: **HELPS** (Phase 1)
- **`ExpandPath()` is the canonical utility** for all path handling in config. Our `CopilotSettings` struct (Phase 1) must use it for any `config_dir`, `env_file`, or similar path fields.
- **`ManageMCPJson`:** Pattern for adding a new boolean config field with nil-default — exactly what we need for potential `copilot.allow_dangerous_mode` or similar flags.

### Implementation Plan Impact
- **Phase 1 (Config Surface):** Our `CopilotSettings` struct plan references `expandTilde`. Must update to use `ExpandPath()` instead. The nil-pointer-with-default pattern for boolean flags should be adopted for any optional Copilot config fields. **Update phase-1 doc.**

### Merge Conflict Risk: **HIGH**
- We modified `config-reference.md` to add a Copilot section and update the Table of Contents. Upstream also modified this file significantly (+76 lines: shell section, skills registry, path resolution, env_file docs).
- Result: **guaranteed merge conflict** in `skills/agent-deck/references/config-reference.md`. Content-wise it's additive on both sides, so resolution is straightforward (keep both additions).
- `userconfig.go` itself: LOW risk. Our Copilot section additions are in a different region of the file.

### API Changes
- **`expandHomePath()` → `ExpandPath()` (RENAMED AND EXTENDED).** Any code referencing the old function name won't compile. Our Phase 1 plan referenced using `expandTilde` — must update.
- **`resolveEnvFilePath()` → `resolvePath()` (RENAMED).** Internal function, but if any of our code planned to call it, update the name.
- **`expandTilde()` in storage.go REMOVED.** Replaced with `fixMalformedTildePath()` + `ExpandPath()`. Our code must not call `expandTilde()`.

### New Patterns to Adopt
- **`ExpandPath()`:** Call for ALL user-provided paths in config.
- **`*bool` with nil-default pattern:** Use for optional boolean config fields (e.g., `copilot.hooks_enabled`).
- **`GetManageMCPJson()` accessor pattern:** Model our `GetCopilotHooksEnabled()` accessor after this.

---

## 5. Storage & StateDB — `storage.go` (+65 lines) + `statedb.go` (+25 lines)

### Changes
- **`expandTilde()` removed** from storage.go. Replaced with `fixMalformedTildePath()` + `ExpandPath()`.
- **`migrateStateDBWithRetry()`:** New retry wrapper (6 attempts, exponential backoff) for SQLite `SQLITE_BUSY` / "database is locked" errors during migration.
- **`isSQLiteBusyError()`:** Helper to detect busy errors.
- **Schema version write optimization:** `statedb.Migrate()` now checks existing version before writing, reducing lock contention.
- **Path expansion in instance loading:** `convertToInstances()` now uses `ExpandPath(fixMalformedTildePath(...))` instead of `expandTilde()`.

### Goal Impact: **HELPS** (Phase 2, Phase 3)
- The retry-on-busy pattern is critical. With the new transition daemon running as a background process, SQLite contention is a real concern. Our Copilot integration writes to the same database — we get this protection for free.
- Schema migration optimization benefits us because we may add Copilot-specific columns in Phase 2.

### Implementation Plan Impact
- **Phase 2 (Command Builder + Storage):** Our plan mentions adding columns to statedb. The retry wrapper means we don't need to worry about contention with the new transition daemon. **No doc update needed but good to know.**

### Merge Conflict Risk: **LOW**
- Our fork hasn't modified `storage.go` or `statedb.go` for code changes. We planned to ADD migrations, not modify existing ones.

### API Changes
- `expandTilde()` is REMOVED. Not exported, so no external impact, but if any of our planned code referenced it: use `ExpandPath()` instead.

---

## 6. CLI Commands — `session_cmd.go` (+346), `main.go` (+101), `launch_cmd.go` (+69), `cli_utils.go` (+76)

### Changes
- **`resolveSessionCommand()`:** New function that parses `--cmd` input to split tool name from extra args (e.g., `"codex --dangerously-bypass-approvals-and-sandbox"` → tool=codex, wrapper=`{command} --dangerously-bypass-approvals-and-sandbox`). Returns `(toolName, command, wrapper, note)`.
- **`firstNonEmpty()`:** New utility for flag merging.
- **`resolveGroupSelection()`:** New function for parent-group inheritance with explicit override.
- **`resolveAutoParentInstance()`:** Auto-detects parent session from `AGENT_DECK_SESSION_ID`, `AGENTDECK_INSTANCE_ID`, or current tmux session.
- **`--no-parent` flag:** New flag on `add` and `launch` to disable automatic parent linking.
- **`session show` JSON output:** Now includes `parent_session_id`, `parent_project_path`.
- **`session send` uses `GetLastResponseBestEffort()`** instead of `GetLastResponse()`.
- **TUI shortcut remap:** `m` = MCP Manager, `s` = Skills Manager, `M` = Move, `r` = Rename, `R` = Restart, `S` = Settings.
- **`handleNotifyDaemon()`:** New CLI command handler for `notify-daemon`.
- **Version bump:** `0.19.1` → `0.19.14`.
- **`sendWithRetry()` refactored** with extracted `sendRetryTarget` interface and `sendRetryOptions`.

### Goal Impact: **HELPS** (Phase 1, Phase 2)
- **`resolveSessionCommand()` is directly applicable to Copilot CLI.** When a user types `--cmd copilot`, our tool detection must work with this new resolution pipeline. Our `detectTool()` addition for "copilot" feeds into this new `resolveSessionCommand()` flow.
- **`--no-parent` flag:** Available for Copilot sessions that shouldn't auto-link to a parent.
- **`sendWithRetry()` refactoring:** The `sendRetryTarget` interface makes it testable and potentially reusable for Copilot-specific send logic.

### Implementation Plan Impact
- **Phase 1 (Config Surface):** Our `detectTool()` modification to add "copilot" now feeds into the upstream `resolveSessionCommand()` pipeline. Must verify compatibility. **Update phase-1 doc.**
- **Phase 2 (Command Builder):** The command resolution flow changed — `resolveSessionCommand()` now handles the tool→command mapping before the Instance is created. Our Copilot command builder must work within this new flow, not the old `if toolDef := session.GetToolDef(...)` pattern that was inlined. **Update phase-2 doc.**

### Merge Conflict Risk: **CRITICAL**
- **`main.go`:** Our fork has the old version string (`0.19.1`), upstream is `0.19.14`. Our fork also doesn't have the new `resolveAutoParentInstance()`, `resolveGroupSelection()`, or `notify-daemon` command. These are ADDITIONS in different regions, so git should auto-merge, but the version line is a guaranteed conflict.
- **`cli_utils.go`:** Upstream ADDED `resolveSessionCommand()`, `firstNonEmpty()`, `resolveGroupSelection()`, `splitFirstWord()` — 76 new lines. Our fork has the pre-change version. This is a **one-way merge** (we take upstream), no conflict if we haven't modified this file.
- **`session_cmd.go`:** Upstream added ~346 new lines. Our fork has the pre-change version. One-way merge.
- **`launch_cmd.go`:** Upstream added 69 lines. Our fork has the pre-change version. One-way merge.

### New Patterns to Adopt
- **`resolveSessionCommand()` pipeline:** Our `detectTool("copilot")` must return tool detection compatible with this pipeline.
- **`sendRetryTarget` interface:** If Copilot needs custom send verification, implement this interface.
- **`CLIOutput` struct:** Use for all Copilot CLI output (already exists, just follow the pattern).

---

## 7. tmux Layer — `pty.go` (+36 lines) + `tmux.go` (+26 lines)

### Changes
- **Terminal style leak fix:** `cleanupAttach()` function in `pty.go` ensures OSC-8 hyperlink state and SGR attributes are reset before returning to Bubble Tea after detach. Prevents underline/color leakage from attached sessions.
- **`CapturePaneFresh()`:** New method in `tmux.go` that bypasses the control-mode pipe cache to get a fresh tmux pane snapshot. Uses direct `tmux capture-pane` subprocess call with 3s timeout.
- **Output completion channel:** `outputDone` channel ensures PTY output goroutine finishes before returning from Attach.

### Goal Impact: **HELPS** (Phase 0, Phase 4)
- **`CapturePaneFresh()` is critical for Copilot Phase 0 (Live Capture).** When Parker is building Copilot status detection patterns, this fresh capture method gives reliable snapshots for pattern matching. The cached `CapturePane()` might return stale data.
- **Style leak fix:** Copilot's TUI likely uses ANSI styling. This fix prevents our sessions from leaking Copilot's styling into the Agent Deck UI.

### Implementation Plan Impact
- **Phase 0 (Live Capture):** Mention `CapturePaneFresh()` as the preferred capture method for pattern development. **Update phase-0 doc.**
- **Phase 4 (Status Detection):** Use `CapturePaneFresh()` for Copilot status verification (analogous to how `sendMessageWhenReady()` uses it for Claude).

### Merge Conflict Risk: **LOW**
- Our fork hasn't modified tmux files.

### API Changes
- New exported method: `(*Session).CapturePaneFresh() (string, error)`.
- No existing method signatures changed.

### New Capabilities to Leverage
- Use `CapturePaneFresh()` in Phase 0 captures and Phase 4 status detection.

---

## 8. UI Layer — `home.go`, `skill_dialog.go`, `mcp_dialog.go`, `newdialog.go`, `help.go`

### Changes
- **TUI shortcut remap:** `m/s/M/S` reassigned (MCP Manager, Skills Manager, Move, Settings). `R` = Restart (was Rename), `r` = Rename.
- **Skill dialog overhaul (+250 lines):** Pool-focused search, scrolling, type-to-jump.
- **MCP dialog (+86 lines):** Type-to-jump navigation.
- **Skills validation:** `validateAttachableSkillCandidate()` ensures only directory skills with SKILL.md can be attached. New `ErrSkillUnsupportedKind` error.
- **Skill materialization hardening:** Symlink validation (resolves macOS /tmp→/private/tmp), falls back to copy if symlink is broken.
- **Help view (+8 lines):** Updated shortcut documentation.

### Goal Impact: **NEUTRAL → SLIGHTLY HELPS**
- The skill/MCP dialog changes don't affect Copilot integration directly.
- The shortcut remap is a UI concern — our Phase 1 TUI integration (adding Copilot to the new session dialog) must respect the current shortcut assignments.

### Implementation Plan Impact
- **Phase 1 (Config Surface):** The new session dialog (`newdialog.go`) had minor changes (+16 lines). Our plan to add Copilot as a tool option in this dialog should still work. **Verify newdialog.go compatibility.**

### Merge Conflict Risk: **LOW**
- Our fork hasn't modified UI files.

---

## 9. Installer & Bridge — `install.sh` (+90 lines) + `bridge.py` (+92 lines)

### Changes
- **`install.sh`:** macOS Bash 3.2 compatibility fix (arrays, test syntax), better error handling.
- **`bridge.py`:** Enhanced command parsing, robust heartbeat delivery with `--no-wait` flag, improved error handling.

### Goal Impact: **NEUTRAL**
- These changes don't affect Copilot CLI integration.

### Merge Conflict Risk: **LOW**
- Our fork has minor `install.sh` differences (the diff showed these in the initial stat). Should auto-merge.

---

## 10. Update System — `update.go` (+20 lines)

### Changes
- **Homebrew detection:** `homebrewUpgradeHint()` detects Cellar-managed installs and returns appropriate `brew upgrade` command instead of self-updating.

### Goal Impact: **NEUTRAL**
- No impact on Copilot integration.

### Merge Conflict Risk: **LOW**

---

## 11. Tests — Various files (+substantial coverage)

### Changes
- **`session_send_test.go` (+257 lines):** New tests for send retry logic, prompt detection.
- **`conductor_test.go` (+814 lines):** Comprehensive conductor tests.
- **`instance_test.go` (+146 lines):** Instance lifecycle tests.
- **`cli_utils_test.go` (+110 lines):** Tests for `resolveSessionCommand()`, `firstNonEmpty()`, etc.
- **`skill_dialog_test.go` (+285 lines):** Skill dialog UI tests.
- **`mcp_dialog_test.go` (+77 lines):** MCP dialog tests.
- **`home_test.go` (+91 lines):** Home view tests.
- **`transition_notifier_test.go` (+125 lines):** Transition notifier tests.
- **`skills_catalog_test.go` (+109 lines):** Skills catalog tests.
- **`session_cmd_test.go` (+46 lines):** Session command tests.
- **`update_test.go` (+41 lines):** Update system tests.
- **`env_test.go` (+25 lines):** Environment expansion tests.

### Goal Impact: **HELPS**
- These tests establish patterns for how to test new tool integrations. Our Copilot tests (Phase 2+) should follow the same conventions.
- The `session_send_test.go` patterns are directly reusable for Copilot send verification tests.

### Merge Conflict Risk: **LOW**
- Our fork hasn't modified test files beyond what was committed in the initial plan.

---

## Prioritized Action List

### Critical (Do Before Any Code)
1. **Merge upstream into our fork.** `git merge upstream/main`. Expected conflicts:
   - `skills/agent-deck/references/config-reference.md` — resolve by keeping both our Copilot section and upstream's shell/skills/path sections.
   - `cmd/agent-deck/main.go` — version string conflict (accept upstream `0.19.14`).
   - `README.md` — minor formatting differences.
   - `CHANGELOG.md` — our fork removed it; upstream added 116 lines. Accept upstream.
   - `.beads/` — our fork added it; upstream removed it. Decision needed (keep or remove).

2. **Update `ExpandPath()` references** in all phase docs. Replace any mention of `expandTilde`, `expandHomePath`, or `resolveEnvFilePath` with the new function names.

### High Priority (Update Phase Docs)
3. **Phase 0 doc:** Add note about `CapturePaneFresh()` as preferred capture method.
4. **Phase 1 doc:** Update to reference `ExpandPath()`, the nil-pointer boolean pattern for optional config, and `resolveSessionCommand()` compatibility for `detectTool("copilot")`.
5. **Phase 2 doc:** Update command resolution flow to work within the new `resolveSessionCommand()` pipeline instead of inline `GetToolDef()` calls.
6. **Phase 3 doc:** Add multi-fallback session recovery pattern (tmux env → disk → terminal) based on `GetLastResponseBestEffort()`.
7. **Phase 4 doc:** Note that transition daemon will automatically detect Copilot sessions. Add `CapturePaneFresh()` for status verification. Note `"idle"` status recognition.

### Medium Priority (Design Decisions)
8. **Decide on `.beads/` directory.** Upstream removed it; we added it. Remove it to stay aligned with upstream.
9. **Evaluate transition notifier for Copilot.** Once Phase 4 is complete, Copilot sessions will automatically get transition notifications. No extra work needed, but document this as a "free" feature.

### Low Priority (Nice to Have)
10. **Adopt `sendRetryTarget` interface** if Copilot needs custom send verification logic.
11. **Use `SendSessionMessageReliable()`** for any Copilot-specific automation scripting.

---

## Merge Strategy Recommendation

**Strategy: `git merge upstream/main` (not rebase)**

Rationale:
- Our fork has 13 commits with distinct work (plan docs, squad setup, config reference additions). Rebasing would replay these onto upstream's 51 commits, creating unnecessary conflict resolution steps.
- Merge preserves our commit history and creates a clear merge point.
- The conflicts are limited to 4-5 files, all resolvable with additive content merging.

**Timing: Immediately.** We're in Phase 0 (no code changes to Copilot integration yet). The cost of merging is minimal now — it grows with every line of implementation code we write against the stale API surface.

**Post-merge verification:**
```bash
go build ./...        # Verify compilation
go test ./...         # Verify all tests pass
```

---

## Phase Plan Updates Needed

| Phase | Update | Severity |
|-------|--------|----------|
| Phase 0 | Add `CapturePaneFresh()` as preferred capture method | Low |
| Phase 1 | Replace `expandTilde` → `ExpandPath()`, add `resolveSessionCommand()` compat | **High** |
| Phase 2 | Update command resolution to use `resolveSessionCommand()` pipeline | **High** |
| Phase 3 | Add multi-fallback session recovery chain pattern | Medium |
| Phase 4 | Note transition daemon auto-detection, `CapturePaneFresh()`, idle recognition | Medium |
| Phase 5 | No changes needed | None |
| Phase 6 | No changes needed | None |
