package dedupe

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
)

type ExistingIssue struct {
	Number int
	Title  string
	URL    string
	State  string
}

// FindByFingerprint searches the harness repo for an open issue with the given fingerprint.
// Returns nil if no duplicate is found.
func FindByFingerprint(client *api.RESTClient, harnessRepo, fingerprint string) (*ExistingIssue, error) {
	q := fmt.Sprintf("fingerprint:%s repo:%s in:body", fingerprint, harnessRepo)
	var result struct {
		TotalCount int `json:"total_count"`
		Items      []struct {
			Number  int    `json:"number"`
			Title   string `json:"title"`
			HTMLURL string `json:"html_url"`
			State   string `json:"state"`
		} `json:"items"`
	}

	if err := client.Get(
		fmt.Sprintf("search/issues?q=%s&per_page=1", encodeQuery(q)),
		&result,
	); err != nil {
		return nil, fmt.Errorf("search issues: %w", err)
	}

	if result.TotalCount == 0 || len(result.Items) == 0 {
		return nil, nil
	}

	item := result.Items[0]
	return &ExistingIssue{
		Number: item.Number,
		Title:  item.Title,
		URL:    item.HTMLURL,
		State:  item.State,
	}, nil
}

// AddComment appends a comment to an existing issue indicating a new occurrence.
func AddComment(client *api.RESTClient, harnessRepo string, issueNumber int, body string) error {
	owner, repo, err := splitRepo(harnessRepo)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("repos/%s/%s/issues/%d/comments", owner, repo, issueNumber)
	payload := map[string]string{"body": body}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var out any
	return client.Post(path, bytes.NewReader(b), &out)
}

func encodeQuery(q string) string {
	// percent-encode for URL query param
	var out []byte
	for i := 0; i < len(q); i++ {
		switch c := q[i]; {
		case c == ' ':
			out = append(out, '+')
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9',
			c == '-', c == '_', c == '.', c == '~', c == ':', c == '/':
			out = append(out, c)
		default:
			out = append(out, fmt.Sprintf("%%%02X", c)...)
		}
	}
	return string(out)
}

func splitRepo(repo string) (owner, name string, err error) {
	for i, c := range repo {
		if c == '/' {
			return repo[:i], repo[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("invalid repo %q: expected owner/repo", repo)
}
