package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rchaganti/agent-plugin-validator/schema"
	"github.com/spf13/cobra"
)

var updateFormat string

var schemaUpdateCmd = &cobra.Command{
	Use:   "update [url]",
	Short: "Download and cache an updated JSON schema",
	Long: `Download a JSON schema from the canonical Agent Plugins URL (or a custom URL) 
and cache it locally in ~/.apv/schemas/plugin.schema.json.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		targetURL := ""
		if len(args) > 0 {
			targetURL = args[0]
		}

		mgr, err := schema.NewManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Schema manager error: %v\n", err)
			os.Exit(2)
		}

		info, err := mgr.Update(targetURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating schema: %v\n", err)
			os.Exit(2)
		}

		if updateFormat == "json" {
			out, _ := json.MarshalIndent(info, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Printf("✓ Schema successfully updated and cached.\n")
			fmt.Printf("  ID:   %s\n", info.ID)
			fmt.Printf("  Path: %s\n", info.Path)
		}
	},
}

func init() {
	schemaUpdateCmd.Flags().StringVarP(&updateFormat, "format", "f", "text", "Output format (text or json)")
	_ = schemaUpdateCmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"text", "json"}, cobra.ShellCompDirectiveNoFileComp
	})
	schemaCmd.AddCommand(schemaUpdateCmd)
}
