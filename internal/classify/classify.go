package classify

import (
	"fmt"
	"time"
)

type Category string

const (
	CategoryToolError   Category = "tool_error"
	CategoryRefusal     Category = "refusal"
	CategoryToolAnomaly Category = "tool_anomaly"
)

// Event is a normalized representation of a raw OpenObserve hit.
type Event struct {
	ID          string
	Timestamp   time.Time
	SessionID   string
	ProjectName string
	EventName   string
	ToolName    string
	Body        string
	ErrorType   string
	Success     string
	Decision    string
	Category    Category
}

func (e *Event) Title() string {
	switch e.Category {
	case CategoryToolError:
		tool := e.ToolName
		if tool == "" {
			tool = "unknown"
		}
		errType := e.ErrorType
		if errType == "" {
			errType = "tool_result_failure"
		}
		return fmt.Sprintf("[tool_error] %s — %s", tool, errType)
	case CategoryRefusal:
		if e.EventName == "claude_code.tool_decision" {
			return fmt.Sprintf("[refusal] tool_decision rejected: %s", e.ToolName)
		}
		return fmt.Sprintf("[refusal] ERROR severity: %s", e.EventName)
	case CategoryToolAnomaly:
		return fmt.Sprintf("[tool_anomaly] %s", e.EventName)
	default:
		return fmt.Sprintf("[unknown] %s", e.EventName)
	}
}

// FromHit converts a raw OpenObserve result row into an Event.
func FromHit(hit map[string]any) *Event {
	e := &Event{
		ID:          strVal(hit, "_id"),
		SessionID:   strVal(hit, "session_id"),
		ProjectName: strVal(hit, "project_name"),
		EventName:   strVal(hit, "event_name"),
		ToolName:    strVal(hit, "tool_name"),
		Body:        strVal(hit, "body"),
		ErrorType:   strVal(hit, "error_type"),
		Success:     strVal(hit, "success"),
		Decision:    strVal(hit, "decision"),
	}

	// _timestamp is microseconds epoch from OpenObserve
	if ts, ok := hit["_timestamp"]; ok {
		switch v := ts.(type) {
		case float64:
			e.Timestamp = time.UnixMicro(int64(v)).UTC()
		case int64:
			e.Timestamp = time.UnixMicro(v).UTC()
		}
	}

	e.Category = categorize(e)
	return e
}

func categorize(e *Event) Category {
	switch e.EventName {
	case "claude_code.tool_result":
		if e.Success == "false" {
			return CategoryToolError
		}
	case "claude_code.tool_decision":
		if e.Decision == "reject" {
			return CategoryRefusal
		}
	case "claude_code.api_error", "claude_code.api_retries_exhausted", "claude_code.internal_error":
		return CategoryToolAnomaly
	}
	return CategoryToolAnomaly
}

func strVal(m map[string]any, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}
