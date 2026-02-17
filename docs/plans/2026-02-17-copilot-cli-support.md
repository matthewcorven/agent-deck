# GitHub Copilot CLI Support — Design Document

> Generated: 2026-02-17  
> Brainstorm perspectives: Architect, Implementer, Devil's Advocate, Release/Support  
> Chosen approach: First-class Copilot tool (built-in parity with Claude/Gemini/OpenCode/Codex)

## Summary

Add GitHub Copilot CLI (`gh copilot`) as a first-class agent-deck tool with the same UX/features as Claude, Gemini, OpenCode, and Codex: selectable in New Session/Setup Wizard, persisted session tracking/resume, status detection, configuration, and guardrails. The Copilot CLI is distributed as a GitHub CLI extension and relies on `gh auth status`; the integration must handle install/auth preflights and map Copilot’s conversational loop into tmux-managed sessions.

## Acceptance Criteria

- [ ] Copilot appears as a selectable tool (icon + description) in New Session, Fork Session, Setup Wizard, and settings with default_tool support.
- [ ] `[copilot]` config section added with defaults (command `gh copilot`, profile override, env file, optional model/mode flags) and documented in sample `config.toml`.
- [ ] Session start builds the correct command for chat mode, supports per-session overrides, and sets tmux env vars needed for resume/restart.
- [ ] Resume flow works: agent-deck can reattach to an existing Copilot conversation after restart or crash (using Copilot’s session identifier/listing or cache files); resume flag/args are passed when available.
- [ ] Status detection matches other tools: prompt/busy/spinner patterns captured; transitions drive UI state and tests in `tmux`/`patterns` packages.
- [ ] Preflight errors are user-friendly: missing `gh`, missing `gh-copilot` extension, or unauthenticated GitHub session surface actionable toasts/tooltips.
- [ ] Storage/state additions (session ID + detected timestamps) are persisted and shown in session details alongside other tools.
- [ ] Unit tests cover command construction, resume argument logic, status pattern detection, and config parsing; manual smoke steps documented.

## Non-Goals

- Building a Copilot API proxy or replacing GitHub CLI auth.  
- Extending Copilot features beyond what the official CLI exposes (e.g., file editing helpers).  
- Adding new MCPs or altering existing MCP pooling behavior.

## Context & Constraints

- Copilot CLI is a `gh` extension; users may not have it installed or authenticated.  
- Chat sessions are stored by the extension (location/format may differ across versions); resume support must tolerate format drift.  
- Output is primarily TTY text, not structured JSON; status must be inferred from streams similar to OpenCode/Codex.  
- Copilot may stream partial tokens; busy/prompt detection must avoid flicker and false-ready states.

## Approaches Considered

1) **First-class built-in tool (Selected):** Mirror the pattern used for Codex/OpenCode: dedicated config struct, command builder, status patterns, persisted session metadata, UI wiring.  
2) **Custom tool template only:** Rely on user-defined `[tools.copilot]` TOML entry. Rejected because it lacks resume/state, status integration, and install/auth guidance.  
3) **HTTP proxy to GitHub APIs:** Would bypass `gh` but duplicates auth/quotas and is higher risk. Rejected as out of scope.

## Design

### Tool surface & configuration
- Add `CopilotSettings` in `userconfig.go`: `command` (default `gh copilot`), `profile` (optional `gh` profile), `mode`/`default_model` pass-through flags, `env_file`, inline `env`, and `dangerous_mode`-style bypass flag only if Copilot adds an approvals gate.  
- Extend sample `config.toml` with `[copilot]` and `tools.copilot` blocks, including busy/prompt pattern overrides.  
- Add `IconCopilot` and description strings for menus, Setup Wizard, and settings; update default tool mapping.

### Command building & session lifecycle
- New `CopilotOptions` struct + JSON (similar to `CodexOptions`) for per-session overrides (profile, model/mode, optional context file).  
- `Instance.buildCopilotCommand` composes: base command + profile flag (`--profile`), mode/model flags, resume flag when session ID known, and working directory env.  
- Introduce `CopilotSessionID` fields in `Instance`, `storage`, and migrations; track detection timestamp for UI.  
- Start flow: after tmux pane spawn, kick off async detection of Copilot session metadata via `gh copilot sessions list --json` (if available) or parsing Copilot CLI cache files; set tmux env (e.g., `COPILOT_SESSION_ID`) when found.  
- Restart flow: if session ID exists, respawn with resume args; otherwise start fresh and re-detect.

### Status detection
- Add default patterns to `tmux` detector: busy examples (`"Thinking..."`, `"Working on it"`, `"Generating..."`), prompt markers (Copilot prompt line, `?` input, `[Ctrl+C] to quit`), spinner characters if present.  
- Mirror Codex/OpenCode tests in `internal/tmux/patterns_test.go` to cover Copilot patterns and avoid collisions with generic `>` prompts.  
- If Copilot emits structured status (e.g., `{"event":"done"}`) in debug logs, optionally hook a lightweight parser in `parseCopilotOutput`.

### Preflight & validation
- Add install/auth checks in session start:  
  - `gh --version` and `gh extension list` contains `github/gh-copilot` (or `gh copilot --help` success).  
  - `gh auth status --check` (or `--show-token`) to confirm login; surface message if missing scopes.  
- UI toasts for missing prerequisites; option to open GH auth instructions from Settings.

### UI wiring
- New session dialog: Copilot card with icon and description; profile/model overrides shown when tool selected.  
- Settings panel: `[copilot]` toggles/fields; default tool radio includes Copilot.  
- Session details view: show Copilot session ID + detected-at timestamps similar to Codex.  
- New session picker/state search includes Copilot tool filtering.

### Testing & rollout
- Unit tests: config parsing defaults, command builder (with/without resume), status pattern detection, state persistence migration.  
- Manual smoke script: install extension, `gh auth login`, start Copilot session from agent-deck, verify busy→prompt transitions, restart app, confirm resume works.  
- Feature flag via config (`[copilot].enabled` or UI toggle) for initial rollout if needed.

## Risks & Open Questions

- Copilot session storage format may change; need abstraction to read session metadata defensively.  
- Resume flag/command shape is version-dependent—confirm via `gh copilot sessions list` or equivalent; fall back to “start new chat” if unavailable.  
- Output patterns could be localized; consider making busy/prompt patterns configurable in `config.toml`.  
- Performance: repeated `gh` invocations for detection should be cached/debounced to avoid slowing startup.

## Incremental Delivery Plan

1) Add config + UI surface (icon, menu entries, Setup Wizard) with default command only.  
2) Implement command builder/options + storage fields (no resume yet) with tests.  
3) Add session detection/resume using CLI session metadata; add migrations and tmux env sync.  
4) Wire status detection patterns + tests; refine after manual smoke.  
5) Preflight checks + user-facing error strings.  
6) Documentation updates (README, sample config, troubleshooting) and enable by default.
