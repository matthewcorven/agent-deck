# Copilot CLI vs SDK/ACP — Deep Comparative Analysis

> Author: Parker (Integration Dev)  
> Date: 2026-02-24  
> Requested by: Matthew Corven (via Ripley)

---

## 1. Executive Summary

**Recommendation: CLI-first with structured ACP migration path.**

The CLI approach (`copilot` binary in tmux) should remain the v1 foundation. It maps 1:1 to agent-deck's proven architecture (tmux session management, status scraping, command builders) used by all four existing tools — Claude, Gemini, OpenCode, Codex — and can ship with minimal new abstractions. The ACP/SDK approach (`copilot --acp --stdio`, JSON-RPC structured protocol) offers superior status detection reliability and richer programmatic control, but requires a fundamentally different session management architecture that no existing tool uses, adding significant risk and complexity to the initial integration. A hybrid Phase 7+ enhancement — ACP as a sidecar for status detection while retaining tmux for visual output — represents the highest-value long-term path.

---

## 2. Technology Landscape

### 2a. Copilot CLI (Interactive TUI)

- **Binary:** `copilot` (standalone, `brew install copilot-cli@prerelease` or `npm install -g @github/copilot`)
- **Interface:** Rich TUI in terminal (ink-based), interactive chat with slash commands
- **Session management:** `--resume SESSION-ID`, `--continue`, `/resume`
- **MCP:** Built-in GitHub MCP server + `--additional-mcp-config` for custom servers
- **Approval flows:** Visual prompts ("1. Yes / 2. Yes, and approve…"), `--yolo`/`--allow-all`
- **Status:** Public preview. Flags well-documented. Active development.

### 2b. Copilot Language Server + ACP Mode

- **Binary:** `@github/copilot-language-server` (npm package, also standalone native binaries per platform)
- **Interface:** JSON-RPC 2.0 over stdin/stdout (ACP mode via `--acp`), or LSP mode (via `--stdio`)
- **ACP protocol:** Open standard (`agentclientprotocol.com`) — `session/new`, `session/load`, `session/prompt`, `session/update`, `session/cancel`, `session/request_permission`
- **MCP:** MCP servers passed as params in `session/new` — agent connects directly
- **Status:** ACP mode marked "Preview" in copilot-language-server README. Protocol version 1. SDK at v0.14.1.
- **Adopters:** Zed, JetBrains AI Assistant, Gemini CLI (as ACP agent)
- **Registry:** Listed as "GitHub Copilot 1.430.0" in ACP Registry

### 2c. Copilot Extensions SDK

- **Package:** `copilot-extensions/preview-sdk.js` (JavaScript)
- **Purpose:** Building extensions that run *inside* Copilot Chat (GitHub.com, VS Code). NOT for controlling Copilot programmatically from an external tool.
- **Relevance to agent-deck:** **None.** This is for building server-side extensions that respond to `@mentions` in Copilot Chat. Not applicable to our session-management use case.

### 2d. Clarification of Terms

| Term | What It Is | Relevant? |
|------|-----------|-----------|
| Copilot CLI | Standalone `copilot` binary, interactive TUI | ✅ Primary |
| Copilot Language Server | `@github/copilot-language-server`, LSP + ACP modes | ✅ ACP mode |
| ACP (Agent Client Protocol) | Open standard for editor↔agent communication | ✅ Protocol |
| ACP TypeScript SDK | `@agentclientprotocol/sdk`, client/agent library | ⚠️ TypeScript-only |
| Copilot Extensions SDK | `copilot-extensions/preview-sdk.js` | ❌ Not relevant |
| `copilot --acp --stdio` | Copilot CLI's ACP mode entry point | ✅ ACP entry |

---

## 3. Feature Comparison Matrix

| Feature | CLI (tmux) | ACP/SDK (stdio) | Hybrid |
|---------|-----------|-----------------|--------|
| **Interactive chat UX** | ✅ Full rich TUI | ❌ No visual UI (raw JSON-RPC) | ✅ CLI for display, ACP for control |
| **Session create** | ✅ `copilot` launches | ✅ `session/new` → sessionId | ✅ Both |
| **Session resume** | ✅ `--resume ID` / `--continue` | ✅ `session/load` (if `loadSession` capability) | ✅ Both |
| **Session fork** | ✅ Start new session with context | ⚠️ No native fork; create new + replay | ✅ CLI fork, ACP status |
| **Initial prompt** | ✅ `-i PROMPT` flag | ✅ `session/prompt` message | ✅ Both |
| **Status detection** | ⚠️ Tmux scraping (regex, fragile) | ✅ Structured `session/update` notifications | ✅ ACP status, CLI display |
| **Busy/idle state** | ⚠️ Pattern matching on TUI output | ✅ Start/end of `session/prompt` response | ✅ ACP |
| **Tool call visibility** | ⚠️ Scrape approval prompt text | ✅ `session/update` with `tool_call` events | ✅ ACP |
| **Permission/approval flow** | ⚠️ Send keys to TUI prompts | ✅ `session/request_permission` → structured response | ✅ ACP |
| **Plan visibility** | ⚠️ Scrape plan mode output | ✅ `session/update` with `plan` entries | ✅ ACP |
| **Cancellation** | ⚠️ Send Escape key to pane | ✅ `session/cancel` notification | ✅ ACP |
| **Model selection** | ✅ `--model MODEL` | ⚠️ Depends on agent capabilities negotiation | ✅ CLI flag |
| **Agent profiles** | ✅ `--agent AGENT` | ⚠️ Not in ACP base spec | ✅ CLI flag |
| **Yolo/auto-approve** | ✅ `--yolo` / `--allow-all` | ⚠️ Grant all permission requests programmatically | ✅ CLI flag simpler |
| **MCP injection** | ✅ `--additional-mcp-config JSON` | ✅ `mcpServers` in `session/new` params | ✅ Both |
| **File system access** | ✅ CLI handles internally | ✅ `fs/read_text_file`, `fs/write_text_file` (client provides) | ✅ Both |
| **Terminal access** | ✅ CLI handles internally | ✅ `terminal/create`, `terminal/output` (client provides) | ⚠️ Complex |
| **Auth handling** | ✅ `/login` interactive or `GH_TOKEN` env | ✅ `authenticate` method + `signIn` flow | ✅ Both |
| **Slash commands** | ✅ Full (`/resume`, `/share`, `/delegate`, `/compact`, etc.) | ⚠️ Not equivalent; protocol-level methods only | ✅ CLI |
| **Visual diff output** | ✅ Rendered in TUI | ✅ Structured diff content in updates | ⚠️ Need custom renderer |
| **Share/export** | ✅ `/share`, `--share-gist` | ❌ Not in ACP spec | ✅ CLI |
| **Delegation to coding agent** | ✅ `/delegate` | ❌ Not in ACP spec | ✅ CLI |
| **Config dir override** | ✅ `--config-dir PATH` | ⚠️ Not standard ACP | ✅ CLI |
| **Custom instructions** | ✅ AGENTS.md loaded automatically | ⚠️ Depends on agent implementation | ✅ CLI |

---

## 4. Use Case Fit Matrix

| Use Case | CLI | ACP/SDK | Hybrid | Notes |
|----------|-----|---------|--------|-------|
| **New interactive session** | 🟢 Exact fit | 🔴 No visual UI | 🟢 CLI display + ACP status | Users expect a visual TUI |
| **Resume after crash** | 🟢 `--resume ID` | 🟢 `session/load` | 🟢 Both work | CLI simpler to implement |
| **Fork session** | 🟢 New pane + context | 🟡 Create new + no visual fork | 🟢 CLI fork + ACP tracking | Fork needs visual pane |
| **Status bar updates** | 🟡 Tmux scraping | 🟢 Structured events | 🟢 ACP updates | ACP eliminates scraping |
| **Conductor orchestration** | 🟡 Send keys to pane | 🟢 `session/prompt` API | 🟢 ACP for commands, CLI for display | Conductor benefits from structured API |
| **Web UI remote view** | 🟡 Tmux capture relay | 🟡 Render JSON updates as HTML | 🟡 Both require work | Neither is free |
| **Notify daemon** | 🟡 Parse pane content | 🟢 Subscribe to `session/update` | 🟢 ACP updates | ACP much more reliable |
| **Auto-approve mode** | 🟢 `--yolo` flag | 🟡 Auto-grant permissions programmatically | 🟢 CLI flag simpler | `--yolo` is one flag |
| **MCP pooling integration** | 🟢 `--additional-mcp-config` | 🟢 `mcpServers` in session params | 🟢 Both | ACP gives more control |
| **Session analytics** | 🟡 Scrape `/usage` output | 🟢 Structured response metadata | 🟢 ACP data | ACP more reliable |
| **Multi-session (tmux tabs)** | 🟢 Native tmux pattern | 🟢 ACP supports concurrent sessions | 🟢 Both | CLI leverages existing infra |
| **Log aggregation** | 🟡 Capture pane output | 🟢 All events are structured JSON | 🟢 ACP data | ACP far superior |
| **CI/headless mode** | 🟡 `-p PROMPT` one-shot | 🟢 Full programmatic control | 🟢 ACP | ACP designed for non-interactive |

---

## 5. Architecture Impact Matrix

| Agent-Deck Subsystem | CLI Impact | ACP/SDK Impact |
|---------------------|-----------|----------------|
| **`internal/session/instance.go`** | ✅ Add `buildCopilotCommand` — follows existing pattern exactly | ⚠️ Need new `AcpSession` manager, JSON-RPC client, goroutine lifecycle — new pattern |
| **`internal/session/tooloptions.go`** | ✅ Add `CopilotOptions` — mirrors `ClaudeOptions`/`CodexOptions` | ⚠️ Different options shape (ACP capabilities, not CLI flags) |
| **`internal/session/userconfig.go`** | ✅ Add `CopilotSettings` — standard pattern | ✅ Same config struct, different runtime behavior |
| **`internal/tmux/patterns.go`** | ✅ Add `DefaultRawPatterns("copilot")` — standard pattern | ❌ Not needed — status comes from JSON-RPC events |
| **`internal/tmux/` (session runtime)** | ✅ Tmux IS the session runtime — no changes | ⚠️ Tmux still used for display, but ACP subprocess is separate |
| **`internal/statedb/`** | ✅ Add copilot fields to `tool_data` JSON — standard migration | ⚠️ Same fields, but also need ACP connection state |
| **`internal/mcppool/`** | ✅ Pass `--additional-mcp-config` flag | ✅ Pass `mcpServers` in `session/new` — potentially cleaner |
| **`internal/ui/`** | ✅ Add icon, tool card — standard | ⚠️ Need to render ACP events into UI if no TUI |
| **`cmd/agent-deck/session_cmd.go`** | ✅ Add copilot to tool dispatch — standard | ⚠️ ACP sessions need different start/stop lifecycle |
| **`internal/session/conductor.go`** | ✅ Send keys to tmux pane | ✅ Call `session/prompt` — much cleaner but different interface |
| **`internal/web/`** | ✅ Relay tmux capture — existing pattern | ⚠️ Relay ACP events — different streaming model |
| **`cmd/agent-deck/notify_daemon_cmd.go`** | ✅ Watch pane content — existing pattern | ✅ Subscribe to ACP events — better but different |

---

## 6. Risk Matrix

| Risk Factor | CLI | ACP/SDK | Hybrid |
|-------------|-----|---------|--------|
| **API stability** | 🟡 Medium — CLI flags may change in preview | 🔴 High — ACP marked "Preview", protocol v1, SDK v0.14 | 🟡 CLI shields from protocol changes |
| **Breaking changes on upgrade** | 🟡 TUI output changes break patterns | 🟡 Protocol changes break JSON-RPC client | 🟡 Both risks present |
| **Pattern fragility** | 🔴 High — regex on TUI is inherently fragile | 🟢 Low — structured events are stable | 🟢 ACP for status eliminates fragility |
| **Documentation quality** | 🟢 Well-documented CLI flags and behavior | 🟡 ACP spec good, but Copilot-specific ACP docs thin | 🟡 Mixed |
| **Community/ecosystem** | 🟢 Wide adoption (direct users) | 🟡 Growing (Zed, JetBrains) but small | 🟡 Mixed |
| **Binary availability** | 🟢 Standalone binary, brew/npm install | 🟢 Same binary with `--acp` flag, or npm language-server | 🟢 Both available |
| **Auth complexity** | 🟢 CLI handles interactively | 🟡 Need to implement `signIn`/device flow programmatically | 🟡 CLI handles auth |
| **Go integration** | 🟢 Exec binary, no dependencies | 🔴 ACP SDK is TypeScript-only; need Go JSON-RPC client | 🟡 Go JSON-RPC exists but untested |
| **Maintenance burden** | 🟡 Pattern updates on CLI version bumps | 🟡 Protocol version bumps, capability negotiation | 🟡 Both |
| **Scope creep** | 🟢 Bounded — same pattern as other tools | 🔴 High — ACP requires fs/terminal providers, permission UI, event rendering | 🟡 Scoped hybrid limits exposure |
| **Time to ship v1** | 🟢 Fast — proven pattern, estimated 2-3 weeks | 🔴 Slow — new architecture, estimated 6-8 weeks | 🟡 CLI ships first, ACP later |
| **Copilot CLI deprecation risk** | 🟡 Low — actively developed, public preview | 🟡 Low — language server is core product | 🟢 Both paths covered |

---

## 7. Implementation Effort Matrix

| Phase / Component | CLI Effort | ACP/SDK Effort | Notes |
|-------------------|-----------|----------------|-------|
| **Phase 0: Live Capture** | 4h — run CLI, capture pane output | 2h — capture ACP JSON-RPC messages | ACP messages are self-documenting |
| **Phase 1: Config Surface** | 4h — `CopilotSettings`, UI wiring | 4h — same config, different runtime flags | Identical for config |
| **Phase 2: Command Builder** | 8h — `buildCopilotCommand`, flags, options | 20h — JSON-RPC client, ACP init, capability negotiation, session/new | ACP needs Go JSON-RPC client from scratch |
| **Phase 3: Session Detection** | 6h — session ID from files/pane/env | 2h — `session/new` returns sessionId directly | ACP trivially gives session ID |
| **Phase 4: Status Detection** | 8h — regex patterns, testing | 2h — parse `session/update` events | ACP is dramatically simpler |
| **Phase 5: Preflight** | 3h — binary check, error UX | 3h — binary check (same binary, `--acp` flag) | Identical |
| **Phase 6: Docs & Polish** | 4h — standard | 8h — more to document (new architecture) | ACP has more surface area |
| **ACP client implementation** | N/A | 16h — Go JSON-RPC bidi client, event dispatcher, goroutine management | New infrastructure |
| **fs/terminal provider** | N/A | 12h — implement `fs/read_text_file`, `fs/write_text_file`, `terminal/create`, `terminal/output` | ACP requires client to provide these |
| **Permission UI** | N/A | 8h — handle `session/request_permission`, route to user | New approval flow UI |
| **Event→UI rendering** | N/A | 12h — render `session/update` events as visual output | Replace TUI with custom renderer |
| **TOTAL** | **~37h** | **~89h** | **CLI is ~2.4× faster to ship** |

---

## 8. DX & Maturity Matrix

| Quality Dimension | CLI | ACP/SDK |
|-------------------|-----|---------|
| **Maturity** | 🟡 Public preview, actively developed | 🔴 Preview, protocol v1, SDK v0.14.1 |
| **Documentation** | 🟢 Comprehensive flag docs, `/help` built-in | 🟡 Good protocol spec, thin Copilot-specific ACP docs |
| **Stability** | 🟡 TUI may change between versions | 🟡 Protocol may change (v1 → v2) |
| **Go support** | 🟢 Shell exec — trivial | 🔴 No Go SDK; need custom JSON-RPC client |
| **TypeScript support** | N/A (not relevant) | 🟢 Official `@agentclientprotocol/sdk` |
| **Error messages** | 🟢 User-facing in TUI | 🟡 JSON error objects need parsing |
| **Debugging** | 🟢 Visual — look at the pane | 🟡 JSON-RPC trace logs |
| **Testing** | 🟡 Mock tmux capture output | 🟢 Mock JSON-RPC messages — deterministic |
| **Upgrade path** | 🟡 Test patterns against new CLI version | 🟡 Test protocol compatibility |
| **Community examples** | 🟢 Many CLI users and examples | 🟡 Zed's integration is primary reference |

---

## 9. Hybrid Approach Analysis

### 9a. Hybrid Architecture: CLI Display + ACP Sidecar

The most promising hybrid approach: run the Copilot CLI in tmux for visual display (the user sees the full TUI), AND spawn a parallel `copilot --acp --stdio` process as a sidecar for structured status detection and programmatic control.

```
┌─────────────────────────────────────────────┐
│  agent-deck                                  │
│  ┌──────────────┐  ┌──────────────────────┐ │
│  │  tmux pane    │  │  ACP sidecar         │ │
│  │  copilot      │  │  copilot --acp       │ │
│  │  (visual TUI) │  │  --stdio             │ │
│  │               │  │  (JSON-RPC events)   │ │
│  └──────┬───────┘  └──────────┬───────────┘ │
│         │ user sees            │ structured   │
│         │ rich output          │ status data  │
│         ▼                      ▼              │
│    tmux capture          event dispatcher     │
│    (fallback only)       (primary source)     │
└─────────────────────────────────────────────┘
```

**Problem:** Two separate Copilot processes for one session = two separate sessions, two auth contexts, two model contexts. They don't share conversation state. This makes the sidecar approach fundamentally limited — the ACP sidecar can't observe or control the TUI session's state.

### 9b. Alternative Hybrid: CLI for v1, ACP for v2

More practical: ship v1 with CLI approach (Phases 0-6), then add ACP as an **alternative session mode** in v2:

- `session.copilot_mode = "cli"` (default, tmux-based) — existing user experience
- `session.copilot_mode = "acp"` (opt-in, structured) — for programmatic use cases (conductor, headless, CI)

This avoids trying to combine two incompatible process models and lets users choose.

### 9c. Hybrid Verdict

The "dual-process sidecar" pattern doesn't work because ACP and CLI are separate session contexts. The **phased migration** (CLI now, ACP later as alternative mode) is the viable hybrid strategy.

---

## 10. Use Cases Deep-Dive: CLI vs ACP

| Use Case | Best Approach | Rationale |
|----------|---------------|-----------|
| Interactive coding session | **CLI** | Users expect visual TUI, approval prompts, rich diff rendering |
| Fork session to worktree | **CLI** | Fork needs visual pane; `--resume` in new tmux pane |
| Conductor sends prompts | **ACP** (future) | `session/prompt` is cleaner than `tmux send-keys` |
| Status bar busy/idle | **CLI** (adequate), **ACP** (ideal) | CLI works with patterns; ACP eliminates false positives |
| Notify on completion | **CLI** (adequate), **ACP** (ideal) | ACP's `stopReason: end_turn` is definitive |
| Web UI streaming | **CLI** (via tmux capture) | tmux capture works today; ACP would need custom HTML renderer |
| Headless/CI agent | **ACP** | No visual output needed; `session/prompt` + `session/update` is ideal |
| Session analytics | **ACP** | Structured data (token counts, tool calls, timing) |
| MCP server injection | **Both** | CLI: `--additional-mcp-config`; ACP: `mcpServers` in `session/new` |
| Log aggregation | **ACP** | Every event is structured JSON |

---

## 11. What ACP Requires That CLI Doesn't

If we chose ACP as the primary approach, agent-deck must implement these **Client-side responsibilities** that the Copilot CLI normally handles internally:

1. **File system provider** — implement `fs/read_text_file` and `fs/write_text_file` methods so the agent can read/write project files
2. **Terminal provider** — implement `terminal/create` and `terminal/output` so the agent can run shell commands
3. **Permission UI** — implement `session/request_permission` handler to show approval dialogs to users
4. **Content renderer** — render `session/update` events (markdown chunks, diffs, tool call status, plans) as visual output
5. **Authentication flow** — implement `signIn` device flow (user code → browser → callback)
6. **Go JSON-RPC 2.0 bidirectional client** — no official Go SDK exists; need custom or adopt `github.com/sourcegraph/jsonrpc2`

This is effectively building a mini-IDE's worth of editor capabilities that tmux + the CLI binary already provide for free.

---

## 12. Recommendation

### Primary: CLI approach for v1 (Phases 0-6)

**Rationale:**
- Maps exactly to the proven architecture used by all 4 existing tools
- ~37 hours estimated effort vs ~89 hours for ACP
- Zero new infrastructure (no JSON-RPC client, no fs/terminal providers, no permission UI)
- Users get the full, polished Copilot TUI experience
- Session resume, MCP injection, model selection all work via well-documented flags
- Status detection via tmux patterns is fragile but proven — Claude, Codex, Gemini, OpenCode all use this successfully

### Secondary: ACP as Phase 7+ Enhancement

**When to pursue ACP:**
- After v1 ships and is stable
- When Copilot's ACP mode exits "Preview" status
- When a Go ACP SDK exists (or protocol is stable enough to justify building one)
- Triggered by specific use cases: conductor automation, headless CI mode, reliable status detection

**ACP scope (future):**
- Add `session.copilot_mode = "acp"` as opt-in alternative
- ACP mode for programmatic-only use cases (conductor, CI, API)
- CLI mode remains default for interactive users

### What NOT to do:
- ❌ Don't build ACP as the v1 foundation — too much new infrastructure, no visual UI
- ❌ Don't try the dual-process hybrid — separate session contexts make it unworkable
- ❌ Don't use Copilot Extensions SDK — not relevant to our use case
- ❌ Don't block v1 on ACP — CLI approach is sufficient and proven

### Decision Summary

| Aspect | Decision |
|--------|----------|
| v1 approach | CLI (tmux-based) |
| v2 enhancement | ACP as alternative session mode |
| Timeline impact | v1 ships ~2.4× faster with CLI |
| Architecture impact | Zero new subsystems for v1 |
| Risk profile | Lower for CLI (proven pattern) |
| Feature parity | CLI covers 100% of v1 acceptance criteria |
| ACP trigger | Post-v1 stability + ACP exits preview |

---

## Appendix A: ACP Protocol Reference (Relevant Methods)

| Method | Direction | Purpose |
|--------|-----------|---------|
| `initialize` | Client → Agent | Negotiate versions, exchange capabilities |
| `authenticate` | Client → Agent | Auth if required |
| `session/new` | Client → Agent | Create session (returns sessionId) |
| `session/load` | Client → Agent | Resume existing session |
| `session/prompt` | Client → Agent | Send user message |
| `session/update` | Agent → Client | Progress notifications (message chunks, tool calls, plans) |
| `session/cancel` | Client → Agent | Cancel ongoing operations |
| `session/request_permission` | Agent → Client | Request user authorization for tool calls |
| `session/set_mode` | Client → Agent | Switch operating modes |
| `fs/read_text_file` | Agent → Client | Read file from workspace |
| `fs/write_text_file` | Agent → Client | Write file to workspace |
| `terminal/create` | Agent → Client | Create terminal for command execution |
| `terminal/output` | Agent → Client | Get terminal output |

## Appendix B: Copilot CLI Flag Reference (Key Flags for agent-deck)

| Flag | Purpose | Used in Phase |
|------|---------|---------------|
| `copilot` | Launch interactive session | 2 |
| `--resume SESSION-ID` | Resume specific session | 2, 3 |
| `--continue` | Resume most recent session | 2, 3 |
| `-i PROMPT` | Auto-execute first prompt | 2 |
| `--model MODEL` | Set AI model | 2 |
| `--agent AGENT` | Use custom agent profile | 2 |
| `--yolo` / `--allow-all` | Skip all approvals | 2 |
| `--config-dir PATH` | Custom config directory | 2 |
| `--additional-mcp-config JSON` | Inject MCP servers | Future |
| `--acp --stdio` | Start in ACP mode | 7+ |
| `--share` / `--share-gist` | Export session | Future |

## Appendix C: Existing Tool Architecture Pattern

All tools follow this pattern — Copilot CLI integration (v1) follows it exactly:

```
UserConfig          →  XSettings struct (config.toml parsing)
ToolOptions         →  XOptions struct (per-session overrides)
Instance            →  buildXCommand() (flag composition)
tmux/patterns.go    →  DefaultRawPatterns("x") (status detection)
statedb             →  tool_data JSON blob (session ID, metadata)
session lifecycle   →  Start → Detect session ID → Resume/Restart
```

ACP would require a parallel track alongside this pattern, not a replacement of it, because all other tools would still use tmux-based management.

---

## Lead Review — Ripley

> Reviewer: Ripley (Lead / Architecture)  
> Date: 2026-02-24  
> Verdict: **Approved — CLI for v1, ACP deferred. One interface-design action required.**

### 1. Coverage Assessment

The analysis is thorough and well-structured. The feature comparison matrix, use-case fit matrix, and architecture impact matrix give a clear picture from multiple angles. Appendix C (existing tool pattern) is a particularly useful anchor — it makes the cost argument concrete.

**Blind spots to note:**

- **Session ID extraction risk is understated.** This is the #1 open risk from my Phase 0 review. The CLI approach requires scraping or discovering the session ID from files/pane/env — and we don't yet know if Copilot CLI exposes it in a detectable way. The analysis acknowledges ACP trivially returns it (Section 7, Phase 3: 2h vs 6h) but doesn't call out that session ID detection failure would block resume/restart functionality entirely. This needs to be a Phase 0 gating question.
- **Test maintainability.** The DX matrix mentions testing briefly but doesn't address the long-term cost: CLI patterns require curated pane-capture fixtures that break on any TUI change. ACP mocks are deterministic JSON. Over 12+ months of Copilot CLI version bumps, the maintenance delta is significant.
- **Upstream infrastructure.** The recent v0.19.2→v0.19.14 merge added `CapturePaneFresh()` (bypass-cache tmux capture) and `GetLastResponseBestEffort()` (multi-fallback response retrieval) — both directly help CLI status detection reliability. These should be called out as mitigants to the pattern-fragility risk.

### 2. Recommendation Soundness

**Agreed.** CLI for v1 is the correct call. The reasoning is architecturally sound:

- Zero new subsystems. Every other tool uses the same pattern. Introducing a parallel JSON-RPC lifecycle model for one tool would fragment the codebase.
- The effort asymmetry (37h vs 89h) is likely *understated* for ACP. Building a Go bidirectional JSON-RPC 2.0 client where the *agent* calls methods on the *client* (fs/terminal providers) is significantly harder than a standard request-response client. The 16h estimate for the ACP client is optimistic — budget 24–30h realistically.
- ACP's "Preview" status with protocol v1 and SDK v0.14.1 makes it a moving target. Building a foundation on it today risks rework when the protocol stabilizes.

### 3. Risk Assessment Accuracy

Mostly accurate. Specific notes:

- **Pattern fragility (CLI: 🔴 High)** — Honest and correct. The upstream `CapturePaneFresh()` mitigates somewhat, but regex-on-TUI is inherently brittle. Accept this as known technical debt.
- **Go integration (ACP: 🔴)** — Understated. It's not just "no Go SDK" — it's that ACP requires the client to *serve* methods (`fs/read_text_file`, `terminal/create`). This is a bidirectional protocol where agent-deck becomes both client and server on the same connection. Significantly harder than the analysis implies.
- **Scope creep (ACP: 🔴 High)** — Accurate. The "mini-IDE" framing in Section 11 is the right way to communicate this risk.
- **Copilot CLI deprecation risk (both: 🟡 Low)** — Agree. GitHub is investing heavily in both paths; neither is getting abandoned near-term.

### 4. Interface Design for Future ACP Transition

The phased migration (CLI now, ACP as alternative mode later) is architecturally sound. However, we should bake in **one abstraction now** to avoid a painful retrofit:

**Action: Define a `StatusProvider` interface in `internal/session/`.**

```go
type StatusProvider interface {
    Status() ToolStatus           // busy, idle, prompting, error
    LastActivity() time.Time
    SessionID() (string, bool)    // (id, detected)
}
```

The CLI implementation wraps tmux pattern matching. A future ACP implementation wraps event subscription. The UI, conductor, notify daemon, and web relay consume this interface instead of reaching into tmux directly. This is a small change (~30 lines) that prevents the largest coupling risk for the ACP transition.

Without this, every consumer of status data (UI status bar, conductor orchestration, notify daemon, web streaming) would need to be touched when ACP arrives. With it, ACP becomes a drop-in `StatusProvider` implementation.

**This is the only design change I'm requesting for v1.**

### 5. Devil's Advocate: Why Not ACP First?

The strongest arguments for starting with ACP:

1. **Session ID is free.** `session/new` returns it. No scraping, no guessing, no Phase 0 capture dependency.
2. **Conductor becomes dramatically cleaner.** `session/prompt` API vs `tmux send-keys` with timing heuristics.
3. **Pattern fragility vanishes.** The single largest maintenance burden for all tools would not exist for Copilot.
4. **Structured analytics from day one.** Token counts, tool call traces, timing — all in JSON.

**Why these don't override the recommendation:**

- Points 1–4 are real benefits, but they come with ~90h of new infrastructure that benefits exactly one tool. The other four tools remain tmux-based regardless.
- Building a Go bidirectional JSON-RPC client + fs/terminal providers is effectively building an editor backend. That's a product, not a feature.
- ACP Preview status means we'd be building on a protocol that could change under us.
- The `StatusProvider` interface (above) preserves these benefits for v2 without paying the cost now.

### 6. Summary

Parker's analysis is solid work — comprehensive, well-structured, and the recommendation is correct. CLI for v1, ACP deferred to Phase 7+. The one addition I'm requesting is a `StatusProvider` interface to future-proof the ACP transition path. Session ID detection remains the gating risk for Phase 0 and should be treated as a hard blocker until resolved.

**Decision: Approved. Proceed with CLI approach (Phases 0–6). Implement `StatusProvider` interface in Phase 1.**
