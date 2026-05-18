package main

import (
	"encoding/json"
	"fmt"

	"github.com/mtgo-labs/mtgo-cli/internal/config"
	"github.com/mtgo-labs/mtgo-cli/invoke"
	"github.com/spf13/cobra"
)

func newMethodsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "methods [prefix]",
		Short: "List available TL methods",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prefix := ""
			if len(args) == 1 {
				prefix = args[0]
			}

			cfg, err := config.Load(cmd)
			if err != nil {
				_ = cfg
			}

			methods := invoke.FilterMethods(prefix)

			if cfg != nil && cfg.Format == "json" {
				out, _ := json.MarshalIndent(methods, "", "  ")
				fmt.Println(string(out))
				return nil
			}

			for _, m := range methods {
				fmt.Println(m)
			}
			fmt.Printf("\n%d methods", len(methods))
			if prefix != "" {
				fmt.Printf(" matching '%s'", prefix)
			}
			fmt.Println()
			return nil
		},
	}
	return cmd
}
