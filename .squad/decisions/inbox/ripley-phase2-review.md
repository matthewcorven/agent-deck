### 2026-02-28: Phase 2 Design Doc Requires 7 Amendments Before Implementation

**By:** Ripley (Lead)
**Status:** Action required — Parker (amend doc), then implementation proceeds
**What:** Verified Phase 2 command builder doc against current codebase (post-Phase 1, post-upstream merge). Found 7 discrepancies: stale line numbers, 6 missing dispatch points (UpdateStatus hooks, PostStartSync, CanRestart, CanFork, Instance restore), underscoped storage pipeline (10+ insertion sites across 4 files not 4 simple edits), over-engineered `buildCopilotExtraFlags`, missing `buildCopilotCommandWithMessage` decision, and minor style issues. CopilotSettings from Phase 1 is confirmed correct. OpenCode is the reference pattern for implementation.
**Why:** Implementing from a stale doc creates merge conflicts, missed wiring, and inconsistent patterns. The storage path alone (MarshalToolData/UnmarshalToolData positional params) will silently break compilation if any arg is missed. Fix the doc first, then Parker and Dallas execute cleanly.
