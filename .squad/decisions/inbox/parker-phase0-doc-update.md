### 2026-02-24T00:00:00Z: Phase 0 doc expanded with Ripley's recommended captures
**By:** Parker (requested by Matthew Corven, based on Ripley's review)
**What:** Added 6 new capture items to `docs/plans/copilot-cli/phase-0-live-capture.md`:
1. `copilot --help` — validates CLI flags for command builder (Phase 2)
2. `copilot --version` — pins the preview version we test against
3. `~/.copilot/` directory listing — informs session ID detection (Phase 3)
4. MCP server TUI output — affects status patterns and MCP pooling
5. tmux pane title — potential alternative detection signal
6. Dual-mode capture (plain text + ANSI) for every state

Structure: new Task 2a for one-time metadata, new rows in Task 2b states table, dual-mode capture commands, updated commit list and exit criteria.
**Why:** These captures directly feed Phases 2–4 and reduce rework risk — better to capture everything in one live session than discover gaps later.
