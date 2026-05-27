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

			cfg, _ := config.Load(cmd)
			w := cmd.OutOrStdout()

			methods := invoke.FilterMethods(prefix)

			if cfg != nil && cfg.Format == "json" {
				out, err := json.MarshalIndent(methods, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal methods: %w", err)
				}
				fmt.Fprintln(w, string(out))
				return nil
			}

			for _, m := range methods {
				fmt.Fprintln(w, m)
			}
			fmt.Fprintf(w, "\n%d methods", len(methods))
			if prefix != "" {
				fmt.Fprintf(w, " matching '%s'", prefix)
			}
			fmt.Fprintln(w)
			return nil
		},
	}
	return cmd
}
