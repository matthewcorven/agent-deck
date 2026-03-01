package statedb

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMarshalUnmarshalToolData_CopilotFields(t *testing.T) {
	copilotID := "copilot-session-abc-123"
	copilotDetected := time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC)

	// Also set other fields to verify Copilot doesn't interfere
	claudeID := "claude-sess-1"
	claudeDetected := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)

	data := MarshalToolData(
		claudeID, claudeDetected,
		"", time.Time{}, // gemini
		nil, "",          // gemini yolo/model
		"", time.Time{}, // opencode
		"", time.Time{}, // codex
		copilotID, copilotDetected,
		"latest prompt", []string{"mcp1"},
		nil, // toolOptionsJSON
	)

	if len(data) == 0 {
		t.Fatal("MarshalToolData returned empty data")
	}

	// Unmarshal
	gotClaudeID, gotClaudeDetected,
		_, _,
		_, _,
		_, _,
		_, _,
		gotCopilotID, gotCopilotDetected,
		gotPrompt, gotMCPs,
		_ := UnmarshalToolData(data)

	if gotCopilotID != copilotID {
		t.Errorf("CopilotSessionID: got %q, want %q", gotCopilotID, copilotID)
	}
	if gotCopilotDetected.Unix() != copilotDetected.Unix() {
		t.Errorf("CopilotDetectedAt: got %v, want %v", gotCopilotDetected, copilotDetected)
	}
	// Verify other fields survived
	if gotClaudeID != claudeID {
		t.Errorf("ClaudeSessionID: got %q, want %q", gotClaudeID, claudeID)
	}
	if gotClaudeDetected.Unix() != claudeDetected.Unix() {
		t.Errorf("ClaudeDetectedAt: got %v, want %v", gotClaudeDetected, claudeDetected)
	}
	if gotPrompt != "latest prompt" {
		t.Errorf("LatestPrompt: got %q, want %q", gotPrompt, "latest prompt")
	}
	if len(gotMCPs) != 1 || gotMCPs[0] != "mcp1" {
		t.Errorf("LoadedMCPNames: got %v, want [mcp1]", gotMCPs)
	}
}

func TestMarshalUnmarshalToolData_CopilotEmpty(t *testing.T) {
	// Copilot fields empty should round-trip as empty
	data := MarshalToolData(
		"", time.Time{},
		"", time.Time{},
		nil, "",
		"", time.Time{},
		"", time.Time{},
		"", time.Time{}, // copilot: empty
		"", nil,
		nil,
	)

	_, _,
		_, _,
		_, _,
		_, _,
		_, _,
		copilotID, copilotDetected,
		_, _,
		_ := UnmarshalToolData(data)

	if copilotID != "" {
		t.Errorf("expected empty CopilotSessionID, got %q", copilotID)
	}
	if !copilotDetected.IsZero() {
		t.Errorf("expected zero CopilotDetectedAt, got %v", copilotDetected)
	}
}

func TestMarshalUnmarshalToolData_CopilotWithToolOptions(t *testing.T) {
	toolOpts := json.RawMessage(`{"tool":"copilot","options":{"model":"gpt-4o","yolo_mode":true}}`)

	data := MarshalToolData(
		"", time.Time{},
		"", time.Time{},
		nil, "",
		"", time.Time{},
		"", time.Time{},
		"copilot-sess-xyz", time.Date(2026, 2, 28, 15, 30, 0, 0, time.UTC),
		"", nil,
		toolOpts,
	)

	_, _,
		_, _,
		_, _,
		_, _,
		_, _,
		gotCopilotID, gotCopilotDetected,
		_, _,
		gotToolOpts := UnmarshalToolData(data)

	if gotCopilotID != "copilot-sess-xyz" {
		t.Errorf("CopilotSessionID: got %q, want %q", gotCopilotID, "copilot-sess-xyz")
	}
	if gotCopilotDetected.IsZero() {
		t.Error("expected non-zero CopilotDetectedAt")
	}
	if len(gotToolOpts) == 0 {
		t.Fatal("expected non-empty ToolOptions")
	}

	// Verify the JSON round-tripped correctly
	var wrapper struct {
		Tool    string          `json:"tool"`
		Options json.RawMessage `json:"options"`
	}
	if err := json.Unmarshal(gotToolOpts, &wrapper); err != nil {
		t.Fatalf("failed to unmarshal tool options: %v", err)
	}
	if wrapper.Tool != "copilot" {
		t.Errorf("expected tool='copilot', got %q", wrapper.Tool)
	}
}

func TestUnmarshalToolData_EmptyData(t *testing.T) {
	_, _,
		_, _,
		_, _,
		_, _,
		_, _,
		copilotID, copilotDetected,
		_, _,
		_ := UnmarshalToolData(nil)

	if copilotID != "" {
		t.Errorf("expected empty CopilotSessionID from nil data, got %q", copilotID)
	}
	if !copilotDetected.IsZero() {
		t.Errorf("expected zero CopilotDetectedAt from nil data, got %v", copilotDetected)
	}
}
