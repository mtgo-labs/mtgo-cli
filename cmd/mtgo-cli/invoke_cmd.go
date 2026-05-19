package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mtgo-labs/mtgo-cli/internal/client"
	"github.com/mtgo-labs/mtgo-cli/internal/config"
	"github.com/mtgo-labs/mtgo-cli/invoke"
	"github.com/mtgo-labs/mtgo-cli/ipc"
	"github.com/spf13/cobra"
)

func newInvokeCmd() *cobra.Command {
	var fast bool
	var timeout int
	var retries int

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

			// Try IPC first
			if ipc.IsSocketActive(cfg.SocketPath) {
				ipcClient := ipc.NewClient(cfg.SocketPath)
				resp, err := ipcClient.Invoke(ipc.InvokePayload{
					TLMethod:   method,
					JSONParams: jsonParams,
					Fast:       fast,
				})
				if err == nil && resp.OK {
					return formatOutput(cfg.Format, resp.Data)
				}
				if err == nil && !resp.OK {
					return fmt.Errorf("invoke failed: %s", resp.Error)
				}
				// IPC failed, fall through to standalone
			}

			// Standalone mode
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

			formatInvokeResult(cfg.Format, result)
			return nil
		},
	}

	cmd.Flags().BoolVar(&fast, "fast", false, "Use fast path (skip TL decode)")
	cmd.Flags().IntVar(&timeout, "timeout", 60, "Per-request timeout in seconds")
	cmd.Flags().IntVar(&retries, "retries", 3, "Retry count on transient errors")

	return cmd
}

func formatOutput(format string, data interface{}) error {
	if data == nil {
		return nil
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func formatInvokeResult(format string, result *invoke.Result) {
	if result.Error != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", result.Error)
		return
	}
	if format == "json" && result.RawJSON != nil {
		var pretty bytes.Buffer
		json.Indent(&pretty, result.RawJSON, "", "  ")
		fmt.Println(pretty.String())
		return
	}
	if result.RawBytes != nil {
		fmt.Printf("%x\n", result.RawBytes)
		return
	}
	if result.Data != nil {
		out, _ := json.MarshalIndent(result.Data, "", "  ")
		fmt.Println(string(out))
		return
	}
}
