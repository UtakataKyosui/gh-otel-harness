package openobserve

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	endpoint string
	org      string
	stream   string
	auth     string
	http     *http.Client
}

func NewClient(endpoint, org, stream, auth string) *Client {
	return &Client{
		endpoint: endpoint,
		org:      org,
		stream:   stream,
		auth:     auth,
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

type SearchRequest struct {
	Query SearchQuery `json:"query"`
}

type SearchQuery struct {
	SQL       string `json:"sql"`
	StartTime int64  `json:"start_time"` // microseconds
	EndTime   int64  `json:"end_time"`   // microseconds
	From      int    `json:"from"`
	Size      int    `json:"size"`
}

type SearchResponse struct {
	Hits  []map[string]any `json:"hits"`
	Total int              `json:"total"`
}

func (c *Client) Search(ctx context.Context, sql string, start, end time.Time, size int) ([]map[string]any, error) {
	req := SearchRequest{
		Query: SearchQuery{
			SQL:       sql,
			StartTime: start.UnixMicro(),
			EndTime:   end.UnixMicro(),
			From:      0,
			Size:      size,
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/%s/_search", c.endpoint, c.org)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", c.auth)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openobserve request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openobserve HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var sr SearchResponse
	if err := json.Unmarshal(raw, &sr); err != nil {
		return nil, fmt.Errorf("openobserve decode: %w", err)
	}
	return sr.Hits, nil
}
