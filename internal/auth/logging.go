package auth

import (
	"os"
	"strings"
	"time"
)

var loggingEnv = os.Getenv("LOGGING")

// LogAuthAttempt appends an authentication attempt record to log/auth.log.
// Fields: timestamp (RFC3339) | level | authType | status | identifier? | message?
// level: debug|info|warning|error|fatal
// authType: Local|Google|...
// status: Success|Fail
// identifier: username, email, userID, etc. (optional)
// message: additional info (optional)
func LogAuthAttempt(level string, authType string, status string, identifier string, message string) {
	if !strings.EqualFold(loggingEnv, "true") {
		return
	}

	// ensure log directory exists
	if err := os.MkdirAll("log", 0o750); err != nil {
		// best-effort: if logging fails, do not crash the app, just return
		return
	}
	f, err := os.OpenFile("log/auth.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		// best-effort: if logging fails, do not crash the app, just return
		return
	}
	defer func() { _ = f.Close() }()

	ts := time.Now().UTC().Format(time.RFC3339)
	parts := []string{ts, level, authType, status}
	if identifier != "" {
		parts = append(parts, identifier)
	}
	if message != "" {
		parts = append(parts, message)
	}
	line := strings.Join(parts, " | ") + "\n"

	if _, err := f.WriteString(line); err != nil {
		// best-effort: ignore write errors
		return
	}
}
