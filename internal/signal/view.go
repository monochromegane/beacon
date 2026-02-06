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
func (s *Signal) ToView() View {
	title := ""
	if s.Environment != nil {
		title = s.Environment.PaneTitle
	}
	if title == "" {
		title = s.CustomMessage
	}
	return View{
		SessionID:     s.SessionID,
		State:         string(s.State),
		Message:       s.Message,
		CustomMessage: s.CustomMessage,
		Title:         title,
	}
}
