package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type Config struct {
	APIID      int32  `json:"api_id"`
	APIHash    string `json:"api_hash"`
	BotToken   string `json:"bot_token"`
	Session    string `json:"session"`
	Phone      string `json:"phone"`
	SocketPath string `json:"socket_path"`
	NoColor    bool   `json:"-"`
	Debug      bool   `json:"-"`
	Format     string `json:"-"`
}

func DefaultSocketPath() string {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "mtgo-cli.sock")
	}
	return filepath.Join(os.TempDir(), "mtgo-cli.sock")
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
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("config file %s: %w", configPath, err)
		}
	}

	// CLI flags override config
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

	// Env vars override everything
	cfg.overrideFromEnv()

	return cfg, cfg.Validate()
}

func (c *Config) overrideFromEnv() {
	if v := os.Getenv("MTGO_CLI_API_ID"); v != "" {
		fmt.Sscanf(v, "%d", &c.APIID)
	}
	if v := os.Getenv("MTGO_CLI_API_HASH"); v != "" {
		c.APIHash = v
	}
	if v := os.Getenv("MTGO_CLI_BOT_TOKEN"); v != "" {
		c.BotToken = v
	}
	if v := os.Getenv("MTGO_CLI_SESSION"); v != "" {
		c.Session = v
	}
	if v := os.Getenv("MTGO_CLI_PHONE"); v != "" {
		c.Phone = v
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
