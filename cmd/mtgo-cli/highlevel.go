package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mtgo-labs/mtgo-cli/internal/client"
	"github.com/mtgo-labs/mtgo-cli/internal/config"
	"github.com/mtgo-labs/mtgo-cli/invoke"
	"github.com/mtgo-labs/mtgo-cli/ipc"
	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/tg"

	"github.com/spf13/cobra"
)

func connectOrIPC(cfg *config.Config, wantUpdates bool) (*telegram.Client, error) {
	c, err := client.New(&client.ClientConfig{
		APIID:       cfg.APIID,
		APIHash:     cfg.APIHash,
		BotToken:    cfg.BotToken,
		Session:     cfg.Session,
		Phone:       cfg.Phone,
		WantUpdates: wantUpdates,
	})
	if err != nil {
		return nil, err
	}
	if err := c.Connect(0); err != nil {
		return nil, err
	}
	return c, nil
}

func newGetMeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-me",
		Short: "Get current user/bot info",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cmd)
			if err != nil {
				return err
			}
			if ipc.IsSocketActive(cfg.SocketPath) {
				ipcClient := ipc.NewClient(cfg.SocketPath)
				resp, _ := ipcClient.Status()
				if resp != nil && resp.OK {
					prettyPrint(cfg.Format, resp.Data)
					return nil
				}
			}
			c, err := connectOrIPC(cfg, false)
			if err != nil {
				return err
			}
			defer c.Stop()
			me := c.Me()
			if me == nil {
				return fmt.Errorf("could not get current user info")
			}
			prettyPrint(cfg.Format, me)
			return nil
		},
	}
}

func newSendMessageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "send-message <peer> <text>",
		Short: "Send a text message",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cmd)
			if err != nil {
				return err
			}
			c, err := connectOrIPC(cfg, false)
			if err != nil {
				return err
			}
			defer c.Stop()

			ctx := context.Background()
			peer, err := invoke.ResolvePeer(ctx, c, args[0])
			if err != nil {
				return fmt.Errorf("resolve peer %q: %w", args[0], err)
			}
			peerJSON := peerToJSON(peer)
			randomID := fmt.Sprintf("%d", time.Now().UnixNano())
			msgJSON, _ := json.Marshal(args[1])
			params := []byte(fmt.Sprintf(`{"peer": %s, "message": %s, "random_id": %s}`, peerJSON, string(msgJSON), randomID))
			result, err := invoke.InvokeFull(ctx, c, "messages.sendMessage", params)
			if err != nil {
				return err
			}
			if result.Error != "" {
				return fmt.Errorf("RPC error: %s", result.Error)
			}
			if cfg.Format == "json" {
				prettyPrint(cfg.Format, result.Data)
			} else {
				fmt.Println("Message sent")
			}
			return nil
		},
	}
}

func newGetUserCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-user <peer>",
		Short: "Get user info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cmd)
			if err != nil {
				return err
			}
			c, err := connectOrIPC(cfg, false)
			if err != nil {
				return err
			}
			defer c.Stop()

			ctx := context.Background()
			peer, err := invoke.ResolvePeer(ctx, c, args[0])
			if err != nil {
				return fmt.Errorf("resolve peer: %w", err)
			}
			params := userToJSON(peer)
			result, err := invoke.InvokeFull(ctx, c, "users.getFullUser", []byte(fmt.Sprintf(`{"id": %s}`, params)))
			if err != nil {
				return err
			}
			if result.Error != "" {
				return fmt.Errorf("RPC error: %s", result.Error)
			}
			prettyPrint(cfg.Format, result.Data)
			return nil
		},
	}
}

func newGetChatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-chat <peer>",
		Short: "Get chat info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cmd)
			if err != nil {
				return err
			}
			c, err := connectOrIPC(cfg, false)
			if err != nil {
				return err
			}
			defer c.Stop()

			ctx := context.Background()
			peer, err := invoke.ResolvePeer(ctx, c, args[0])
			if err != nil {
				return fmt.Errorf("resolve peer: %w", err)
			}
			chatID := extractChatID(peer)
			result, err := invoke.InvokeFull(ctx, c, "messages.getFullChat", []byte(fmt.Sprintf(`{"chat_id": %d}`, chatID)))
			if err != nil {
				return err
			}
			if result.Error != "" {
				return fmt.Errorf("RPC error: %s", result.Error)
			}
			prettyPrint(cfg.Format, result.Data)
			return nil
		},
	}
}

func newListChatsCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "list-chats",
		Short: "List recent dialogs",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cmd)
			if err != nil {
				return err
			}
			c, err := connectOrIPC(cfg, false)
			if err != nil {
				return err
			}
			defer c.Stop()

			ctx := context.Background()
			params := fmt.Sprintf(`{"offset_date": 0, "offset_id": 0, "offset_peer": {"_":"inputPeerEmpty"}, "limit": %d, "hash": 0}`, limit)
			result, err := invoke.InvokeFull(ctx, c, "messages.getDialogs", []byte(params))
			if err != nil {
				return err
			}
			if result.Error != "" {
				return fmt.Errorf("RPC error: %s", result.Error)
			}
			prettyPrint(cfg.Format, result.Data)
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of dialogs to fetch")
	return cmd
}

func newListMessagesCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "list-messages <peer>",
		Short: "List recent messages",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cmd)
			if err != nil {
				return err
			}
			c, err := connectOrIPC(cfg, false)
			if err != nil {
				return err
			}
			defer c.Stop()

			ctx := context.Background()
			peer, err := invoke.ResolvePeer(ctx, c, args[0])
			if err != nil {
				return fmt.Errorf("resolve peer: %w", err)
			}
			peerJSON := peerToJSON(peer)
			result, err := invoke.InvokeFull(ctx, c, "messages.getHistory", []byte(fmt.Sprintf(`{"peer": %s, "limit": %d}`, peerJSON, limit)))
			if err != nil {
				return err
			}
			if result.Error != "" {
				return fmt.Errorf("RPC error: %s", result.Error)
			}
			prettyPrint(cfg.Format, result.Data)
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of messages to fetch")
	return cmd
}

func newResolvePeerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resolve-peer <identifier>",
		Short: "Resolve a peer identifier to access info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cmd)
			if err != nil {
				return err
			}
			c, err := connectOrIPC(cfg, false)
			if err != nil {
				return err
			}
			defer c.Stop()

			peer, err := invoke.ResolvePeer(context.Background(), c, args[0])
			if err != nil {
				return fmt.Errorf("resolve: %w", err)
			}
			fmt.Printf("Resolved %q -> %s\n", args[0], invoke.PeerString(peer))
			return nil
		},
	}
}

func newExportSessionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export-session",
		Short: "Export the current session as a string",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cmd)
			if err != nil {
				return err
			}
			c, err := connectOrIPC(cfg, false)
			if err != nil {
				return err
			}
			defer c.Stop()

			sessionStr, err := c.ExportSessionString()
			if err != nil {
				return fmt.Errorf("export session: %w", err)
			}
			fmt.Println(sessionStr)
			return nil
		},
	}
}

func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion <shell>",
		Short: "Generate shell completions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			default:
				return fmt.Errorf("unsupported shell: %s (use bash, zsh, or fish)", args[0])
			}
		},
	}
}

func userToJSON(peer tg.InputPeerClass) string {
	switch p := peer.(type) {
	case *tg.InputPeerSelf:
		return `{"_":"inputUserSelf"}`
	case *tg.InputPeerUser:
		return fmt.Sprintf(`{"_":"inputUser","user_id":%d,"access_hash":%d}`, p.UserID, p.AccessHash)
	default:
		return `{"_":"inputUserSelf"}`
	}
}

func peerToJSON(peer tg.InputPeerClass) string {
	switch p := peer.(type) {
	case *tg.InputPeerSelf:
		return `{"_":"inputPeerSelf"}`
	case *tg.InputPeerUser:
		return fmt.Sprintf(`{"_":"inputPeerUser","user_id":%d,"access_hash":%d}`, p.UserID, p.AccessHash)
	case *tg.InputPeerChannel:
		return fmt.Sprintf(`{"_":"inputPeerChannel","channel_id":%d,"access_hash":%d}`, p.ChannelID, p.AccessHash)
	default:
		return `{}`
	}
}

func extractChatID(peer tg.InputPeerClass) int64 {
	switch p := peer.(type) {
	case *tg.InputPeerChat:
		return p.ChatID
	case *tg.InputPeerChannel:
		return p.ChannelID
	default:
		return 0
	}
}

func prettyPrint(format string, data interface{}) {
	if format == "json" {
		out, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(out))
		return
	}
	fmt.Printf("%+v\n", data)
}
