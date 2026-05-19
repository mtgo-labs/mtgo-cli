package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/mtgo-labs/mtgo-cli/internal/client"
	"github.com/mtgo-labs/mtgo-cli/internal/config"
	"github.com/mtgo-labs/mtgo-cli/invoke"
	"github.com/mtgo-labs/mtgo-cli/ipc"
	"github.com/spf13/cobra"
)

func newInvokeCmd() *cobra.Command {
	var fast bool
	var maxBytes int

	cmd := &cobra.Command{
		Use:   "invoke <method> [json-params]",
		Short: "Invoke a TL method",
		Long: `Invoke any Telegram TL method with JSON parameters.

Interface fields use a "_" key to specify the constructor:
  {"peer": {"_": "inputPeerUser", "user_id": 123, "access_hash": 456}}

If a listener is running, the invoke routes through the IPC socket.
Otherwise it creates a standalone connection.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			method := args[0]
			var jsonParams []byte
			if len(args) == 2 {
				jsonParams = []byte(args[1])
			} else {
				jsonParams = []byte("{}")
			}

			cfg, err := config.Load(cmd)
			if err != nil {
				return err
			}

			ctx := context.Background()

			if ipc.IsSocketActive(cfg.SocketPath) {
				ipcClient := ipc.NewClient(cfg.SocketPath)
				resp, err := ipcClient.Invoke(ipc.InvokePayload{
					TLMethod:   method,
					JSONParams: jsonParams,
					Fast:       fast,
				})
				if err == nil && resp.OK {
					return formatOutput(cmd.OutOrStdout(), cfg.Format, resp.Data)
				}
				if err == nil && !resp.OK {
					return fmt.Errorf("invoke failed: %s", resp.Error)
				}
			}

			mtgoClient, err := client.New(&client.ClientConfig{
				APIID:     cfg.APIID,
				APIHash:   cfg.APIHash,
				BotToken:  cfg.BotToken,
				Session:   cfg.Session,
				Phone:     cfg.Phone,
				NoUpdates: true,
			})
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}
			if err := mtgoClient.Connect(0); err != nil {
				return fmt.Errorf("connect: %w", err)
			}
			defer mtgoClient.Stop()

			var result *invoke.Result
			if fast {
				result, err = invoke.InvokeFast(ctx, mtgoClient, method, jsonParams)
			} else {
				result, err = invoke.InvokeFull(ctx, mtgoClient, method, jsonParams)
			}
			if err != nil {
				return err
			}

			if result.Error != "" {
				return fmt.Errorf("RPC error: %s", result.Error)
			}

			formatInvokeResult(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg.Format, result, maxBytes)
			return nil
		},
	}

	cmd.Flags().BoolVar(&fast, "fast", false, "Use fast path (skip TL decode)")
	cmd.Flags().IntVar(&maxBytes, "max-bytes", 256, "Max raw bytes to display (0 = unlimited)")
	return cmd
}

func formatOutput(w io.Writer, format string, data interface{}) error {
	if data == nil {
		return nil
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(w, string(out))
	return nil
}

func formatInvokeResult(w io.Writer, errW io.Writer, format string, result *invoke.Result, maxBytes int) {
	if result.Error != "" {
		fmt.Fprintf(errW, "Error: %s\n", result.Error)
		return
	}
	if format == "json" && result.RawJSON != nil {
		var pretty bytes.Buffer
		json.Indent(&pretty, result.RawJSON, "", "  ")
		fmt.Fprintln(w, pretty.String())
		return
	}
	if result.RawBytes != nil {
		raw := result.RawBytes
		truncated := false
		if maxBytes > 0 && len(raw) > maxBytes {
			raw = raw[:maxBytes]
			truncated = true
		}
		fmt.Fprintf(w, "%x\n", raw)
		if truncated {
			fmt.Fprintf(w, "... (%d bytes truncated, use --max-bytes=0 for full output)\n", len(result.RawBytes)-maxBytes)
		}
		return
	}
	if result.Data != nil {
		out, _ := json.MarshalIndent(result.Data, "", "  ")
		fmt.Fprintln(w, string(out))
		return
	}
}
