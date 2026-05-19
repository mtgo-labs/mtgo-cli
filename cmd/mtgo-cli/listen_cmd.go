package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	if invoke.IsMethodBlocked(payload.TLMethod) {
		return &ipc.Response{OK: false, Error: "method not allowed via IPC"}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var result *invoke.Result
	var err error
	if payload.Fast {
		result, err = invoke.InvokeFast(ctx, h.client, payload.TLMethod, payload.JSONParams)
	} else {
		result, err = invoke.InvokeFull(ctx, h.client, payload.TLMethod, payload.JSONParams)
	}
	if err != nil {
		return &ipc.Response{OK: false, Error: "internal error"}, nil
	}
	if result.Error != "" {
		return &ipc.Response{OK: false, Error: result.Error, DurMs: result.Duration.Milliseconds()}, nil
	}
	return &ipc.Response{OK: true, Data: json.RawMessage(result.RawJSON), DurMs: result.Duration.Milliseconds()}, nil
}

func (h *invokeHandler) HandleStatus() *ipc.Response {
	return &ipc.Response{OK: true, Data: map[string]bool{"connected": h.client.Me() != nil}}
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

			shutdown := make(chan os.Signal, 1)
			signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
			defer signal.Stop(shutdown)

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

			w := cmd.OutOrStdout()

			me := mtgoClient.Me()
			if me != nil {
				fmt.Fprintf(w, "Connected (ID: %d)\n", me.ID)
			} else {
				fmt.Fprintln(w, "Connected (anonymous)")
			}

			handler := &invokeHandler{client: mtgoClient}
			srv := ipc.NewServer(cfg.SocketPath, handler)
			if err := srv.Start(); err != nil {
				return fmt.Errorf("start IPC server: %w", err)
			}
			defer srv.Stop()

			fmt.Fprintf(w, "IPC server listening on %s\n", cfg.SocketPath)

			<-shutdown
			fmt.Fprintln(w, "\nShutting down...")
			return nil
		},
	}
	return cmd
}
