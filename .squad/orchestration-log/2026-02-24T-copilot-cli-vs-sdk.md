# Orchestration Log — Parker + Ripley: Copilot CLI vs SDK/ACP Analysis

**Timestamp:** 2026-02-24
**Agents:** Parker (Integration Dev), Ripley (Lead)
**Requested by:** Matthew Corven (via Ripley)

---

## Parker (Integration Dev)

### Routing

- **Why chosen:** Parker is the Integration Dev — responsible for deep technical analysis of tool integration approaches. CLI vs SDK comparison requires hands-on research into ACP protocol, Copilot language server, and Copilot Extensions SDK.
- **Mode:** sync (VS Code subagent)

### Input Artifacts

- `docs/plans/2026-02-17-copilot-cli-support.md` (parent design doc)
- `docs/plans/copilot-cli/phase-0-*.md` through `phase-6-*.md` (phased implementation plan)
- ACP protocol spec (`agentclientprotocol.com`)
- Copilot CLI docs, `copilot-language-server` README
- `copilot-extensions/preview-sdk.js` (evaluated, found not relevant)

### Outcome

Parker produced an 8-matrix deep comparative analysis covering:
1. Executive summary with CLI-first recommendation
2. Technology landscape (CLI, ACP/Language Server, Extensions SDK — with disambiguation)
3. Feature comparison (28 features across CLI, ACP, Hybrid)
4. Use case fit (10 use cases)
5. Architecture impact (12 subsystems)
6. Risk matrix (12 risk factors)
7. Implementation effort (~37h CLI vs ~89h ACP — 2.4× delta)
8. DX & maturity comparison

Key finding: dual-process hybrid (CLI display + ACP sidecar) doesn't work — separate session contexts.

Recommendation: CLI for v1, ACP deferred to Phase 7+ when protocol exits preview.

### Files Produced

- `docs/plans/copilot-cli/copilot-cli-vs-sdk-analysis.md` (full analysis)
- `.squad/decisions/inbox/parker-copilot-cli-vs-sdk.md` (decision record)

---

## Ripley (Lead)

### Routing

- **Why chosen:** Ripley is the Lead — responsible for architectural review, scope gating, and decision approval. Parker's analysis requires Lead-level validation before proceeding.
- **Mode:** sync (VS Code subagent)

### Input Artifacts

- `docs/plans/copilot-cli/copilot-cli-vs-sdk-analysis.md` (Parker's analysis)
- `.squad/decisions/inbox/parker-copilot-cli-vs-sdk.md` (proposed decision)

### Outcome

Ripley reviewed Parker's analysis and appended architectural review comments to the analysis file. Review validated the CLI-first recommendation and confirmed ACP deferral rationale.

### Files Produced

- Appended review to `docs/plans/copilot-cli/copilot-cli-vs-sdk-analysis.md`
