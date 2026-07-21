package client

import (
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram"
	tgconv "github.com/mtgo-labs/session-converter"
)

type ClientConfig struct {
	APIID       int32
	APIHash     string
	BotToken    string
	Session     string
	Phone       string
	NoUpdates   bool
	WantUpdates bool
}

func New(cfg *ClientConfig) (*telegram.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("client: config is nil")
	}

	telegramCfg := &telegram.Config{
		InMemory:  true,
		SavePeers: true,
	}

	if cfg.WantUpdates {
		telegramCfg.NoUpdates = false
	} else if cfg.NoUpdates {
		telegramCfg.NoUpdates = true
	}

	switch {
	case cfg.Session != "":
		str, err := tgconv.Convert(cfg.Session, tgconv.FormatTelethon)
		if err != nil {
			return nil, fmt.Errorf("client: invalid session string: %w", err)
		}
		telegramCfg.SessionString = str

	case cfg.BotToken != "":
		telegramCfg.BotToken = cfg.BotToken

	case cfg.Phone != "":
		telegramCfg.PhoneNumber = cfg.Phone
	}

	client, err := telegram.NewClient(cfg.APIID, cfg.APIHash, telegramCfg)
	if err != nil {
		return nil, fmt.Errorf("client: create: %w", err)
	}

	return client, nil
}
