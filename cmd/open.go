package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/UtakataKyosui/gh-c2-harness/internal/classify"
	"github.com/UtakataKyosui/gh-c2-harness/internal/config"
	"github.com/UtakataKyosui/gh-c2-harness/internal/dedupe"
	"github.com/UtakataKyosui/gh-c2-harness/internal/fingerprint"
	"github.com/UtakataKyosui/gh-c2-harness/internal/issue"
	"github.com/UtakataKyosui/gh-c2-harness/internal/openobserve"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/spf13/cobra"
)

func newOpenCmd() *cobra.Command {
	var (
		harnessRepo string
		dryRun      bool
		noDedupe    bool
	)

	cmd := &cobra.Command{
		Use:   "open <event-id>",
		Short: "イベント ID を指定して Issue を起票する",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eventID := args[0]

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

			e, err := fetchEventByID(cfg, eventID)
			if err != nil {
				return err
			}
			if e == nil {
				return fmt.Errorf("event %q not found in OpenObserve", eventID)
			}

			fp := fingerprint.Compute(e.EventName, e.ToolName, e.ErrorType, e.Body)

			ghClient, err := api.DefaultRESTClient()
			if err != nil {
				return fmt.Errorf("gh auth: %w", err)
			}

			if !noDedupe && !dryRun {
				existing, err := dedupe.FindByFingerprint(ghClient, cfg.Harness.Repo, fp)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warn: dedup check failed: %v\n", err)
				} else if existing != nil {
					fmt.Fprintf(os.Stderr, "既存 Issue が見つかりました: #%d %s\n%s\n", existing.Number, existing.Title, existing.URL)
					fmt.Fprintln(os.Stderr, "スキップします (--no-dedupe で強制起票)")
					return nil
				}
			}

			result, dryOutput, err := issue.Create(ghClient, cfg.Harness.Repo, e, cfg.Harness.DefaultLabels, dryRun)
			if err != nil {
				return err
			}
			if dryRun {
				fmt.Println(dryOutput)
				return nil
			}
			fmt.Fprintf(os.Stdout, "Issue #%d を作成しました: %s\n", result.Number, result.HTMLURL)
			return nil
		},
	}

	cmd.Flags().StringVar(&harnessRepo, "harness-repo", "", "起票先リポジトリ (owner/repo)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Issue を作らず title/body を stdout に出す")
	cmd.Flags().BoolVar(&noDedupe, "no-dedupe", false, "fingerprint 重複チェックをスキップ")
	return cmd
}

func fetchEventByID(cfg *config.Config, id string) (*classify.Event, error) {
	client := openobserve.NewClient(
		cfg.OpenObserve.Endpoint,
		cfg.OpenObserve.Org,
		cfg.OpenObserve.Stream,
		cfg.OpenObserve.Auth,
	)

	sql := fmt.Sprintf(`SELECT * FROM "%s" WHERE _id = '%s' LIMIT 1`,
		cfg.OpenObserve.Stream, escapeSingleQuote(id))

	now := time.Now()
	start := now.Add(-30 * 24 * time.Hour)

	hits, err := client.Search(context.Background(), sql, start, now, 1)
	if err != nil {
		return nil, err
	}
	if len(hits) == 0 {
		return nil, nil
	}
	return classify.FromHit(hits[0]), nil
}

func escapeSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
