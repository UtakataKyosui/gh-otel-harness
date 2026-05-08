package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/UtakataKyosui/gh-c2-harness/internal/classify"
	"github.com/UtakataKyosui/gh-c2-harness/internal/config"
	"github.com/UtakataKyosui/gh-c2-harness/internal/dedupe"
	"github.com/UtakataKyosui/gh-c2-harness/internal/fingerprint"
	"github.com/UtakataKyosui/gh-c2-harness/internal/issue"
	"github.com/UtakataKyosui/gh-c2-harness/internal/openobserve"
	"github.com/UtakataKyosui/gh-c2-harness/internal/tui"
	"github.com/cli/go-gh/v2/pkg/api"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	var (
		since       string
		types       string
		project     string
		harnessRepo string
		noDedupe    bool
	)

	root := &cobra.Command{
		Use:   "c2-harness",
		Short: "Claude Code 失敗履歴を OpenObserve から取得してハーネスリポジトリに Issue を起票する",
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

			ooClient := openobserve.NewClient(
				cfg.OpenObserve.Endpoint,
				cfg.OpenObserve.Org,
				cfg.OpenObserve.Stream,
				cfg.OpenObserve.Auth,
			)
			hits, err := openobserve.FetchEvents(context.Background(), ooClient, cfg.OpenObserve.Stream, openobserve.FetchOptions{
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

			if len(events) == 0 {
				fmt.Fprintln(os.Stderr, "候補イベントが見つかりませんでした")
				return nil
			}

			// Run TUI
			m := tui.New(events)
			prog := tea.NewProgram(m, tea.WithAltScreen())
			finalModel, err := prog.Run()
			if err != nil {
				return err
			}
			final := finalModel.(*tui.Model)
			if len(final.Chosen) == 0 {
				fmt.Fprintln(os.Stderr, "選択なし — 終了します")
				return nil
			}

			ghClient, err := api.DefaultRESTClient()
			if err != nil {
				return fmt.Errorf("gh auth: %w", err)
			}

			created := 0
			skipped := 0
			for _, e := range final.Chosen {
				fp := fingerprint.Compute(e.EventName, e.ToolName, e.ErrorType, e.Body)

				if !noDedupe {
					existing, err := dedupe.FindByFingerprint(ghClient, cfg.Harness.Repo, fp)
					if err != nil {
						fmt.Fprintf(os.Stderr, "warn: dedup check for %s: %v\n", fp, err)
					} else if existing != nil {
						fmt.Fprintf(os.Stderr, "skip (dup) #%d: %s\n", existing.Number, existing.URL)
						skipped++
						continue
					}
				}

				result, _, err := issue.Create(ghClient, cfg.Harness.Repo, e, cfg.Harness.DefaultLabels, false)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error: create issue for %s: %v\n", fp, err)
					continue
				}
				fmt.Fprintf(os.Stdout, "Issue #%d: %s\n", result.Number, result.HTMLURL)
				created++
			}

			fmt.Fprintf(os.Stderr, "\n%d 件起票, %d 件スキップ\n", created, skipped)
			return nil
		},
	}

	root.Flags().StringVar(&since, "since", "", "取得期間 (1h / 24h / 7d)")
	root.Flags().StringVar(&types, "type", "", "カテゴリフィルタ (tool_error,refusal,tool_anomaly)")
	root.Flags().StringVar(&project, "project", "", "project_name フィルタ")
	root.Flags().StringVar(&harnessRepo, "harness-repo", "", "起票先リポジトリ (owner/repo)")
	root.Flags().BoolVar(&noDedupe, "no-dedupe", false, "fingerprint 重複チェックをスキップ")

	root.AddCommand(
		newListCmd(),
		newOpenCmd(),
		newPromptCmd(),
		newConfigureCmd(),
	)

	return root
}
