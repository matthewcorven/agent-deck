# Decisions

> Canonical decision ledger for the Agent Deck squad. Append-only.

<!-- Decisions are appended below by Scribe after merging from .squad/decisions/inbox/ -->

---

### 2026-02-23: No Formal PRD Exists — Design Docs Are Sufficient

**Author:** Ripley (Lead) · **Status:** Observation (no action required)

Searched entire repo for PRD-level documentation. No formal PRD, ARCHITECTURE.md, or ADR directory exists. Architectural guidance is distributed across README.md, `docs/plans/*.md`, `.claude/plans/DECISIONS.md`, and CHANGELOG.md. The Copilot CLI phased integration plan is fully aligned with established patterns. No changes needed.

---

### 2026-02-24: Phase 0 Doc Expanded with Ripley's Recommended Captures

**Author:** Parker (Integration Dev) · **Status:** Completed

Added 6 new capture items to `docs/plans/copilot-cli/phase-0-live-capture.md`: `--help`, `--version`, `~/.copilot/` directory listing, MCP server TUI output, tmux pane title, and dual-mode capture (plain text + ANSI). These feed Phases 2–4 and reduce rework risk.

---

### 2026-02-24: Upstream Review — 51 Commits (v0.19.2 → v0.19.14)

**Author:** Ripley (Lead) · **Status:** Action required — merge upstream

Reviewed 59 files, +5873/−807 lines. Key changes: `expandHomePath()` → `ExpandPath()`, `resolveEnvFilePath()` → `resolvePath()`, `expandTilde()` removed, `SetupConductor()` gained 6th param, new `GetLastResponseBestEffort()`, new `CapturePaneFresh()`, new `resolveSessionCommand()` pipeline, new transition daemon/notifier system, `sendMessageWhenReady()` accepts `"idle"` state. Merge recommendation: `git merge upstream/main` immediately. Phase docs 0–4 need updates for renamed APIs.

---

### 2026-02-24: Copilot CLI vs SDK/ACP — CLI for v1, ACP Deferred

**Author:** Parker (Integration Dev) · **Reviewed by:** Ripley (Lead) · **Status:** Proposed

Use CLI (tmux-based) for v1. Defer ACP to Phase 7+ as alternative session mode. CLI maps 1:1 to existing architecture (~37h effort). ACP requires Go JSON-RPC bidi client, fs/terminal providers, permission UI (~89h effort, 2.4× slower). ACP is Preview-status, protocol v1, SDK v0.14.1 (TypeScript-only). Dual-process hybrid doesn't work — separate session contexts. Full analysis: `docs/plans/copilot-cli/copilot-cli-vs-sdk-analysis.md`.
