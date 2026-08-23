package logwatch

import (
	"strings"
	"time"
)

const eqTimeLayout = "Mon Jan 2 15:04:05 2006"

// parseLogLineTime extracts the EQ timestamp prefix from a log line, if present.
// Supports "[Fri Aug 23 09:30:37 2026] …" and "Fri Aug 23 09:30:37 2026 …".
func parseLogLineTime(line string) (time.Time, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return time.Time{}, false
	}
	if strings.HasPrefix(line, "[") {
		if end := strings.Index(line, "]"); end > 1 {
			return parseEQTime(line[1:end])
		}
	}
	fields := strings.Fields(line)
	if len(fields) >= 5 {
		return parseEQTime(strings.Join(fields[:5], " "))
	}
	return time.Time{}, false
}

func parseEQTime(s string) (time.Time, bool) {
	t, err := time.ParseInLocation(eqTimeLayout, strings.TrimSpace(s), time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// logTimeRecent reports whether ts is fresh enough to affect presence.
func logTimeRecent(ts, now time.Time, window time.Duration) bool {
	if ts.IsZero() {
		return false
	}
	age := now.Sub(ts)
	if age < 0 {
		age = -age
	}
	if age > window {
		return false
	}
	// Reject clocks impossibly far in the future (bad parse / corrupt line).
	return !ts.After(now.Add(2 * time.Minute))
}
