package signal

import (
	"encoding/json"
	"io"
)

// HookEvent represents a Claude Code hook event received from stdin.
type HookEvent struct {
	SessionID        string `json:"session_id"`
	HookEventName    string `json:"hook_event_name"`
	Source           string `json:"source"`
	NotificationType string `json:"notification_type"`
}

// HookEventResult represents the mapping result of a hook event.
type HookEventResult struct {
	State        State
	Message      string
	ShouldDelete bool
}

// ParseHookEvent parses a hook event from JSON input.
func ParseHookEvent(r io.Reader) (*HookEvent, error) {
	var event HookEvent
	if err := json.NewDecoder(r).Decode(&event); err != nil {
		return nil, err
	}
	return &event, nil
}

// MapEventToState maps a hook event to a signal state.
func MapEventToState(event *HookEvent) HookEventResult {
	switch event.HookEventName {
	case "SessionStart":
		return HookEventResult{
			State:   StateStarted,
			Message: "claude:started:" + event.Source,
		}
	case "UserPromptSubmit":
		return HookEventResult{
			State:   StateRunning,
			Message: "claude:running",
		}
	case "PreToolUse":
		return HookEventResult{
			State:   StateRunning,
			Message: "claude:running",
		}
	case "Notification":
		if event.NotificationType == "permission_prompt" || event.NotificationType == "elicitation_dialog" {
			return HookEventResult{
				State:   StateWaiting,
				Message: "claude:waiting",
			}
		}
		return HookEventResult{
			State:   StateRunning,
			Message: "claude:running",
		}
	case "Stop":
		return HookEventResult{
			State:   StateIdle,
			Message: "claude:idle",
		}
	case "SessionEnd":
		return HookEventResult{
			ShouldDelete: true,
		}
	default:
		return HookEventResult{
			State:   StateRunning,
			Message: "claude:running",
		}
	}
}
