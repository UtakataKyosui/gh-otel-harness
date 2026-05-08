package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/UtakataKyosui/gh-c2-harness/internal/classify"
	"github.com/UtakataKyosui/gh-c2-harness/internal/config"
	"github.com/UtakataKyosui/gh-c2-harness/internal/fingerprint"
	"github.com/UtakataKyosui/gh-c2-harness/internal/openobserve"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var (
		since      string
		types      string
		project    string
		harnessRepo string
		jsonOut    bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "失敗イベント候補を一覧表示する",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := cfg.Validate(); err != nil {
				return err
			}
			if harnessRepo != "" {
				cfg.Harness.Repo = harnessRepo
			}
			if since == "" {
				since = cfg.Query.DefaultSince
			}

			dur, err := openobserve.ParseDuration(since)
			if err != nil {
				return err
			}

			var typeList []string
			if types != "" {
				typeList = strings.Split(types, ",")
			}

			client := openobserve.NewClient(
				cfg.OpenObserve.Endpoint,
				cfg.OpenObserve.Org,
				cfg.OpenObserve.Stream,
				cfg.OpenObserve.Auth,
			)

			hits, err := openobserve.FetchEvents(context.Background(), client, cfg.OpenObserve.Stream, openobserve.FetchOptions{
				Since:         dur,
				Types:         typeList,
				ProjectFilter: project,
				Limit:         200,
			})
			if err != nil {
				return fmt.Errorf("fetch events: %w", err)
			}

			events := make([]*classify.Event, 0, len(hits))
			for _, h := range hits {
				events = append(events, classify.FromHit(h))
			}

			if jsonOut {
				type row struct {
					ID          string `json:"id"`
					Fingerprint string `json:"fingerprint"`
					Category    string `json:"category"`
					EventName   string `json:"event_name"`
					ToolName    string `json:"tool_name"`
					ErrorType   string `json:"error_type"`
					SessionID   string `json:"session_id"`
					Project     string `json:"project"`
					Timestamp   string `json:"timestamp"`
					Title       string `json:"title"`
				}
				rows := make([]row, len(events))
				for i, e := range events {
					rows[i] = row{
						ID:          e.ID,
						Fingerprint: fingerprint.Compute(e.EventName, e.ToolName, e.ErrorType, e.Body),
						Category:    string(e.Category),
						EventName:   e.EventName,
						ToolName:    e.ToolName,
						ErrorType:   e.ErrorType,
						SessionID:   e.SessionID,
						Project:     e.ProjectName,
						Timestamp:   e.Timestamp.Format("2006-01-02T15:04:05Z"),
						Title:       e.Title(),
					}
				}
				return json.NewEncoder(os.Stdout).Encode(rows)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "FINGERPRINT\tCATEGORY\tTIMESTAMP\tTITLE")
			for _, e := range events {
				fp := fingerprint.Compute(e.EventName, e.ToolName, e.ErrorType, e.Body)
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					fp,
					e.Category,
					e.Timestamp.Format("2006-01-02 15:04"),
					truncate(e.Title(), 60),
				)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&since, "since", "", "取得期間 (1h / 24h / 7d)")
	cmd.Flags().StringVar(&types, "type", "", "カテゴリフィルタ (tool_error,refusal,tool_anomaly)")
	cmd.Flags().StringVar(&project, "project", "", "project_name フィルタ")
	cmd.Flags().StringVar(&harnessRepo, "harness-repo", "", "起票先リポジトリ (owner/repo)")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "JSON 出力")
	return cmd
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
