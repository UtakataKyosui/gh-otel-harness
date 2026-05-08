package openobserve

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// EventTypes used as --type filter values
const (
	TypeToolError    = "tool_error"
	TypeRefusal      = "refusal"
	TypeToolAnomaly  = "tool_anomaly"
)

var AllTypes = []string{TypeToolError, TypeRefusal, TypeToolAnomaly}

type FetchOptions struct {
	Since         time.Duration
	Types         []string
	ProjectFilter string
	Limit         int
}

// BuildSQL returns the SQL to fetch failure events from the configured stream.
func BuildSQL(stream string, opts FetchOptions) string {
	if len(opts.Types) == 0 {
		opts.Types = AllTypes
	}
	if opts.Limit == 0 {
		opts.Limit = 200
	}

	var clauses []string

	for _, t := range opts.Types {
		switch t {
		case TypeToolError:
			clauses = append(clauses,
				`(event_name = 'claude_code.tool_result' AND json_extract(body, '$.success') = false)`)
		case TypeRefusal:
			clauses = append(clauses,
				`(event_name = 'claude_code.tool_decision' AND json_extract(body, '$.decision') = 'reject')`,
				`severityText = 'ERROR'`)
		case TypeToolAnomaly:
			clauses = append(clauses,
				`event_name IN ('claude_code.api_error', 'claude_code.api_retries_exhausted', 'claude_code.internal_error')`)
		}
	}

	where := fmt.Sprintf("(%s)", strings.Join(clauses, "\n    OR "))
	if opts.ProjectFilter != "" {
		where = fmt.Sprintf("project_name = '%s' AND %s", escapeSQLString(opts.ProjectFilter), where)
	}

	return fmt.Sprintf(`SELECT _timestamp, _id, session_id, project_name, event_name, tool_name,
       body, severityText,
       CAST(json_extract(body, '$.error_type') AS VARCHAR) AS error_type,
       CAST(json_extract(body, '$.success') AS VARCHAR) AS success,
       CAST(json_extract(body, '$.decision') AS VARCHAR) AS decision
FROM "%s"
WHERE service_name = 'claude_code'
  AND %s
ORDER BY _timestamp DESC
LIMIT %d`, stream, where, opts.Limit)
}

func FetchEvents(ctx context.Context, client *Client, stream string, opts FetchOptions) ([]map[string]any, error) {
	if opts.Since == 0 {
		opts.Since = 24 * time.Hour
	}
	end := time.Now()
	start := end.Add(-opts.Since)
	sql := BuildSQL(stream, opts)
	return client.Search(ctx, sql, start, end, opts.Limit)
}

// ParseDuration handles "1h", "24h", "7d" durations.
func ParseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		days := strings.TrimSuffix(s, "d")
		var n int
		if _, err := fmt.Sscanf(days, "%d", &n); err != nil {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q", s)
	}
	return d, nil
}

func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
