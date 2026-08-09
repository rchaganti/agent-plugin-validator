package cmd

import (
	"fmt"
	"os"

	"github.com/rchaganti/agent-plugin-validator/schema"
	"github.com/spf13/cobra"
)

var resetType string

var schemaResetCmd = &cobra.Command{
	Use:   "reset [manifest|mcp|all]",
	Short: "Revert to embedded default schema(s)",
	Long:  `Delete locally cached schema file(s), reverting apv to use embedded default schemas.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		targetType := resetType
		if len(args) > 0 {
			targetType = args[0]
		}
		if targetType == "all" {
			targetType = ""
		}

		mgr, err := schema.NewManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Schema manager error: %v\n", err)
			os.Exit(2)
		}

		if err := mgr.Reset(targetType); err != nil {
			fmt.Fprintf(os.Stderr, "Error resetting schema: %v\n", err)
			os.Exit(2)
		}

		if targetType == "" {
			fmt.Println("✓ All cached schemas reset. Reverted to embedded defaults.")
		} else {
			fmt.Printf("✓ Schema (%s) reset. Reverted to embedded default.\n", targetType)
		}
	},
}

func init() {
	schemaResetCmd.Flags().StringVarP(&resetType, "type", "t", "all", "Schema type (all, manifest, or mcp)")
	_ = schemaResetCmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"all", "manifest", "mcp"}, cobra.ShellCompDirectiveNoFileComp
	})
	schemaCmd.AddCommand(schemaResetCmd)
}
