package signal

// View represents a signal's display information, independent of environment.
type View struct {
	SessionID     string `json:"session_id"`
	State         string `json:"state"`
	Message       string `json:"message"`
	CustomMessage string `json:"custom_message,omitempty"`
	Title         string `json:"title"`
}

// ToView converts a Signal to a View for display.
// Title resolution: PaneTitle (from environment) takes precedence, then CustomMessage.
func (s *Signal) ToView() View {
	return View{
		SessionID:     s.SessionID,
		State:         string(s.State),
		Message:       s.Message,
		CustomMessage: s.CustomMessage,
		Title:         s.resolveTitle(),
	}
}

// resolveTitle returns the display title using PaneTitle with CustomMessage as fallback.
func (s *Signal) resolveTitle() string {
	if s.Environment != nil && s.Environment.PaneTitle != "" {
		return s.Environment.PaneTitle
	}
	return s.CustomMessage
}
