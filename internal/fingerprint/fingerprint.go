package fingerprint

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

var redactPatterns = []*regexp.Regexp{
	// UUIDs / session IDs
	regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`),
	// absolute paths
	regexp.MustCompile(`/(?:Users|home|var|tmp|opt)/[^\s"']+`),
	// timestamps (ISO 8601)
	regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z?`),
	// Unix epoch integers > 10 digits (ms/us timestamps)
	regexp.MustCompile(`\b\d{10,}\b`),
	// line numbers like ":42:" or "line 42"
	regexp.MustCompile(`(?:line |:)\d+`),
}

// Normalize strips volatile parts from body so semantically identical events
// produce the same fingerprint regardless of session or timestamp.
func Normalize(s string) string {
	for _, re := range redactPatterns {
		s = re.ReplaceAllString(s, "<redacted>")
	}
	return s
}

// Compute returns a 12-char hex fingerprint for an event.
func Compute(eventName, toolName, errorType, body string) string {
	excerpt := body
	if len(excerpt) > 512 {
		excerpt = excerpt[:512]
	}
	normalized := Normalize(excerpt)
	raw := strings.Join([]string{eventName, toolName, errorType, normalized}, "\x1f")
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum[:6]) // 6 bytes = 12 hex chars
}
