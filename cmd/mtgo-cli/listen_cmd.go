package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mtgo-labs/mtgo-cli/internal/client"
	"github.com/mtgo-labs/mtgo-cli/internal/config"
	"github.com/mtgo-labs/mtgo-cli/invoke"
	"github.com/mtgo-labs/mtgo-cli/ipc"
	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/spf13/cobra"
)

type invokeHandler struct {
	client *telegram.Client
}

func (h *invokeHandler) HandleInvoke(payload ipc.InvokePayload) (*ipc.Response, error) {
	ctx := context.Background()
	result, err := invoke.InvokeFull(ctx, h.client, payload.TLMethod, payload.JSONParams)
	if err != nil {
		return &ipc.Response{OK: false, Error: err.Error()}, nil
	}
	if result.Error != "" {
		return &ipc.Response{OK: false, Error: result.Error, DurMs: result.Duration.Milliseconds()}, nil
	}
	return &ipc.Response{OK: true, Data: result.Data, DurMs: result.Duration.Milliseconds()}, nil
}

func (h *invokeHandler) HandleStatus() *ipc.Response {
	me := h.client.Me()
	data := map[string]interface{}{
		"connected": true,
	}
	if me != nil {
		data["user_id"] = me.ID
		data["first_name"] = me.FirstName
	}
	return &ipc.Response{OK: true, Data: data}
}

func newListenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "listen",
		Short: "Start persistent listener with IPC server",
		Long: `Start a persistent Telegram client that accepts invoke commands over IPC.

While listen is running, other mtgo-cli commands (invoke, get-me, etc.)
automatically route through the IPC socket.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cmd)
			if err != nil {
				return err
			}

			mtgoClient, err := client.New(&client.ClientConfig{
				APIID:       cfg.APIID,
				APIHash:     cfg.APIHash,
				BotToken:    cfg.BotToken,
				Session:     cfg.Session,
				Phone:       cfg.Phone,
				WantUpdates: true,
			})
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			if err := mtgoClient.Connect(0); err != nil {
				return fmt.Errorf("connect: %w", err)
			}
			defer mtgoClient.Stop()

			me := mtgoClient.Me()
			if me != nil {
				fmt.Printf("Connected as %s (ID: %d)\n", me.FirstName, me.ID)
			} else {
				fmt.Println("Connected (anonymous)")
			}

			handler := &invokeHandler{client: mtgoClient}
			srv := ipc.NewServer(cfg.SocketPath, handler)
			if err := srv.Start(); err != nil {
				return fmt.Errorf("start IPC server: %w", err)
			}
			defer srv.Stop()

			fmt.Printf("IPC server listening on %s\n", cfg.SocketPath)

			shutdown := make(chan os.Signal, 1)
			signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

			<-shutdown
			fmt.Println("\nShutting down...")
			return nil
		},
	}
	return cmd
}
