package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mtgo-cli",
	Short: "Telegram MTProto debug and invoke CLI",
	Long: `mtgo-cli — call any TL method, trace API calls, and manage Telegram sessions.

Built on mtgo (MTProto Go).`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().Int32("api-id", 0, "Telegram API ID")
	rootCmd.PersistentFlags().String("api-hash", "", "Telegram API Hash (prefer --api-hash-file)")
	rootCmd.PersistentFlags().String("api-hash-file", "", "Read Telegram API Hash from file")
	rootCmd.PersistentFlags().String("session", "", "Session string (prefer --session-file)")
	rootCmd.PersistentFlags().String("session-file", "", "Read session string from file")
	rootCmd.PersistentFlags().String("bot-token", "", "Bot token (prefer --bot-token-file)")
	rootCmd.PersistentFlags().String("bot-token-file", "", "Read bot token from file")
	rootCmd.PersistentFlags().String("phone", "", "Phone number for user login")
	rootCmd.PersistentFlags().String("socket", "", "Unix socket path for IPC")
	rootCmd.PersistentFlags().String("config", "", "Config file path (default: ~/.mtgo-cli.json)")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable verbose debug output")
	rootCmd.PersistentFlags().String("format", "text", "Output format: text, json")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(newInvokeCmd())
	rootCmd.AddCommand(newMethodsCmd())
	rootCmd.AddCommand(newListenCmd())
	rootCmd.AddCommand(newTraceCmd())
	rootCmd.AddCommand(newGetMeCmd())
	rootCmd.AddCommand(newSendMessageCmd())
	rootCmd.AddCommand(newSendPhotoCmd())
	rootCmd.AddCommand(newSendFileCmd())
	rootCmd.AddCommand(newDownloadCmd())
	rootCmd.AddCommand(newGetUserCmd())
	rootCmd.AddCommand(newGetChatCmd())
	rootCmd.AddCommand(newListChatsCmd())
	rootCmd.AddCommand(newListMessagesCmd())
	rootCmd.AddCommand(newCreateGroupCmd())
	rootCmd.AddCommand(newAddBotCmd())
	rootCmd.AddCommand(newPromoteBotCmd())
	rootCmd.AddCommand(newResolvePeerCmd())
	rootCmd.AddCommand(newExportSessionCmd())
	rootCmd.AddCommand(newCompletionCmd())
}
