package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/UtakataKyosui/gh-c2-harness/internal/config"
	"github.com/spf13/cobra"
)

func newConfigureCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "configure",
		Short: "対話的に ~/.config/gh-c2-harness/config.toml を初期化する",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				cfg = &config.Config{}
			}

			r := bufio.NewReader(os.Stdin)
			prompt := func(label, current string) (string, error) {
				if current != "" {
					fmt.Fprintf(os.Stderr, "%s [%s]: ", label, current)
				} else {
					fmt.Fprintf(os.Stderr, "%s: ", label)
				}
				line, err := r.ReadString('\n')
				if err != nil {
					return "", err
				}
				line = strings.TrimSpace(line)
				if line == "" {
					return current, nil
				}
				return line, nil
			}

			cfg.OpenObserve.Endpoint, err = prompt("OpenObserve endpoint", cfg.OpenObserve.Endpoint)
			if err != nil {
				return err
			}
			cfg.OpenObserve.Org, err = prompt("OpenObserve org", cfg.OpenObserve.Org)
			if err != nil {
				return err
			}
			cfg.OpenObserve.Stream, err = prompt("OpenObserve stream", cfg.OpenObserve.Stream)
			if err != nil {
				return err
			}
			cfg.OpenObserve.Auth, err = prompt(`Auth (e.g. "Basic <base64(email:password)>")`, cfg.OpenObserve.Auth)
			if err != nil {
				return err
			}
			cfg.Harness.Repo, err = prompt("Harness repo (owner/repo)", cfg.Harness.Repo)
			if err != nil {
				return err
			}
			cfg.Query.DefaultSince, err = prompt("Default since (e.g. 24h, 7d)", cfg.Query.DefaultSince)
			if err != nil {
				return err
			}

			path := config.DefaultPath()
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Fprintf(os.Stderr, "\nSaved to %s (chmod 600)\n", path)
			return nil
		},
	}
}
