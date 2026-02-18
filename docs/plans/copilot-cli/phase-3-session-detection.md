# Phase 3 — Session Detection + Resume

> **Goal:** agent-deck can detect the Copilot session ID after spawn and resume sessions across restarts.  
> **Depends on:** Phase 2 (Instance fields + buildCopilotCommand must exist), Phase 0 (session ID strategy).  
> **Estimated scope:** ~2 files modified, detection logic + tests.

## Context

Each tool has a different session ID detection strategy:

| Tool | Strategy |
|------|----------|
| Claude | Parse `~/.claude/projects/<path>/` session files, or hooks provide it |
| Gemini | `gemini --output-format stream-json` emits session ID in first message, or scan `~/.gemini/sessions/` |
| OpenCode | `opencode session list --format json` filtered by project directory |
| Codex | Scan `~/.codex/sessions/YYYY/MM/DD/*.jsonl` for most recent match |

For Copilot, the detection strategy depends on **Phase 0 findings**. The three candidate approaches are:

- **Option A (file-based):** Parse `~/.copilot/` storage directory for session files matching the working directory.
- **Option B (banner scraping):** Parse the tmux pane content for a session ID shown in the welcome banner.
- **Option C (--continue only):** Don't track explicit IDs; always use `--continue` for resume. Simplest fallback.

This doc describes the implementation framework. Fill in the detection logic after Phase 0 captures are committed.

## Files to Modify

### 1. `internal/session/instance.go` — Session detection

**Add `detectCopilotSessionAsync()`** (follow `detectOpenCodeSessionAsync()` / `detectCodexSessionAsync()` pattern):

```go
// detectCopilotSessionAsync detects the Copilot session ID after startup.
// Strategy depends on Phase 0 findings.
func (i *Instance) detectCopilotSessionAsync() {
    // Short delay to let Copilot start up
    time.Sleep(2 * time.Second)

    // === OPTION A: File-based detection ===
    // configDir := "~/.copilot"  // or from CopilotSettings.ConfigDir
    // Scan configDir for session storage files
    // Match by project path / most recently modified
    // Extract session ID from filename or file content

    // === OPTION B: Banner scraping ===
    // content, err := i.tmuxSession.CapturePane()
    // Parse session ID from welcome banner using regex

    // === OPTION C: --continue fallback ===
    // No explicit ID needed; mark as "detected" so Restart uses --continue

    // If session ID found:
    // i.CopilotSessionID = detectedID
    // i.CopilotDetectedAt = time.Now()
    // if i.tmuxSession != nil {
    //     i.tmuxSession.SetEnvironment("COPILOT_SESSION_ID", detectedID)
    // }
    // sessionLog.Info("copilot_session_detected", slog.String("session_id", detectedID))
}

// DetectCopilotSession is the public wrapper for async detection.
// Call for restored sessions that don't have a session ID yet.
func (i *Instance) DetectCopilotSession() {
    i.detectCopilotSessionAsync()
}
```

**Update `Start()`** — after the `buildCopilotCommand` case, start async detection:

```go
case "copilot":
    command = i.buildCopilotCommand(i.Command)
    i.CopilotStartedAt = time.Now().UnixMilli()
    // Start async session ID detection after tmux pane spawn
    // (launched after tmux.Start() completes, below)
```

Then after the `tmuxSession.Start()` call, add:

```go
if i.Tool == "copilot" && i.CopilotSessionID == "" {
    go i.detectCopilotSessionAsync()
}
```

**Update `Restart()`** — add Copilot respawn block (after the Codex block):

```go
// If Copilot session AND tmux session exists, use respawn-pane
if i.Tool == "copilot" && i.tmuxSession != nil && i.tmuxSession.Exists() {
    // Try to recover session ID from tmux env
    if i.CopilotSessionID == "" {
        if envID, err := i.tmuxSession.GetEnvironment("COPILOT_SESSION_ID"); err == nil && envID != "" {
            i.CopilotSessionID = envID
            i.CopilotDetectedAt = time.Now()
        }
    }

    var resumeCmd string
    if i.CopilotSessionID != "" {
        resumeCmd = i.buildCopilotCommand("copilot") // will use --resume
    } else {
        // No session ID — use --continue to resume most recent
        envPrefix := i.buildEnvSourceCommand()
        copilotCmd := "copilot"
        if config, err := LoadUserConfig(); err == nil && config != nil {
            copilotCmd = config.Copilot.GetCommand()
        }
        resumeCmd = envPrefix + copilotCmd + " --continue" + i.buildCopilotExtraFlags()
    }

    resumeCmd, err := i.applyWrapper(resumeCmd)
    if err != nil {
        return err
    }

    if err := i.tmuxSession.RespawnPane(resumeCmd); err != nil {
        return fmt.Errorf("failed to restart Copilot session: %w", err)
    }

    if i.CopilotSessionID == "" {
        go i.detectCopilotSessionAsync()
    }
    i.Status = StatusWaiting
    return nil
}
```

**Update Restart() fallback** — in the "recreate tmux session" block (~line 2955), add:

```go
} else if i.Tool == "copilot" && i.CopilotSessionID != "" {
    command = i.buildCopilotCommand("copilot")
```

And in the fresh-start switch:

```go
case "copilot":
    command = i.buildCopilotCommand(i.Command)
    i.CopilotStartedAt = time.Now().UnixMilli()
```

### 2. `internal/session/instance.go` — PostStartSync

Update `PostStartSync()` if Copilot needs synchronous post-start detection:

```go
case "copilot":
    // Async detection already started by Start(), skip here
```

## Tests to Add

### `internal/session/instance_test.go`

```go
func TestCopilotResume_WithSessionID(t *testing.T)
    // Instance with CopilotSessionID set
    // buildCopilotCommand("copilot") → contains "--resume SESSION-ID"
    // tmux set-environment COPILOT_SESSION_ID set

func TestCopilotResume_ContinueFallback(t *testing.T)
    // Instance with no CopilotSessionID
    // Restart logic should produce "--continue" flag

func TestCopilotResume_FreshStart(t *testing.T)
    // Instance with no session ID, first start
    // buildCopilotCommand("copilot") → just "copilot" + extra flags, no resume
```

### Detection tests (depends on chosen strategy)

If Option A (file-based):
```go
func TestDetectCopilotSession_FileMatch(t *testing.T)
    // Create temp dir mimicking ~/.copilot/ structure
    // Verify correct session file is matched by project path

func TestDetectCopilotSession_NoMatch(t *testing.T)
    // Empty dir → CopilotSessionID stays empty
```

If Option B (banner scraping):
```go
func TestDetectCopilotSession_BannerParse(t *testing.T)
    // Mock tmux pane content with session ID in banner
    // Verify session ID extracted correctly
```

## Exit Criteria

- [ ] `detectCopilotSessionAsync()` implemented (strategy chosen from Phase 0)
- [ ] tmux environment variable `COPILOT_SESSION_ID` set when detected
- [ ] `Restart()` handles Copilot: resume with `--resume ID` or `--continue` fallback
- [ ] Async detection launched from `Start()` when session ID unknown
- [ ] Manual smoke: start Copilot session → quit agent-deck → restart → session resumes
- [ ] All resume/detection tests pass
