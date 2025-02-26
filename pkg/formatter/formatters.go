package formatter

import (
	"fmt"
	"time"
)

// FormatTime formats a time.Time into a human-readable relative time
func FormatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)
	
	if diff < time.Minute {
		return "Less than a minute ago"
	} else if diff < time.Hour {
		return fmt.Sprintf("%d minutes ago", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(diff.Hours()))
	} else if diff < 48*time.Hour {
		return "Yesterday"
	} else {
		return fmt.Sprintf("%d days ago", int(diff.Hours()/24))
	}
}

// FormatSize formats bytes into a human-readable size
func FormatSize(bytes float64) string {
	const unit = 1024.0
	if bytes < unit {
		return fmt.Sprintf("%.1f B", bytes)
	}
	div, exp := unit, 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", bytes/div, "KMGTPE"[exp])
}