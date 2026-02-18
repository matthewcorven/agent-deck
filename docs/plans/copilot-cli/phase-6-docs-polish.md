# Phase 6 — Documentation, Polish, Enable by Default

> **Goal:** Copilot integration is production-ready and documented.  
> **Depends on:** All prior phases (0–5).  
> **Estimated scope:** Documentation updates + final config cleanup.

## Context

At this point, all functional code is in place. This phase covers the final documentation, config cleanup, and a comprehensive smoke checklist to validate the integration end-to-end.

## Tasks

### 1. README updates

Update `README.md`:
- Add Copilot to the "Supported Tools" list (alongside Claude, Gemini, OpenCode, Codex)
- Mention install prerequisite: `brew install copilot-cli`
- Add a brief example of starting a Copilot session

### 2. Sample config finalization

In `CreateExampleConfig()` (`internal/session/userconfig.go`), ensure the `[copilot]` section is present with the finalized field set. Uncomment defaults that are stable.

### 3. Troubleshooting section

Add to README or a separate troubleshooting doc:

| Issue | Solution |
|-------|----------|
| `copilot: command not found` | Install: `brew install copilot-cli` or `npm install -g @github/copilot` |
| Auth expired / not logged in | Run `copilot` and use `/login`, or set `GH_TOKEN` env var |
| Session resume fails | Try `copilot --continue` manually; check `~/.copilot/` for session data |
| Status bar not updating | Verify patterns match Copilot's output; check `config.toml` for pattern overrides |
| Model not available | Check `copilot /usage` for available models; verify `--model` flag spelling |

### 4. Session details view

Verify that the session details panel shows:
- `CopilotSessionID` (once detected)
- `CopilotDetectedAt` timestamp
- Active model / agent if overridden

Search for where other tool session details are rendered:
```
grep -rn 'ClaudeSessionID\|GeminiSessionID\|session_id' internal/ui/
```

### 5. Experimental features (optional)

If desired, surface in `CopilotOptions` and settings:
- `--experimental` (autopilot mode) — analogous to Claude's `--dangerously-skip-permissions`
- `--available-tools` — restrict which tools Copilot can use

These are additive and don't block the core integration.

### 6. Remove feature flag (if used)

If any `copilot.enabled` or experiment flag was used during development, remove it so Copilot is available by default.

### 7. CHANGELOG entry

Add entry to `CHANGELOG.md`:
```
### Added
- GitHub Copilot CLI (`copilot`) as a first-class tool: selectable in New Session,
  session resume/restart, status detection, config support
```

## Full Smoke Checklist

Run through every item manually before considering the integration complete:

- [ ] New session starts Copilot CLI in tmux pane
- [ ] Initial prompt delivered via `-i` or send-keys
- [ ] Status bar shows **busy** during thinking
- [ ] Status bar shows **prompt** when idle
- [ ] Tool approval prompts detected correctly (not shown as busy)
- [ ] Session ID detected and stored (check session details view)
- [ ] Restart resumes session (`--resume SESSION-ID` or `--continue`)
- [ ] Fork session works (`ToArgsForFork` produces valid flags)
- [ ] Missing binary shows install guidance (not a crash)
- [ ] `config.toml` overrides work: `yolo_mode`, `default_model`, `default_agent`, `config_dir`
- [ ] Per-session overrides (model, agent, yolo) applied correctly via New Session dialog
- [ ] Custom `command` in config works (e.g., `command = "copilot-beta"`)
- [ ] `default_tool = "copilot"` pre-selects Copilot in New Session
- [ ] All existing tool tests still pass (no regressions)
- [ ] `go test ./...` passes
- [ ] README documents Copilot support

## Future Enhancements (Not Blocking)

These are recorded here for tracking but are out of scope for the v1 integration:

- **ACP protocol:** `copilot --acp --stdio` for programmatic status detection (eliminates tmux scraping)
- **MCP injection:** `--additional-mcp-config` to inject agent-deck MCP servers
- **`/delegate` tracking:** Link Copilot coding agent PRs back to sessions
- **`/share` integration:** Expose session export in agent-deck's session actions
- **Plan mode toggle:** Expose plan mode (Shift+Tab) as a session option
- **Copilot Memory surfacing:** Show persistent memory entries in session details

## Exit Criteria

- [ ] README updated with Copilot in supported tools list
- [ ] Troubleshooting section added
- [ ] CHANGELOG entry added
- [ ] Sample config finalized
- [ ] Full smoke checklist completed with all items passing
- [ ] No feature flags remaining
- [ ] Copilot at feature parity with Codex/OpenCode integrations
