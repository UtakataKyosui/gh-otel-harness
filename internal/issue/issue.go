package issue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/UtakataKyosui/gh-c2-harness/internal/classify"
	"github.com/UtakataKyosui/gh-c2-harness/internal/fingerprint"
	"github.com/cli/go-gh/v2/pkg/api"
)

const bodyExcerptMaxBytes = 2048

// BuildBody returns the Markdown body for a harness Issue.
func BuildBody(e *classify.Event, fp string) string {
	categoryLabel := string(e.Category)
	bodyExcerpt := e.Body
	if len(bodyExcerpt) > bodyExcerptMaxBytes {
		bodyExcerpt = bodyExcerpt[:bodyExcerptMaxBytes] + "\n... (truncated)"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## Summary\n%s\n\n", e.Title())
	fmt.Fprintf(&b, "## Detected category\n")
	for _, cat := range []string{"tool_error", "refusal", "tool_anomaly"} {
		check := " "
		if cat == categoryLabel {
			check = "x"
		}
		fmt.Fprintf(&b, "- [%s] %s\n", check, cat)
	}
	fmt.Fprintf(&b, "\n## Reproduction context\n")
	fmt.Fprintf(&b, "- session_id: `%s`\n", e.SessionID)
	fmt.Fprintf(&b, "- project: `%s`\n", e.ProjectName)
	fmt.Fprintf(&b, "- timestamp: `%s`\n", e.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(&b, "- tool_name: `%s`\n", e.ToolName)
	fmt.Fprintf(&b, "- event_name: `%s`\n", e.EventName)
	if e.ErrorType != "" {
		fmt.Fprintf(&b, "- error_type: `%s`\n", e.ErrorType)
	}
	fmt.Fprintf(&b, "\n## Raw body excerpt\n```json\n%s\n```\n", bodyExcerpt)
	fmt.Fprintf(&b, "\n## Harness TODO\n")
	fmt.Fprintf(&b, "- [ ] このケースを再現するテストを追加\n")
	fmt.Fprintf(&b, "- [ ] 再発防止のルール / フックを検討\n")
	fmt.Fprintf(&b, "\n<!-- gh-c2-harness:fingerprint:%s -->\n", fp)
	fmt.Fprintf(&b, "<!-- gh-c2-harness:event_id:%s -->\n", e.ID)
	return b.String()
}

type CreateResult struct {
	Number  int
	HTMLURL string
}

// Create opens a GitHub Issue in the harness repo.
func Create(client *api.RESTClient, harnessRepo string, e *classify.Event, labels []string, dryRun bool) (*CreateResult, string, error) {
	fp := fingerprint.Compute(e.EventName, e.ToolName, e.ErrorType, e.Body)
	title := e.Title()
	body := BuildBody(e, fp)

	if dryRun {
		out := fmt.Sprintf("# DRY RUN\nTitle: %s\n\n%s", title, body)
		return nil, out, nil
	}

	owner, repo, err := splitRepo(harnessRepo)
	if err != nil {
		return nil, "", err
	}

	payload := map[string]any{
		"title":  title,
		"body":   body,
		"labels": labels,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	var result struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
	}
	if err := client.Post(fmt.Sprintf("repos/%s/%s/issues", owner, repo), bytes.NewReader(b), &result); err != nil {
		return nil, "", fmt.Errorf("create issue: %w", err)
	}
	return &CreateResult{Number: result.Number, HTMLURL: result.HTMLURL}, "", nil
}

func splitRepo(repo string) (owner, name string, err error) {
	for i, c := range repo {
		if c == '/' {
			return repo[:i], repo[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("invalid repo %q: expected owner/repo", repo)
}
