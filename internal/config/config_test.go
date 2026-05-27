package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"valid bot", Config{APIID: 1, APIHash: "h", BotToken: "t"}, false},
		{"valid session", Config{APIID: 1, APIHash: "h", Session: "s"}, false},
		{"missing api-id", Config{APIHash: "h", BotToken: "t"}, true},
		{"missing api-hash", Config{APIID: 1, BotToken: "t"}, true},
		{"missing auth", Config{APIID: 1, APIHash: "h"}, true},
		{"bad format", Config{APIID: 1, APIHash: "h", BotToken: "t", Format: "xml"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultSocketPath(t *testing.T) {
	path := DefaultSocketPath()
	if path == "" {
		t.Error("DefaultSocketPath returned empty")
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("MTGO_CLI_API_ID", "123")
	t.Setenv("MTGO_CLI_API_HASH", "testhash")
	t.Setenv("MTGO_CLI_BOT_TOKEN", "testtoken")

	cmd := &cobra.Command{}
	cmd.Flags().String("config", "", "")
	cmd.Flags().Int32("api-id", 0, "")
	cmd.Flags().String("api-hash", "", "")
	cmd.Flags().String("bot-token", "", "")
	cmd.Flags().String("session", "", "")
	cmd.Flags().String("phone", "", "")
	cmd.Flags().String("socket", "", "")
	cmd.Flags().Bool("no-color", false, "")
	cmd.Flags().Bool("debug", false, "")
	cmd.Flags().String("format", "", "")

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIID != 123 {
		t.Errorf("APIID = %d, want 123", cfg.APIID)
	}
	if cfg.APIHash != "testhash" {
		t.Errorf("APIHash = %s, want testhash", cfg.APIHash)
	}
}

func TestConfigFilePath(t *testing.T) {
	dir := t.TempDir()
	tmpConfig := filepath.Join(dir, "config.json")
	os.WriteFile(tmpConfig, []byte(`{"api_id": 1, "api_hash": "h", "bot_token": "t"}`), 0600)

	cmd := &cobra.Command{}
	cmd.Flags().String("config", tmpConfig, "")
	cmd.Flags().Int32("api-id", 0, "")
	cmd.Flags().String("api-hash", "", "")
	cmd.Flags().String("bot-token", "", "")
	cmd.Flags().String("session", "", "")
	cmd.Flags().String("phone", "", "")
	cmd.Flags().String("socket", "", "")
	cmd.Flags().Bool("no-color", false, "")
	cmd.Flags().Bool("debug", false, "")
	cmd.Flags().String("format", "", "")

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BotToken != "t" {
		t.Errorf("BotToken = %s, want t", cfg.BotToken)
	}
}
