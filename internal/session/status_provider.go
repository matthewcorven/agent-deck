package session

import "time"

// ToolStatus represents the current status of an AI tool session.
type ToolStatus int

const (
	ToolStatusUnknown   ToolStatus = iota
	ToolStatusBusy                 // Tool is processing / generating
	ToolStatusIdle                 // Tool is waiting for input
	ToolStatusPrompting            // Tool is asking user a question
	ToolStatusError                // Tool encountered an error
)

// String returns the human-readable name of the status.
func (s ToolStatus) String() string {
	switch s {
	case ToolStatusBusy:
		return "busy"
	case ToolStatusIdle:
		return "idle"
	case ToolStatusPrompting:
		return "prompting"
	case ToolStatusError:
		return "error"
	default:
		return "unknown"
	}
}

// StatusProvider abstracts tool session status detection.
// The CLI implementation wraps tmux pattern matching; a future ACP
// implementation would wrap session/update event subscription.
type StatusProvider interface {
	Status() ToolStatus
	LastActivity() time.Time
	SessionID() (string, bool) // (id, detected)
}
