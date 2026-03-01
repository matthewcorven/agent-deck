package session

import (
	"testing"
)

// ============================================================================
// ToolStatus.String() Tests (Phase 1 — Config Surface)
// ============================================================================

func TestToolStatus_String(t *testing.T) {
	tests := []struct {
		status ToolStatus
		want   string
	}{
		{ToolStatusUnknown, "unknown"},
		{ToolStatusBusy, "busy"},
		{ToolStatusIdle, "idle"},
		{ToolStatusPrompting, "prompting"},
		{ToolStatusError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("ToolStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestToolStatus_StringCoversAllValues(t *testing.T) {
	// Ensure we test every defined constant — if someone adds a new status,
	// this test reminds them to update String() and the test table above.
	allStatuses := []ToolStatus{
		ToolStatusUnknown,
		ToolStatusBusy,
		ToolStatusIdle,
		ToolStatusPrompting,
		ToolStatusError,
	}

	for _, s := range allStatuses {
		str := s.String()
		if str == "" {
			t.Errorf("ToolStatus(%d).String() returned empty string", s)
		}
	}
}

func TestToolStatus_UnknownIsZeroValue(t *testing.T) {
	// StatusUnknown should be the zero value so an uninitialized ToolStatus
	// defaults to "unknown" rather than a misleading state.
	var zero ToolStatus
	if zero != ToolStatusUnknown {
		t.Errorf("Zero value ToolStatus = %d, want %d (ToolStatusUnknown)", zero, ToolStatusUnknown)
	}
}
