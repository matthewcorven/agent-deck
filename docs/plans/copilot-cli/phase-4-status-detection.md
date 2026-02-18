# Phase 4 — Status Detection

> **Goal:** agent-deck's status bar accurately reflects Copilot's busy/prompt/approval state.  
> **Depends on:** Phase 0 (captured terminal content), Phase 2 (tool wired into Start).  
> **Estimated scope:** 1 file modified + tests.

## Context

Status detection works by scraping the tmux pane content and matching against known patterns.

Each tool registers patterns via `DefaultRawPatterns(toolName)` in `internal/tmux/patterns.go`. Patterns are either:
- **Plain strings:** matched with `strings.Contains`
- **Regex:** prefixed with `"re:"`, compiled and matched with `regexp`

Patterns are split into:
- `BusyPatterns` — tool is processing (spinner, "esc to interrupt", thinking text)
- `PromptPatterns` — tool is idle, waiting for user input
- `SpinnerChars` — characters to strip for stable hashing (optional, Claude-specific)
- `WhimsicalWords` — thinking words combined with spinner chars for combo patterns (optional)

The polling loop in the TUI checks: busy → prompt → fallback. If busy matches, status = "running". If prompt matches, status = "waiting" (idle/ready). If neither, status is inferred from other signals.

## Files to Modify

### 1. `internal/tmux/patterns.go` — Add `case "copilot"`

Add to `DefaultRawPatterns()` switch (after the `"codex"` case, ~line 75):

```go
case "copilot":
    return &RawPatterns{
        BusyPatterns: []string{
            // === FILL FROM PHASE 0 CAPTURES ===
            // Expected candidates (verify against actual terminal output):
            "esc to interrupt",       // Common busy indicator
            // "Thinking",            // Spinner text if visible
            // "re:Tool:.*running",   // Tool execution progress
        },
        PromptPatterns: []string{
            // === FILL FROM PHASE 0 CAPTURES ===
            // Expected candidates (verify against actual terminal output):
            // "copilot>",           // Prompt line (if any)
            // "Type your message",  // Input prompt text
            "1. Yes",                // Approval prompt (first option)
            "Yes, and approve",      // Approval prompt (approve-all option)
        },
    }
```

**IMPORTANT:** The patterns above are placeholders. Replace with actual text observed in Phase 0 captures. Do NOT ship patterns that haven't been verified against a real Copilot CLI session.

### 2. Spinner chars (if needed)

If Copilot uses unique spinner characters (braille dots, asterisks, etc.), add them to:
- `defaultSpinnerChars()` — for detection in busy patterns
- `SpinnerRuneSet()` — for content normalization / stable hashing

Only add if Copilot's spinner characters differ from existing ones.

## Testing Strategy

### `internal/tmux/patterns_test.go`

Add test cases following the existing pattern test style. Each test: compile patterns → run against captured content → assert match/no-match.

```go
func TestCopilotBusyPatterns(t *testing.T) {
    raw := DefaultRawPatterns("copilot")
    if raw == nil {
        t.Fatal("DefaultRawPatterns('copilot') returned nil")
    }
    resolved, err := CompilePatterns(raw)
    if err != nil {
        t.Fatalf("CompilePatterns: %v", err)
    }

    // Paste captured busy terminal content from Phase 0 here
    busyContent := `
    <PASTE ACTUAL BUSY TERMINAL CONTENT FROM PHASE 0 CAPTURES>
    `

    // Should match at least one busy pattern
    matched := false
    for _, s := range resolved.BusyStrings {
        if strings.Contains(busyContent, s) {
            matched = true
            break
        }
    }
    for _, re := range resolved.BusyRegexps {
        if re.MatchString(busyContent) {
            matched = true
            break
        }
    }
    if !matched {
        t.Error("busy content should match at least one busy pattern")
    }
}

func TestCopilotPromptPatterns(t *testing.T) {
    raw := DefaultRawPatterns("copilot")
    resolved, err := CompilePatterns(raw)
    if err != nil {
        t.Fatalf("CompilePatterns: %v", err)
    }

    // Paste captured idle/prompt terminal content from Phase 0 here
    promptContent := `
    <PASTE ACTUAL PROMPT TERMINAL CONTENT FROM PHASE 0 CAPTURES>
    `

    matched := false
    for _, s := range resolved.PromptStrings {
        if strings.Contains(promptContent, s) {
            matched = true
            break
        }
    }
    for _, re := range resolved.PromptRegexps {
        if re.MatchString(promptContent) {
            matched = true
            break
        }
    }
    if !matched {
        t.Error("prompt content should match at least one prompt pattern")
    }
}

func TestCopilotPatternsNoCollision(t *testing.T) {
    // Verify Copilot patterns don't false-positive on other tools' content
    copilotRaw := DefaultRawPatterns("copilot")
    if copilotRaw == nil {
        t.Fatal("nil raw patterns")
    }
    resolved, _ := CompilePatterns(copilotRaw)

    // Generic content that should NOT trigger Copilot busy detection
    genericContent := "$ ls -la\ntotal 42\ndrwxr-xr-x  5 user user 160 Feb 18 10:00 .\n"

    for _, s := range resolved.BusyStrings {
        if strings.Contains(genericContent, s) {
            t.Errorf("busy pattern %q false-positives on generic content", s)
        }
    }
    for _, re := range resolved.BusyRegexps {
        if re.MatchString(genericContent) {
            t.Errorf("busy regex %v false-positives on generic content", re)
        }
    }
}
```

### Approval state detection

If Copilot approval prompts ("1. Yes / 2. Yes, and approve...") should be detected as a distinct state (e.g., "waiting for approval" vs "waiting for input"), document how the UI layer should interpret the match. Currently, both busy and prompt matches are used — approval prompts typically match as "prompt" (waiting for user action).

## Config Override Examples

Users can override or extend Copilot patterns in `config.toml`:

```toml
# Extend defaults (recommended)
[tools.copilot]
busy_patterns_extra = ["my custom busy text"]
prompt_patterns_extra = ["Custom>"]

# Replace all defaults (use with caution)
[tools.copilot]
busy_patterns = ["only-this-pattern"]
prompt_patterns = ["only-this-prompt"]
```

This is handled automatically by `MergeRawPatterns()` — no special code needed for Copilot.

## Exit Criteria

- [ ] `DefaultRawPatterns("copilot")` returns non-nil `RawPatterns`
- [ ] Busy patterns match actual Copilot thinking/busy terminal content
- [ ] Prompt patterns match actual Copilot idle/approval terminal content
- [ ] No false positives on generic shell content or other tools' output
- [ ] `patterns_test.go` has Copilot-specific test cases with real captured content
- [ ] Manual smoke: start a Copilot session → verify status bar shows busy during thinking → shows prompt when idle
- [ ] No regressions in existing Claude/Gemini/OpenCode/Codex pattern tests
