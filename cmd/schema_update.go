package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rchaganti/agent-plugin-validator/schema"
	"github.com/spf13/cobra"
)

var updateFormat string
var updateType string

var schemaUpdateCmd = &cobra.Command{
	Use:   "update [manifest|mcp] [url]",
	Short: "Download and cache an updated JSON schema",
	Long: `Download a JSON schema from canonical Agent Plugins URL (or a custom URL) 
and cache it locally in ~/.apv/schemas/.`,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		targetType := updateType
		targetURL := ""

		if len(args) == 1 {
			if args[0] == "manifest" || args[0] == "mcp" {
				targetType = args[0]
			} else {
				targetURL = args[0]
			}
		} else if len(args) == 2 {
			targetType = args[0]
			targetURL = args[1]
		}

		if targetType != schema.SchemaTypeMCP {
			targetType = schema.SchemaTypeManifest
		}

		mgr, err := schema.NewManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Schema manager error: %v\n", err)
			os.Exit(2)
		}

		info, err := mgr.Update(targetType, targetURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating schema: %v\n", err)
			os.Exit(2)
		}

		if updateFormat == "json" {
			out, _ := json.MarshalIndent(info, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Printf("✓ Schema (%s) successfully updated and cached.\n", info.Type)
			fmt.Printf("  ID:   %s\n", info.ID)
			fmt.Printf("  Path: %s\n", info.Path)
		}
	},
}

func init() {
	schemaUpdateCmd.Flags().StringVarP(&updateFormat, "format", "f", "text", "Output format (text or json)")
	schemaUpdateCmd.Flags().StringVarP(&updateType, "type", "t", "manifest", "Schema type (manifest or mcp)")

	_ = schemaUpdateCmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"text", "json"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = schemaUpdateCmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"manifest", "mcp"}, cobra.ShellCompDirectiveNoFileComp
	})

	schemaCmd.AddCommand(schemaUpdateCmd)
}
