package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type Config struct {
	APIID        int32     `json:"api_id"`
	APIHash      string    `json:"api_hash"`
	BotToken     string    `json:"bot_token"`
	Session      string    `json:"session"`
	Phone        string    `json:"phone"`
	SocketPath   string    `json:"socket_path"`
	NoColor      bool      `json:"-"`
	Debug        bool      `json:"-"`
	Format       string    `json:"-"`
	configPath   string
	configModTime time.Time
}

func DefaultSocketPath() string {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "mtgo-cli.sock")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "mtgo-cli.sock")
	}
	dir := filepath.Join(home, ".local", "run")
	os.MkdirAll(dir, 0700)
	return filepath.Join(dir, "mtgo-cli.sock")
}

func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".mtgo-cli.json")
}

func Load(cmd *cobra.Command) (*Config, error) {
	cfg := &Config{}

	configPath, _ := cmd.Flags().GetString("config")
	if configPath == "" {
		configPath = DefaultConfigPath()
	}

	if data, err := os.ReadFile(configPath); err == nil {
		if err := checkFilePerms(configPath); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
		if fi, err := os.Stat(configPath); err == nil {
			cfg.configPath = configPath
			cfg.configModTime = fi.ModTime()
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("config file: %w", err)
		}
	}

	if v, err := cmd.Flags().GetInt32("api-id"); err == nil && v != 0 {
		cfg.APIID = v
	}
	if v, err := cmd.Flags().GetString("api-hash"); err == nil && v != "" {
		cfg.APIHash = v
	}
	if v, err := cmd.Flags().GetString("bot-token"); err == nil && v != "" {
		cfg.BotToken = v
	}
	if v, err := cmd.Flags().GetString("session"); err == nil && v != "" {
		cfg.Session = v
	}

	if v, err := cmd.Flags().GetString("api-hash-file"); err == nil && v != "" {
		secret, err := readSecretFile(v)
		if err != nil {
			return nil, fmt.Errorf("--api-hash-file: %w", err)
		}
		cfg.APIHash = secret
	}
	if v, err := cmd.Flags().GetString("bot-token-file"); err == nil && v != "" {
		secret, err := readSecretFile(v)
		if err != nil {
			return nil, fmt.Errorf("--bot-token-file: %w", err)
		}
		cfg.BotToken = secret
	}
	if v, err := cmd.Flags().GetString("session-file"); err == nil && v != "" {
		secret, err := readSecretFile(v)
		if err != nil {
			return nil, fmt.Errorf("--session-file: %w", err)
		}
		cfg.Session = secret
	}
	if v, err := cmd.Flags().GetString("phone"); err == nil && v != "" {
		cfg.Phone = v
	}
	if v, err := cmd.Flags().GetString("socket"); err == nil && v != "" {
		cfg.SocketPath = v
	} else {
		cfg.SocketPath = DefaultSocketPath()
	}
	if v, err := cmd.Flags().GetBool("no-color"); err == nil {
		cfg.NoColor = v
	}
	if v, err := cmd.Flags().GetBool("debug"); err == nil {
		cfg.Debug = v
	}
	if v, err := cmd.Flags().GetString("format"); err == nil {
		cfg.Format = v
	}

	cfg.overrideFromEnv()

	return cfg, cfg.Validate()
}

func readSecretFile(path string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if fi.Mode().Perm() > 0600 {
		return "", fmt.Errorf("file %s has overly permissive mode %o (should be 0600)", filepath.Base(path), fi.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func checkFilePerms(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return nil
	}
	if fi.Mode().Perm() > 0600 {
		return fmt.Errorf("config file %s has overly permissive mode %o (should be 0600)", filepath.Base(path), fi.Mode().Perm())
	}
	return nil
}

func (c *Config) overrideFromEnv() {
	if v := os.Getenv("MTGO_CLI_API_ID"); v != "" {
		fmt.Sscanf(v, "%d", &c.APIID)
		os.Unsetenv("MTGO_CLI_API_ID")
	}
	if v := os.Getenv("MTGO_CLI_API_HASH"); v != "" {
		c.APIHash = v
		os.Unsetenv("MTGO_CLI_API_HASH")
	}
	if v := os.Getenv("MTGO_CLI_BOT_TOKEN"); v != "" {
		c.BotToken = v
		os.Unsetenv("MTGO_CLI_BOT_TOKEN")
	}
	if v := os.Getenv("MTGO_CLI_SESSION"); v != "" {
		c.Session = v
		os.Unsetenv("MTGO_CLI_SESSION")
	}
	if v := os.Getenv("MTGO_CLI_PHONE"); v != "" {
		c.Phone = v
		os.Unsetenv("MTGO_CLI_PHONE")
	}
}

func (c *Config) Validate() error {
	if c.APIID == 0 {
		return fmt.Errorf("api-id is required (set via --api-id, MTGO_CLI_API_ID, or config file)")
	}
	if c.APIHash == "" {
		return fmt.Errorf("api-hash is required (set via --api-hash, MTGO_CLI_API_HASH, or config file)")
	}
	if c.BotToken == "" && c.Session == "" && c.Phone == "" {
		return fmt.Errorf("one of --bot-token, --session, or --phone is required")
	}
	if c.Format != "" && c.Format != "text" && c.Format != "json" {
		return fmt.Errorf("--format must be 'text' or 'json', got: %s", c.Format)
	}
	if c.Format == "" {
		c.Format = "text"
	}
	return nil
}

func (c *Config) HasAuth() bool {
	return c.Session != "" || c.BotToken != "" || c.Phone != ""
}

func (c *Config) CheckIntegrity() error {
	if c.configPath == "" || c.configModTime.IsZero() {
		return nil
	}
	fi, err := os.Stat(c.configPath)
	if err != nil {
		return nil
	}
	if fi.ModTime().After(c.configModTime) {
		return fmt.Errorf("config file %s was modified after loading (possible tampering)", filepath.Base(c.configPath))
	}
	return nil
}
