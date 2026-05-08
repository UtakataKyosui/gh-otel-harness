package cmd

import (
	"fmt"

	"github.com/UtakataKyosui/gh-otel-harness/internal/config"
	"github.com/UtakataKyosui/gh-otel-harness/internal/promptgen"
	"github.com/spf13/cobra"
)

func newPromptCmd() *cobra.Command {
	var harnessRepo string

	cmd := &cobra.Command{
		Use:   "prompt <event-id>",
		Short: "Claude Code 用の Markdown プロンプトを stdout に出力する",
		Long: `指定したイベント ID の失敗情報を含む Markdown プロンプトを生成し stdout に出力します。
Claude Code に渡すことで、ハーネステスト追加と Issue 起票を自動化できます。

例:
  gh otel-harness prompt abc123def456 | claude --print
  gh otel-harness prompt abc123def456 > /tmp/harness-prompt.md`,
		Args: cobra.ExactArgs(1),
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

			prompt := promptgen.Generate(e, cfg.Harness.Repo, cfg.Harness.DefaultLabels)
			fmt.Print(prompt)
			return nil
		},
	}

	cmd.Flags().StringVar(&harnessRepo, "harness-repo", "", "起票先リポジトリ (owner/repo)")
	return cmd
}
