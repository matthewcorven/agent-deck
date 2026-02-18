# Phase 2 — Command Builder, Options, Storage

> **Goal:** `buildCopilotCommand` produces the correct shell command; session metadata is persisted; `-i PROMPT` delivers initial messages.  
> **Depends on:** Phase 1 (CopilotSettings struct must exist).  
> **Estimated scope:** ~4 files modified, ~3 files for tests.

## Context

Every built-in tool follows this pattern for command construction:

1. **Options struct** in `internal/session/tooloptions.go` — holds per-session overrides, implements `ToolOptions` interface (`ToolName()`, `ToArgs()`)
2. **`buildXCommand(baseCommand string)`** in `internal/session/instance.go` — composes the full shell command including env prefix, resume flags, model/yolo flags
3. **Instance fields** for session metadata (session ID, detected timestamp, per-session overrides)
4. **Storage fields** in `internal/statedb/migrate.go` — JSON migration struct mirrors Instance fields
5. **Start/Restart dispatch** — switch cases in `Start()`, `StartWithMessage()`, `Restart()` route to the tool's command builder

### Existing patterns to follow

- `buildGeminiCommand` (~line 470 of instance.go): env prefix + yolo flag + model flag + resume by session ID
- `buildCodexCommand` (~line 616 of instance.go): env prefix + AGENTDECK env vars + yolo flag + resume by session ID
- `buildOpenCodeCommand` (~line 531 of instance.go): env prefix + model flag + agent flag + resume by session ID
- `CodexOptions` / `OpenCodeOptions` in tooloptions.go: full marshal/unmarshal pattern

## Files to Modify

### 1. `internal/session/tooloptions.go` — Add `CopilotOptions`

Add after `CodexOptions` (or after `OpenCodeOptions`):

```go
// CopilotOptions holds launch options for Copilot CLI sessions
type CopilotOptions struct {
    // SessionMode: "new" (default), "continue" (--continue), or "resume" (--resume)
    SessionMode string `json:"session_mode,omitempty"`
    // ResumeSessionID for --resume SESSION-ID (only when SessionMode="resume")
    ResumeSessionID string `json:"resume_session_id,omitempty"`
    // Model overrides the default model (--model)
    Model string `json:"model,omitempty"`
    // Agent overrides the default agent (--agent)
    Agent string `json:"agent,omitempty"`
    // YoloMode enables --yolo/--allow-all (nil = inherit from config)
    YoloMode *bool `json:"yolo_mode,omitempty"`
    // ConfigDir overrides --config-dir
    ConfigDir string `json:"config_dir,omitempty"`
}

func (o *CopilotOptions) ToolName() string { return "copilot" }

func (o *CopilotOptions) ToArgs() []string {
    var args []string

    switch o.SessionMode {
    case "continue":
        args = append(args, "--continue")
    case "resume":
        if o.ResumeSessionID != "" {
            args = append(args, "--resume", o.ResumeSessionID)
        }
    }

    if o.Model != "" {
        args = append(args, "--model", o.Model)
    }
    if o.Agent != "" {
        args = append(args, "--agent", o.Agent)
    }
    if o.YoloMode != nil && *o.YoloMode {
        args = append(args, "--yolo")
    }
    if o.ConfigDir != "" {
        args = append(args, "--config-dir", o.ConfigDir)
    }

    return args
}

// ToArgsForFork returns arguments for fork resume (session mode excluded)
func (o *CopilotOptions) ToArgsForFork() []string {
    var args []string
    if o.Model != "" {
        args = append(args, "--model", o.Model)
    }
    if o.Agent != "" {
        args = append(args, "--agent", o.Agent)
    }
    if o.YoloMode != nil && *o.YoloMode {
        args = append(args, "--yolo")
    }
    if o.ConfigDir != "" {
        args = append(args, "--config-dir", o.ConfigDir)
    }
    return args
}

func NewCopilotOptions(config *UserConfig) *CopilotOptions {
    opts := &CopilotOptions{SessionMode: "new"}
    if config != nil {
        opts.Model = config.Copilot.DefaultModel
        opts.Agent = config.Copilot.DefaultAgent
        if config.Copilot.ConfigDir != "" {
            opts.ConfigDir = config.Copilot.ConfigDir
        }
        if config.Copilot.YoloMode {
            yolo := true
            opts.YoloMode = &yolo
        }
    }
    return opts
}

func UnmarshalCopilotOptions(data json.RawMessage) (*CopilotOptions, error) {
    if len(data) == 0 {
        return nil, nil
    }
    var wrapper ToolOptionsWrapper
    if err := json.Unmarshal(data, &wrapper); err != nil {
        return nil, err
    }
    if wrapper.Tool != "copilot" {
        return nil, nil
    }
    var opts CopilotOptions
    if err := json.Unmarshal(wrapper.Options, &opts); err != nil {
        return nil, err
    }
    return &opts, nil
}
```

### 2. `internal/session/instance.go` — Add Copilot fields + command builder

**Add Instance fields** (after the Codex fields, ~line 101):

```go
// Copilot CLI integration
CopilotSessionID  string    `json:"copilot_session_id,omitempty"`
CopilotDetectedAt time.Time `json:"copilot_detected_at,omitempty"`
CopilotStartedAt  int64     `json:"-"` // Unix millis when started (for session matching)
```

**Add `buildCopilotCommand`** (after `buildCodexCommand`):

```go
// buildCopilotCommand builds the command for GitHub Copilot CLI
// Resume: copilot --resume SESSION-ID or copilot --continue
// Initial prompt: copilot -i "PROMPT"
func (i *Instance) buildCopilotCommand(baseCommand string) string {
    if i.Tool != "copilot" {
        return baseCommand
    }

    envPrefix := i.buildEnvSourceCommand()
    agentdeckEnvPrefix := fmt.Sprintf("AGENTDECK_INSTANCE_ID=%s AGENTDECK_TITLE=%q AGENTDECK_TOOL=%s ",
        i.ID, i.Title, i.Tool)
    envPrefix += agentdeckEnvPrefix

    // Resolve copilot binary
    copilotCmd := "copilot"
    if config, err := LoadUserConfig(); err == nil && config != nil {
        copilotCmd = config.Copilot.GetCommand()
    }

    if baseCommand == copilotCmd || baseCommand == "copilot" {
        extraFlags := i.buildCopilotExtraFlags()

        if i.CopilotSessionID != "" {
            return envPrefix + fmt.Sprintf(
                "tmux set-environment COPILOT_SESSION_ID %s; %s --resume %s%s",
                i.CopilotSessionID, copilotCmd, i.CopilotSessionID, extraFlags)
        }
        return envPrefix + copilotCmd + extraFlags
    }

    return envPrefix + baseCommand
}
```

**Add `buildCopilotExtraFlags`** helper:

```go
func (i *Instance) buildCopilotExtraFlags() string {
    opts := i.GetCopilotOptions()
    if opts == nil {
        if config, err := LoadUserConfig(); err == nil && config != nil {
            opts = NewCopilotOptions(config)
        }
    }
    if opts == nil {
        return ""
    }

    var flags string
    if opts.YoloMode != nil && *opts.YoloMode {
        flags += " --yolo"
    } else {
        // Check global config fallback
        if config, err := LoadUserConfig(); err == nil && config != nil && config.Copilot.YoloMode {
            flags += " --yolo"
        }
    }
    if opts.Model != "" {
        flags += " --model " + opts.Model
    }
    if opts.Agent != "" {
        flags += " --agent " + opts.Agent
    }
    if opts.ConfigDir != "" {
        flags += " --config-dir " + opts.ConfigDir
    }
    return flags
}
```

**Add `GetCopilotOptions`** accessor (follow `GetCodexOptions` pattern):

```go
func (i *Instance) GetCopilotOptions() *CopilotOptions {
    if len(i.ToolOptionsJSON) == 0 {
        return nil
    }
    opts, _ := UnmarshalCopilotOptions(i.ToolOptionsJSON)
    return opts
}
```

**Wire into Start()** (~line 1264, add case in switch):

```go
case "copilot":
    command = i.buildCopilotCommand(i.Command)
    i.CopilotStartedAt = time.Now().UnixMilli()
```

**Wire into StartWithMessage()** (~line 1350, add case in switch):

```go
case "copilot":
    command = i.buildCopilotCommand(i.Command)
    i.CopilotStartedAt = time.Now().UnixMilli()
```

Note: For `StartWithMessage`, the Copilot CLI supports `-i PROMPT` for initial messages. If the existing message-delivery mechanism (send-keys after ready) works, use that. Otherwise, compose: `copilot -i "PROMPT"` in the command builder.

**Wire into Restart()** — add the respawn + fallback blocks (follow the Codex pattern in Restart).

### 3. `internal/statedb/migrate.go` — Add JSON fields

Add to `jsonInstanceData` struct (after `CodexDetectedAt`):

```go
CopilotSessionID  string    `json:"copilot_session_id,omitempty"`
CopilotDetectedAt time.Time `json:"copilot_detected_at,omitempty"`
```

### 4. Storage serialization

Search for where `CodexSessionID` is read/written to `tool_data` or instance JSON. The `statedb` package stores tool-specific data in a `tool_data TEXT` column as JSON. Ensure `CopilotSessionID` and `CopilotDetectedAt` are included in the serialization/deserialization path.

Key search:
```
grep -rn 'CodexSessionID\|codex_session_id' internal/statedb/
grep -rn 'tool_data\|ToolData' internal/statedb/
```

## Tests to Add

All tests go in the appropriate `_test.go` files alongside the source.

### `internal/session/instance_test.go`

```go
func TestBuildCopilotCommand_New(t *testing.T)
    // No session ID → starts fresh: "copilot"
    // Verify AGENTDECK_INSTANCE_ID is set

func TestBuildCopilotCommand_Resume(t *testing.T)
    // CopilotSessionID set → "copilot --resume SESSION-ID"
    // Verify tmux set-environment COPILOT_SESSION_ID

func TestBuildCopilotCommand_WithOptions(t *testing.T)
    // Model + Agent + Yolo + ConfigDir → all flags present

func TestBuildCopilotCommand_CustomCommand(t *testing.T)
    // Non-"copilot" base command passes through with env prefix
```

### `internal/session/tooloptions_test.go`

```go
func TestCopilotOptions_ToArgs(t *testing.T)
    // New session: no resume flags
    // Resume: --resume SESSION-ID
    // Continue: --continue
    // With model/agent/yolo/config-dir

func TestCopilotOptions_ToArgsForFork(t *testing.T)
    // Session mode flags excluded; model/agent/yolo preserved

func TestCopilotOptions_MarshalUnmarshal(t *testing.T)
    // Round-trip: marshal → unmarshal → compare

func TestNewCopilotOptions_DefaultsFromConfig(t *testing.T)
    // Config with yolo_mode=true → opts.YoloMode = &true
    // Config with default_model → opts.Model set
```

## Exit Criteria

- [ ] `CopilotOptions` struct implements `ToolOptions` interface
- [ ] `buildCopilotCommand` produces correct command for: new session, resume by ID, with model/agent/yolo/config-dir overrides
- [ ] Instance fields `CopilotSessionID`, `CopilotDetectedAt`, `CopilotStartedAt` exist
- [ ] `Start()` and `StartWithMessage()` dispatch to `buildCopilotCommand` for tool `"copilot"`
- [ ] `Restart()` handles Copilot sessions (respawn with resume if ID known, fresh start otherwise)
- [ ] Storage migration struct includes Copilot fields
- [ ] All new tests pass; existing tests unaffected
