# Parker — History

## Project Context

Agent Deck is a Go CLI/TUI tool for managing multiple AI coding agent sessions via tmux. Built with charmbracelet (bubbletea/bubbles/lipgloss), SQLite (modernc.org/sqlite), and WebSockets. Currently adding GitHub Copilot CLI as a first-class tool alongside Claude Code, Gemini CLI, and OpenCode.

**User:** Matthew Corven
**Module:** `github.com/asheshgoplani/agent-deck`
**Go version:** 1.24+

## Copilot CLI Integration

Design docs at `docs/plans/copilot-cli/` — 7 phases (0–6).
Binary: `copilot` (standalone, `brew install copilot-cli@prerelease` or `npm install -g @github/copilot`)

## Learnings

<!-- Append Copilot integration patterns, decisions, and file paths below -->

- **2026-02-24:** Updated `docs/plans/copilot-cli/phase-0-live-capture.md` with additional captures per Ripley's recommendations: CLI metadata section (2a: --help, --version, ~/.copilot/ dir structure), MCP server output and pane title rows in the states table, dual-mode capture (plain text + ANSI escapes), and expanded commit/exit criteria. These captures feed Phase 2 (command builder flags), Phase 3 (session ID detection), and status detection patterns.
