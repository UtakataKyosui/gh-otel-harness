package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	OpenObserve OpenObserveConfig `toml:"openobserve"`
	Harness     HarnessConfig     `toml:"harness"`
	Query       QueryConfig       `toml:"query"`
}

type OpenObserveConfig struct {
	Endpoint string `toml:"endpoint"`
	Org      string `toml:"org"`
	Stream   string `toml:"stream"`
	Auth     string `toml:"auth"` // "Basic <base64>" — overridden by OO_AUTH env
}

type HarnessConfig struct {
	Repo          string   `toml:"repo"`
	DefaultLabels []string `toml:"default_labels"`
}

type QueryConfig struct {
	DefaultSince  string `toml:"default_since"`
	ProjectFilter string `toml:"project_filter"`
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "gh-otel-harness", "config.toml")
}

func Load() (*Config, error) {
	return LoadFrom(DefaultPath())
}

func LoadFrom(path string) (*Config, error) {
	cfg := &Config{
		OpenObserve: OpenObserveConfig{
			Endpoint: "http://localhost:5080",
			Org:      "default",
			Stream:   "default",
		},
		Harness: HarnessConfig{
			DefaultLabels: []string{"claude-code", "telemetry-derived"},
		},
		Query: QueryConfig{
			DefaultSince: "24h",
		},
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	// env overrides
	if v := os.Getenv("OO_AUTH"); v != "" {
		cfg.OpenObserve.Auth = v
	}
	if v := os.Getenv("HARNESS_REPO"); v != "" {
		cfg.Harness.Repo = v
	}

	return cfg, nil
}

func Save(cfg *Config) error {
	return SaveTo(cfg, DefaultPath())
}

func SaveTo(cfg *Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

func (c *Config) Validate() error {
	if c.OpenObserve.Auth == "" {
		return fmt.Errorf("openobserve.auth is required (run: gh otel-harness configure)")
	}
	if c.Harness.Repo == "" {
		return fmt.Errorf("harness.repo is required (run: gh otel-harness configure)")
	}
	return nil
}
