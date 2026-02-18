# Phase 5 — Preflight Checks + Error UX

> **Goal:** Users without the Copilot CLI installed get actionable guidance instead of a cryptic failure.  
> **Depends on:** Phase 4 (Copilot must be fully wired as a tool before adding preflight gate).  
> **Estimated scope:** ~2 files modified. Minimal logic.

## Context

Other tools don't currently have explicit install checks — they either fail in tmux (showing a shell error) or rely on the user to have the binary installed. This phase adds a friendlier experience for Copilot specifically because:

1. Copilot CLI is new and users are less likely to have it installed.
2. The install path is straightforward (`brew install copilot-cli`).
3. Auth is fully handled by the CLI itself (`/login`), so no external auth check needed.

## Files to Modify

### 1. Session start path — binary check

Find where `Start()` (or the UI layer that calls it) runs for new Copilot sessions. Add a preflight check before spawning the tmux pane.

**Search for the check pattern:**
```
grep -rn 'which\|exec.LookPath\|commandExists\|preflight' internal/session/ internal/ui/ cmd/
```

**Add check** — either in `instance.go`'s `Start()` or in the UI layer that creates sessions:

```go
// Before starting a Copilot session, verify the binary exists
if i.Tool == "copilot" {
    copilotCmd := "copilot"
    if config, err := LoadUserConfig(); err == nil && config != nil {
        copilotCmd = config.Copilot.GetCommand()
    }
    if _, err := exec.LookPath(copilotCmd); err != nil {
        return fmt.Errorf(
            "copilot CLI not found in PATH. Install it:\n"+
                "  brew install copilot-cli\n"+
                "  npm install -g @github/copilot\n"+
                "  curl -fsSL https://gh.io/copilot-install | bash")
    }
}
```

### 2. UI layer — toast/tooltip

If the project uses toast messages or error toasts in the TUI, surface the preflight error there instead of (or in addition to) `Start()` returning an error.

**Search for toast/notification patterns:**
```
grep -rn 'toast\|showError\|notification\|ShowToast' internal/ui/
```

Wire the preflight error to show a dismissable toast with install instructions.

### 3. Settings panel mention

In the Settings panel (if there's a Copilot section from Phase 1), add a note:

```
Requires: copilot CLI (brew install copilot-cli)
```

Search for where other tool prerequisites are mentioned in the UI:
```
grep -rn 'prerequisite\|requires\|install' internal/ui/
```

## Tests to Add

### `internal/session/instance_test.go` (or a new preflight test file)

```go
func TestCopilotPreflight_BinaryNotFound(t *testing.T) {
    inst := NewInstanceWithTool("test", "/tmp/test", "copilot")
    inst.Command = "copilot"
    
    // Set an invalid command to simulate missing binary
    // (or use a CopilotSettings with command = "nonexistent-copilot-binary")
    // The Start() call should return an error with install instructions
    
    // Verify the error message contains install instructions
    err := inst.Start()
    if err == nil {
        // Binary might actually exist — skip test or mock
        t.Skip("copilot binary found in PATH, cannot test missing binary scenario")
    }
    if !strings.Contains(err.Error(), "brew install") {
        t.Errorf("expected install instructions in error, got: %v", err)
    }
}
```

Note: This test is hard to write without mocking `exec.LookPath`. Consider:
- Using a config override to set `command = "nonexistent-binary-name"` 
- Using `t.Setenv("PATH", "/nonexistent")` to temporarily clear PATH (careful with side effects)
- Accepting that this is primarily a manual test

## Exit Criteria

- [ ] Starting a Copilot session when binary is missing shows clear error with install instructions
- [ ] Error includes at least one install method (brew / npm)
- [ ] No crash or hang when binary is missing
- [ ] Settings panel mentions Copilot CLI prerequisite
- [ ] Manual test: rename/remove `copilot` from PATH → start session → see install instructions
