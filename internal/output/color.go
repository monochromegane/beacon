package output

import (
	"os"

	"golang.org/x/term"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

// ColorScheme holds color codes for each state.
type ColorScheme struct {
	Waiting string
	Idle    string
	Started string
	Running string
	Reset   string
}

// NewColorScheme creates a ColorScheme based on whether colors are enabled.
func NewColorScheme(useColor bool) ColorScheme {
	if !useColor {
		return ColorScheme{}
	}
	return ColorScheme{
		Waiting: colorYellow,
		Idle:    colorCyan,
		Started: colorBlue,
		Running: colorGreen,
		Reset:   colorReset,
	}
}

// ShouldUseColor determines if color output should be used.
// It respects the --color flag and NO_COLOR environment variable.
func ShouldUseColor(colorFlag string) bool {
	switch colorFlag {
	case "always":
		return true
	case "never":
		return false
	default: // "auto"
		// Respect NO_COLOR environment variable
		if _, exists := os.LookupEnv("NO_COLOR"); exists {
			return false
		}
		// Check if stdout is a terminal
		return term.IsTerminal(int(os.Stdout.Fd()))
	}
}
