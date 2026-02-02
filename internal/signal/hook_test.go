package signal

import (
	"strings"
	"testing"
)

func TestParseHookEvent(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *HookEvent
		wantErr bool
	}{
		{
			name:  "SessionStart",
			input: `{"session_id":"abc123","hook_event_name":"SessionStart","source":"cli"}`,
			want: &HookEvent{
				SessionID:     "abc123",
				HookEventName: "SessionStart",
				Source:        "cli",
			},
		},
		{
			name:  "Notification with type",
			input: `{"session_id":"abc123","hook_event_name":"Notification","notification_type":"permission_prompt"}`,
			want: &HookEvent{
				SessionID:        "abc123",
				HookEventName:    "Notification",
				NotificationType: "permission_prompt",
			},
		},
		{
			name:    "Invalid JSON",
			input:   `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			got, err := ParseHookEvent(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHookEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.SessionID != tt.want.SessionID {
				t.Errorf("SessionID = %q, want %q", got.SessionID, tt.want.SessionID)
			}
			if got.HookEventName != tt.want.HookEventName {
				t.Errorf("HookEventName = %q, want %q", got.HookEventName, tt.want.HookEventName)
			}
		})
	}
}

func TestMapEventToState(t *testing.T) {
	tests := []struct {
		name        string
		event       *HookEvent
		wantState   State
		wantMessage string
		wantDelete  bool
		wantSkip    bool
	}{
		{
			name:        "SessionStart",
			event:       &HookEvent{HookEventName: "SessionStart", Source: "cli"},
			wantState:   StateStarted,
			wantMessage: "claude:started:cli",
		},
		{
			name:        "UserPromptSubmit",
			event:       &HookEvent{HookEventName: "UserPromptSubmit"},
			wantState:   StateRunning,
			wantMessage: "claude:running",
		},
		{
			name:        "PreToolUse",
			event:       &HookEvent{HookEventName: "PreToolUse"},
			wantState:   StateRunning,
			wantMessage: "claude:running",
		},
		{
			name:        "Notification permission_prompt",
			event:       &HookEvent{HookEventName: "Notification", NotificationType: "permission_prompt"},
			wantState:   StateWaiting,
			wantMessage: "claude:waiting",
		},
		{
			name:        "Notification elicitation_dialog",
			event:       &HookEvent{HookEventName: "Notification", NotificationType: "elicitation_dialog"},
			wantState:   StateWaiting,
			wantMessage: "claude:waiting",
		},
		{
			name:        "Notification other",
			event:       &HookEvent{HookEventName: "Notification", NotificationType: "other"},
			wantState:   StateRunning,
			wantMessage: "claude:running",
		},
		{
			name:     "Notification idle_prompt",
			event:    &HookEvent{HookEventName: "Notification", NotificationType: "idle_prompt"},
			wantSkip: true,
		},
		{
			name:        "Stop",
			event:       &HookEvent{HookEventName: "Stop"},
			wantState:   StateIdle,
			wantMessage: "claude:idle",
		},
		{
			name:       "SessionEnd",
			event:      &HookEvent{HookEventName: "SessionEnd"},
			wantDelete: true,
		},
		{
			name:        "Unknown event",
			event:       &HookEvent{HookEventName: "Unknown"},
			wantState:   StateRunning,
			wantMessage: "claude:running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapEventToState(tt.event)
			if result.ShouldDelete != tt.wantDelete {
				t.Errorf("ShouldDelete = %v, want %v", result.ShouldDelete, tt.wantDelete)
			}
			if result.ShouldSkip != tt.wantSkip {
				t.Errorf("ShouldSkip = %v, want %v", result.ShouldSkip, tt.wantSkip)
			}
			if !tt.wantDelete && !tt.wantSkip {
				if result.State != tt.wantState {
					t.Errorf("State = %q, want %q", result.State, tt.wantState)
				}
				if result.Message != tt.wantMessage {
					t.Errorf("Message = %q, want %q", result.Message, tt.wantMessage)
				}
			}
		})
	}
}
