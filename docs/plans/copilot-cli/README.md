# GitHub Copilot CLI Integration — Implementation Guide

> Parent design doc: [../2026-02-17-copilot-cli-support.md](../2026-02-17-copilot-cli-support.md)

## Overview

Add `copilot` (GitHub Copilot CLI) as a first-class agent-deck tool with parity to Claude, Gemini, OpenCode, and Codex: selectable in New Session / Fork / Setup Wizard, persisted session tracking/resume, status detection, configuration, and preflight checks.

Binary: `copilot` (standalone, `brew install copilot-cli` or `npm install -g @github/copilot`)

## Non-Goals

- Building a Copilot API proxy or bypassing CLI auth.
- Extending Copilot CLI features beyond what it exposes.
- Managing Copilot's MCP configuration (users use `~/.copilot/mcp-config.json`).
- ACP protocol integration (deferred to future enhancement).

## Phase Dependency Graph

```
Phase 0 (Live Capture)
  └─▶ Phase 1 (Config Surface)
        └─▶ Phase 2 (Command Builder + Storage)
              ├─▶ Phase 3 (Session Detection + Resume)
              └─▶ Phase 4 (Status Detection)  ← also depends on Phase 0
                    └─▶ Phase 5 (Preflight Checks)
                          └─▶ Phase 6 (Docs + Polish)
```

## Phase Index

| Phase | Doc | Goal | Depends On |
|-------|-----|------|------------|
| 0 | [phase-0-live-capture.md](phase-0-live-capture.md) | Capture Copilot CLI terminal output in tmux | — |
| 1 | [phase-1-config-surface.md](phase-1-config-surface.md) | Copilot appears as a selectable tool in UI/config | Phase 0 |
| 2 | [phase-2-command-builder.md](phase-2-command-builder.md) | Command construction, options struct, storage fields | Phase 1 |
| 3 | [phase-3-session-detection.md](phase-3-session-detection.md) | Detect session ID + resume across restarts | Phase 2 |
| 4 | [phase-4-status-detection.md](phase-4-status-detection.md) | Busy/prompt status detection from tmux content | Phase 0, Phase 2 |
| 5 | [phase-5-preflight.md](phase-5-preflight.md) | Missing binary error UX with install guidance | Phase 4 |
| 6 | [phase-6-docs-polish.md](phase-6-docs-polish.md) | Documentation, enable by default, smoke checklist | Phase 5 |

## Key File References

These files are touched repeatedly across phases. Each phase doc lists exactly which files it modifies.

| File | What lives here |
|------|-----------------|
| `internal/session/userconfig.go` | `CopilotSettings` struct, `UserConfig.Copilot` field, sample config |
| `internal/session/tooloptions.go` | `CopilotOptions` struct, marshal/unmarshal, `NewCopilotOptions` |
| `internal/session/instance.go` | `Instance` Copilot fields, `buildCopilotCommand`, Start/Restart dispatch |
| `internal/tmux/patterns.go` | `DefaultRawPatterns("copilot")` case |
| `internal/ui/styles.go` | `IconCopilot` constant, `ToolIcon`/`ToolColor` switch cases |
| `internal/statedb/statedb.go` | `tool_data` JSON column (Copilot fields stored here) |
| `internal/statedb/migrate.go` | `jsonInstanceData` Copilot fields (JSON migration struct) |

## Copilot CLI Command Reference (Quick)

| Flag | Purpose |
|------|---------|
| `copilot` | Launch interactive session |
| `--resume [SESSION-ID]` | Resume a specific session |
| `--continue` | Resume most recent session |
| `-i PROMPT` | Interactive + auto-execute first prompt |
| `--model MODEL` | Set AI model |
| `--agent AGENT` | Use custom agent profile |
| `--allow-all` / `--yolo` | Skip all approval prompts |
| `--config-dir PATH` | Custom config directory |
| `--additional-mcp-config JSON` | Inject MCP servers |
