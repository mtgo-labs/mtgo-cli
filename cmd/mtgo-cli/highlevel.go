package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/mtgo-labs/mtgo-cli/internal/client"
	"github.com/mtgo-labs/mtgo-cli/internal/config"
	"github.com/mtgo-labs/mtgo-cli/invoke"
	"github.com/mtgo-labs/mtgo-cli/ipc"
	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
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
					prettyPrint(cmd.OutOrStdout(), cfg.Format, resp.Data)
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
			prettyPrint(cmd.OutOrStdout(), cfg.Format, me)
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
			randomID := time.Now().UnixNano()
			rpc := c.Raw()
			result, err := rpc.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
				Peer:     peer,
				Message:  args[1],
				RandomID: randomID,
			})
			if err != nil {
				return fmt.Errorf("send message: %w", err)
			}
			if cfg.Format == "json" {
				prettyPrint(cmd.OutOrStdout(), cfg.Format, result)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Message sent")
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
			inputUser := peerToInputUser(peer)
			rpc := c.Raw()
			result, err := rpc.UsersGetFullUser(ctx, &tg.UsersGetFullUserRequest{
				ID: inputUser,
			})
			if err != nil {
				return fmt.Errorf("get user: %w", err)
			}
			prettyPrint(cmd.OutOrStdout(), cfg.Format, result)
			return nil
		},
	}
}

func newGetChatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-chat <peer>",
		Short: "Get chat/channel/user info",
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
			rpc := c.Raw()
			var result any
			switch p := peer.(type) {
			case *tg.InputPeerUser:
				result, err = rpc.UsersGetFullUser(ctx, &tg.UsersGetFullUserRequest{
					ID: &tg.InputUser{UserID: p.UserID, AccessHash: p.AccessHash},
				})
			case *tg.InputPeerSelf:
				result, err = rpc.UsersGetFullUser(ctx, &tg.UsersGetFullUserRequest{
					ID: &tg.InputUserSelf{},
				})
			case *tg.InputPeerChat:
				result, err = rpc.MessagesGetFullChat(ctx, &tg.MessagesGetFullChatRequest{
					ChatID: p.ChatID,
				})
			case *tg.InputPeerChannel:
				result, err = rpc.ChannelsGetFullChannel(ctx, &tg.ChannelsGetFullChannelRequest{
					Channel: &tg.InputChannel{ChannelID: p.ChannelID, AccessHash: p.AccessHash},
				})
			default:
				return fmt.Errorf("unsupported peer type: %T", peer)
			}
			if err != nil {
				return fmt.Errorf("get info: %w", err)
			}
			prettyPrint(cmd.OutOrStdout(), cfg.Format, result)
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
			rpc := c.Raw()
			result, err := rpc.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
				OffsetDate: 0,
				OffsetID:   0,
				OffsetPeer: &tg.InputPeerEmpty{},
				Limit:      int32(limit),
				Hash:       0,
			})
			if err != nil {
				return fmt.Errorf("get dialogs: %w", err)
			}
			prettyPrint(cmd.OutOrStdout(), cfg.Format, result)
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
			rpc := c.Raw()
			result, err := rpc.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
				Peer:       peer,
				Limit:      int32(limit),
				OffsetID:   0,
				OffsetDate: 0,
				AddOffset:  0,
				MaxID:      0,
				MinID:      0,
				Hash:       0,
			})
			if err != nil {
				return fmt.Errorf("get history: %w", err)
			}
			prettyPrint(cmd.OutOrStdout(), cfg.Format, result)
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of messages to fetch")
	return cmd
}

func newCreateGroupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create-group <title>",
		Short: "Create a basic group (userbot only)",
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
			rpc := c.Raw()

			result, err := rpc.MessagesCreateChat(ctx, &tg.MessagesCreateChatRequest{
				Users: []tg.InputUserClass{&tg.InputUserEmpty{}},
				Title: args[0],
			})
			if err != nil {
				return fmt.Errorf("create chat: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Group %q created\n", args[0])
			prettyPrint(cmd.OutOrStdout(), cfg.Format, result)
			return nil
		},
	}
}

func newSendPhotoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "send-photo <peer> <file> [caption]",
		Short: "Send a photo",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateUploadFile(args[1]); err != nil {
				return err
			}
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

			caption := ""
			if len(args) == 3 {
				caption = args[2]
			}

			msg, err := c.SendPhoto(ctx, extractChatID(peer), types.Path(args[1]), caption)
			if err != nil {
				return fmt.Errorf("send photo: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Photo sent (ID: %d)\n", msg.ID)
			return nil
		},
	}
}

func newSendFileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "send-file <peer> <file> [caption]",
		Short: "Send a document/file",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateUploadFile(args[1]); err != nil {
				return err
			}
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

			caption := ""
			if len(args) == 3 {
				caption = args[2]
			}

			msg, err := c.SendDocument(ctx, extractChatID(peer), types.Path(args[1]), caption)
			if err != nil {
				return fmt.Errorf("send file: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "File sent (ID: %d)\n", msg.ID)
			return nil
		},
	}
}

func validateUploadFile(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("%s: not a regular file (mode %s)", path, fi.Mode())
	}
	return nil
}

func newDownloadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "download <peer> <msg-id> [dest]",
		Short: "Download media from a message",
		Args:  cobra.RangeArgs(2, 3),
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
			msgID, err := parseMessageID(args[1])
			if err != nil {
				return err
			}

			msgs, err := c.GetMessages(ctx, extractChatID(peer), []int32{msgID})
			if err != nil {
				return fmt.Errorf("get message: %w", err)
			}
			if len(msgs) == 0 || msgs[0].Media == nil {
				return fmt.Errorf("message %d has no media", msgID)
			}

			dest := fmt.Sprintf("download_%d", msgID)
			if len(args) == 3 {
				dest = args[2]
			}

			if fi, err := os.Lstat(dest); err == nil {
				if fi.Mode()&os.ModeSymlink != 0 {
					return fmt.Errorf("refusing to write to symlink %s", dest)
				}
				if fi.IsDir() {
					return fmt.Errorf("destination %s is a directory", dest)
				}
			}

			err = c.DownloadMediaToFile(ctx, msgs[0].Media, "", dest, 0, &params.Download{DCID: 0})
			if err != nil {
				return fmt.Errorf("download: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Downloaded to %s\n", dest)
			return nil
		},
	}
}

func parseMessageID(s string) (int32, error) {
	id, err := strconv.ParseInt(s, 10, 32)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid message id %q", s)
	}
	return int32(id), nil
}

func newAddBotCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add-bot <group-peer> <bot-peer>",
		Short: "Add a bot to a group (userbot only)",
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
			var groupPeer tg.InputPeerClass
			var chatID int64

			if rawID, parseErr := strconv.ParseInt(args[0], 10, 64); parseErr == nil {
				chatID = rawID
			} else {
				groupPeer, err = invoke.ResolvePeer(ctx, c, args[0])
				if err != nil {
					return fmt.Errorf("resolve group: %w", err)
				}
			}

			botPeer, err := invoke.ResolvePeer(ctx, c, args[1])
			if err != nil {
				return fmt.Errorf("resolve bot: %w", err)
			}
			botUser := peerToInputUser(botPeer)
			rpc := c.Raw()

			if chatID != 0 {
				_, err = rpc.MessagesAddChatUser(ctx, &tg.MessagesAddChatUserRequest{
					ChatID:   chatID,
					UserID:   botUser,
					FwdLimit: 100,
				})
			} else {
				switch gp := groupPeer.(type) {
				case *tg.InputPeerChannel:
					_, err = rpc.ChannelsInviteToChannel(ctx, &tg.ChannelsInviteToChannelRequest{
						Channel: &tg.InputChannel{ChannelID: gp.ChannelID, AccessHash: gp.AccessHash},
						Users:   []tg.InputUserClass{botUser},
					})
				case *tg.InputPeerChat:
					_, err = rpc.MessagesAddChatUser(ctx, &tg.MessagesAddChatUserRequest{
						ChatID:   gp.ChatID,
						UserID:   botUser,
						FwdLimit: 100,
					})
				default:
					return fmt.Errorf("group peer must be a chat or channel, got %T", groupPeer)
				}
			}
			if err != nil {
				return fmt.Errorf("add bot: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Bot added to group")
			return nil
		},
	}
}

func newPromoteBotCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "promote-bot <channel-peer> <bot-peer>",
		Short: "Promote a bot to admin in a channel (userbot only)",
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
			channelPeer, err := invoke.ResolvePeer(ctx, c, args[0])
			if err != nil {
				return fmt.Errorf("resolve channel: %w", err)
			}
			botPeer, err := invoke.ResolvePeer(ctx, c, args[1])
			if err != nil {
				return fmt.Errorf("resolve bot: %w", err)
			}

			ch, ok := channelPeer.(*tg.InputPeerChannel)
			if !ok {
				return fmt.Errorf("promote-bot requires a channel/supergroup peer, got %T", channelPeer)
			}

			botUser := peerToInputUser(botPeer)
			rpc := c.Raw()
			_, err = rpc.ChannelsEditAdmin(ctx, &tg.ChannelsEditAdminRequest{
				Channel: &tg.InputChannel{ChannelID: ch.ChannelID, AccessHash: ch.AccessHash},
				UserID:  botUser,
				AdminRights: &tg.ChatAdminRights{
					ChangeInfo:     true,
					PostMessages:   true,
					EditMessages:   true,
					DeleteMessages: true,
					BanUsers:       true,
					InviteUsers:    true,
					PinMessages:    true,
					ManageTopics:   true,
				},
				Rank: "admin",
			})
			if err != nil {
				return fmt.Errorf("promote bot: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Bot promoted to admin")
			return nil
		},
	}
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
			fmt.Fprintf(cmd.OutOrStdout(), "Resolved %q -> %s\n", args[0], invoke.PeerString(peer))
			return nil
		},
	}
}

func newExportSessionCmd() *cobra.Command {
	var outputFile string
	cmd := &cobra.Command{
		Use:   "export-session",
		Short: "Export the current session as a string",
		Long: `Export the session string for the current Telegram account.

WARNING: The session string grants full account access. Anyone who obtains
it can impersonate your account. Handle with extreme care.`,
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

			if outputFile != "" {
				if err := os.WriteFile(outputFile, []byte(sessionStr+"\n"), 0600); err != nil {
					return fmt.Errorf("write session file: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Session written to %s (mode 0600)\n", outputFile)
				return nil
			}

			fmt.Fprintln(cmd.ErrOrStderr(), "WARNING: session string grants full account access. Pipe to a file or use --output.")
			fmt.Fprintln(cmd.OutOrStdout(), sessionStr)
			return nil
		},
	}
	cmd.Flags().StringVar(&outputFile, "output", "", "Write session to file (mode 0600) instead of stdout")
	return cmd
}

func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion <shell>",
		Short: "Generate shell completions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(w)
			case "zsh":
				return rootCmd.GenZshCompletion(w)
			case "fish":
				return rootCmd.GenFishCompletion(w, true)
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

func peerToInputUser(peer tg.InputPeerClass) tg.InputUserClass {
	switch p := peer.(type) {
	case *tg.InputPeerSelf:
		return &tg.InputUserSelf{}
	case *tg.InputPeerUser:
		return &tg.InputUser{UserID: p.UserID, AccessHash: p.AccessHash}
	default:
		return &tg.InputUserSelf{}
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
	case *tg.InputPeerUser:
		return p.UserID
	case *tg.InputPeerChat:
		return p.ChatID
	case *tg.InputPeerChannel:
		return p.ChannelID
	default:
		return 0
	}
}

func prettyPrint(w io.Writer, format string, data any) {
	if format == "json" {
		out, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			fmt.Fprintf(w, "error: marshal: %v\n", err)
			return
		}
		fmt.Fprintln(w, string(out))
		return
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(w, "%+v\n", data)
		return
	}
	fmt.Fprintln(w, string(out))
}
