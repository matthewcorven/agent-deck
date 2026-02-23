# Agent Deck — Squad Team

## Project Context

**Project:** Agent Deck — AI agent command center (CLI/TUI)
**Stack:** Go 1.24+, tmux, charmbracelet (bubbletea/bubbles/lipgloss), SQLite (modernc.org/sqlite), WebSockets (gorilla), fsnotify, web push
**Module:** `github.com/asheshgoplani/agent-deck`
**User:** Matthew Corven
**Description:** Terminal-based mission control for managing multiple AI coding agent sessions (Claude Code, Gemini CLI, OpenCode, Codex). Features: session management, MCP socket pooling, git worktrees, fork sessions, status detection, conductors, web UI. Currently adding GitHub Copilot CLI as a first-class tool.

## Members

| Name | Role | Specialization | Emoji |
|------|------|----------------|-------|
| Ripley | Lead | Architecture, code review, integration design | 🏗️ |
| Dallas | Systems Dev | Go core: tmux, session management, MCP pooling, statedb | 🔧 |
| Parker | Integration Dev | Copilot CLI phases, command builder, status detection, tool parity | ⚛️ |
| Lambert | Tester | Go tests, edge cases, cross-tool parity checks | 🧪 |
| Scribe | (silent) | Memory, decisions, session logs | 📋 |
| Ralph | Work Monitor | — | 🔄 |

## Key Architecture

- **Entry point:** `cmd/agent-deck/main.go`
- **Session lifecycle:** `internal/session/` (instance, config, discovery, env, hooks)
- **Tool options:** `internal/session/tooloptions.go` (per-tool options structs)
- **User config:** `internal/session/userconfig.go` (TOML config at `~/.agent-deck/config.toml`)
- **tmux integration:** `internal/tmux/` (patterns, status detection)
- **State DB:** `internal/statedb/` (SQLite persistence)
- **TUI:** `internal/ui/` (charmbracelet-based)
- **MCP pooling:** `internal/mcppool/` (socket proxy, HTTP server)
- **Web UI:** `internal/web/`
- **CLI commands:** `cmd/agent-deck/` (add, session, mcp, web, worktree, etc.)

## Upcoming Work

GitHub Copilot CLI integration — 7 phases:
0. Live Capture (tmux output)
1. Config Surface (selectable tool)
2. Command Builder + Storage
3. Session Detection + Resume
4. Status Detection
5. Preflight Checks
6. Docs + Polish

Design docs: `docs/plans/copilot-cli/`
