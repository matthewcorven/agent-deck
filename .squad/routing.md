# Routing Table

> Maps work signals to the right agent.

| Signal Pattern | Route To | Reason |
|---------------|----------|--------|
| Architecture, design decisions, code review, integration strategy | Ripley | Lead owns structural decisions |
| tmux, session lifecycle, instance.go, MCP pooling, statedb, platform | Dallas | Systems Dev owns core plumbing |
| Copilot CLI, tooloptions, command builder, status detection, preflight | Parker | Integration Dev owns Copilot work |
| Tests, test files, quality, edge cases, parity checks | Lambert | Tester owns quality |
| `internal/session/` (general) | Dallas + Parker | Both touch session code — coordinate |
| `internal/tmux/patterns.go` | Dallas (patterns) or Parker (copilot case) | Shared file — scope determines owner |
| `internal/ui/styles.go` | Parker (copilot icons) or Dallas (general) | Scope determines owner |
| `internal/statedb/` | Dallas | Systems Dev owns persistence |
| `cmd/agent-deck/` (CLI commands) | Dallas (existing) or Parker (copilot-specific) | Scope determines owner |
| `internal/mcppool/` | Dallas | Systems Dev owns MCP pooling |
| `internal/web/` | Dallas | Systems Dev owns web layer |
| `docs/plans/copilot-cli/` | Parker | Integration Dev owns Copilot design docs |
| Config, TOML, userconfig | Dallas (general) or Parker (CopilotSettings) | Scope determines owner |
| Multi-domain / "Team" request | Ripley + relevant agents | Lead coordinates |

## Escalation

- If uncertain, route to Ripley for triage.
- If a task crosses Dallas + Parker boundaries, Ripley reviews the interface.
