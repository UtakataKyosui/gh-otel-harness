package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/UtakataKyosui/gh-otel-harness/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newConfigureCmd() *cobra.Command {
	var (
		endpoint    string
		org         string
		stream      string
		auth        string
		harnessRepo string
		since       string
	)

	cmd := &cobra.Command{
		Use:   "configure",
		Short: "~/.config/gh-otel-harness/config.toml を初期化する",
		Long: `設定ファイルを初期化します。フラグで値を渡すと非対話的に動作します。

例（非対話的）:
  gh otel-harness configure \
    --auth "Basic dXNlcjpwYXNz" \
    --harness-repo owner/claude-harness

例（対話的、TTY 必須）:
  gh otel-harness configure`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				cfg = &config.Config{}
			}

			// フラグで渡された値を即適用
			if endpoint != "" {
				cfg.OpenObserve.Endpoint = endpoint
			}
			if org != "" {
				cfg.OpenObserve.Org = org
			}
			if stream != "" {
				cfg.OpenObserve.Stream = stream
			}
			if auth != "" {
				cfg.OpenObserve.Auth = auth
			}
			if harnessRepo != "" {
				cfg.Harness.Repo = harnessRepo
			}
			if since != "" {
				cfg.Query.DefaultSince = since
			}

			// フラグが全部揃っていれば対話不要
			allProvided := endpoint != "" && auth != "" && harnessRepo != ""
			if !allProvided {
				// TTY チェック：非対話環境では省略してエラーにしない
				if !term.IsTerminal(int(os.Stdin.Fd())) {
					fmt.Fprintln(os.Stderr, "非対話環境では --endpoint / --auth / --harness-repo を指定してください")
					fmt.Fprintln(os.Stderr, "例: gh otel-harness configure --auth 'Basic ...' --harness-repo owner/repo")
					return fmt.Errorf("missing required flags in non-interactive mode")
				}
				// 対話モード: 未設定の項目だけ聞く
				r := bufio.NewReader(os.Stdin)
				prompt := func(label, current string) (string, error) {
					if current != "" {
						fmt.Fprintf(os.Stderr, "%s [%s]: ", label, current)
					} else {
						fmt.Fprintf(os.Stderr, "%s: ", label)
					}
					line, err := r.ReadString('\n')
					if err != nil {
						return current, nil // EOF のときは既存値をそのまま使う
					}
					line = strings.TrimSpace(line)
					if line == "" {
						return current, nil
					}
					return line, nil
				}

				cfg.OpenObserve.Endpoint, _ = prompt("OpenObserve endpoint", cfg.OpenObserve.Endpoint)
				cfg.OpenObserve.Org, _ = prompt("OpenObserve org", cfg.OpenObserve.Org)
				cfg.OpenObserve.Stream, _ = prompt("OpenObserve stream", cfg.OpenObserve.Stream)
				cfg.OpenObserve.Auth, _ = prompt(`Auth (e.g. "Basic <base64>")`, cfg.OpenObserve.Auth)
				cfg.Harness.Repo, _ = prompt("Harness repo (owner/repo)", cfg.Harness.Repo)
				cfg.Query.DefaultSince, _ = prompt("Default since (e.g. 24h, 7d)", cfg.Query.DefaultSince)
			}

			path := config.DefaultPath()
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Saved to %s\n", path)

			// 保存内容を確認表示
			fmt.Fprintf(os.Stderr, "\n[openobserve]\n")
			fmt.Fprintf(os.Stderr, "  endpoint = %s\n", cfg.OpenObserve.Endpoint)
			fmt.Fprintf(os.Stderr, "  org      = %s\n", cfg.OpenObserve.Org)
			fmt.Fprintf(os.Stderr, "  stream   = %s\n", cfg.OpenObserve.Stream)
			fmt.Fprintf(os.Stderr, "  auth     = %s...\n", cfg.OpenObserve.Auth[:min(len(cfg.OpenObserve.Auth), 12)])
			fmt.Fprintf(os.Stderr, "[harness]\n")
			fmt.Fprintf(os.Stderr, "  repo     = %s\n", cfg.Harness.Repo)
			fmt.Fprintf(os.Stderr, "[query]\n")
			fmt.Fprintf(os.Stderr, "  since    = %s\n", cfg.Query.DefaultSince)
			return nil
		},
	}

	cmd.Flags().StringVar(&endpoint, "endpoint", "", "OpenObserve endpoint (デフォルト: http://localhost:5080)")
	cmd.Flags().StringVar(&org, "org", "", "OpenObserve org (デフォルト: default)")
	cmd.Flags().StringVar(&stream, "stream", "", "OpenObserve stream (デフォルト: default)")
	cmd.Flags().StringVar(&auth, "auth", "", `認証ヘッダー値 (例: "Basic <base64(email:password)>")`)
	cmd.Flags().StringVar(&harnessRepo, "harness-repo", "", "起票先リポジトリ (owner/repo)")
	cmd.Flags().StringVar(&since, "since", "", "デフォルト取得期間 (例: 24h, 7d)")
	return cmd
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
