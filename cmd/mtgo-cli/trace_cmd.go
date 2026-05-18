package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mtgo-labs/mtgo-cli/internal/client"
	"github.com/mtgo-labs/mtgo-cli/internal/config"
	"github.com/mtgo-labs/mtgo-cli/ipc"
	"github.com/mtgo-labs/mtgo-cli/trace"
	"github.com/spf13/cobra"
)

func newTraceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trace",
		Short: "Listen with correlation ID tracing",
		Long: `Like listen, but adds correlation IDs linking RPC calls and updates.

Output format:
  [1] >> messages.sendMessage
  [1]    {Message: "hello", ...}
  [1] << messages.sendMessage [12ms]
  [2] UPDATE updateNewMessage`,
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
			}

			tr := trace.NewTracer(os.Stdout)
			mtgoClient.UseInvokerMiddleware(tr.Middleware())
			mtgoClient.OnRawUpdate(tr.UpdateHandler())

			handler := &invokeHandler{client: mtgoClient}
			srv := ipc.NewServer(cfg.SocketPath, handler)
			if err := srv.Start(); err != nil {
				return fmt.Errorf("start IPC server: %w", err)
			}
			defer srv.Stop()

			fmt.Printf("Tracing started (socket: %s)\n", cfg.SocketPath)

			shutdown := make(chan os.Signal, 1)
			signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
			<-shutdown
			fmt.Println("\nShutting down...")
			return nil
		},
	}
	return cmd
}
