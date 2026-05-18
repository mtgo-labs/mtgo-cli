package main

import (
	"fmt"

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
	rootCmd.PersistentFlags().String("api-hash", "", "Telegram API Hash")
	rootCmd.PersistentFlags().String("session", "", "Session string (auto-detects format)")
	rootCmd.PersistentFlags().String("bot-token", "", "Bot token")
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
	rootCmd.AddCommand(newGetUserCmd())
	rootCmd.AddCommand(newGetChatCmd())
	rootCmd.AddCommand(newListChatsCmd())
	rootCmd.AddCommand(newListMessagesCmd())
	rootCmd.AddCommand(newResolvePeerCmd())
	rootCmd.AddCommand(newExportSessionCmd())
	rootCmd.AddCommand(newCompletionCmd())
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("mtgo-cli %s (commit %s, built %s)\n", version, commit, buildTime)
	},
}

// Stubs — replaced by actual implementations in later tasks
func newInvokeCmd() *cobra.Command       { return &cobra.Command{Use: "invoke"} }
func newMethodsCmd() *cobra.Command       { return &cobra.Command{Use: "methods"} }
func newListenCmd() *cobra.Command        { return &cobra.Command{Use: "listen"} }
func newTraceCmd() *cobra.Command         { return &cobra.Command{Use: "trace"} }
func newGetMeCmd() *cobra.Command         { return &cobra.Command{Use: "get-me"} }
func newSendMessageCmd() *cobra.Command   { return &cobra.Command{Use: "send-message"} }
func newGetUserCmd() *cobra.Command       { return &cobra.Command{Use: "get-user"} }
func newGetChatCmd() *cobra.Command       { return &cobra.Command{Use: "get-chat"} }
func newListChatsCmd() *cobra.Command     { return &cobra.Command{Use: "list-chats"} }
func newListMessagesCmd() *cobra.Command  { return &cobra.Command{Use: "list-messages"} }
func newResolvePeerCmd() *cobra.Command   { return &cobra.Command{Use: "resolve-peer"} }
func newExportSessionCmd() *cobra.Command { return &cobra.Command{Use: "export-session"} }
func newCompletionCmd() *cobra.Command    { return &cobra.Command{Use: "completion"} }
