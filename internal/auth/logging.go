package auth

import (
	"fmt"
	"os"
	"path/filepath"
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
	if strings.ToLower(loggingEnv) != "true" {
		return
	}
	// ensure log directory exists
	dir := filepath.Join("log")
	_ = os.MkdirAll(dir, 0o755)

	fpath := filepath.Join(dir, "auth.log")
	f, err := os.OpenFile(fpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		// best-effort: if logging fails, do not crash the app, just return
		return
	}
	defer f.Close()

	ts := time.Now().UTC().Format(time.RFC3339)
	line := fmt.Sprintf("%s | %s | %s | %s", ts, level, authType, status)
	if identifier != "" {
		line = line + " | " + identifier
	}
	if message != "" {
		line = line + " | " + message
	}
	line = line + "\n"

	_, _ = f.WriteString(line)
}
