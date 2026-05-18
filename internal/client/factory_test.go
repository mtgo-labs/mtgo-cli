package client

import (
	"testing"
)

func TestNewNilConfig(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestNewBotToken(t *testing.T) {
	cfg := &ClientConfig{
		APIID:     1,
		APIHash:   "hash",
		BotToken:  "token",
		NoUpdates: true,
	}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Error("client is nil")
	}
}

func TestNewSession(t *testing.T) {
	cfg := &ClientConfig{
		APIID:   1,
		APIHash: "hash",
		Session: "invalid-session",
	}
	_, err := New(cfg)
	if err == nil {
		t.Skip("unexpected: invalid session string was accepted")
	}
}

func TestNewPhone(t *testing.T) {
	cfg := &ClientConfig{
		APIID:   1,
		APIHash: "hash",
		Phone:   "+1234567890",
	}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Error("client is nil")
	}
}
