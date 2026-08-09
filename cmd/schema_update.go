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
	Use:   "update [manifest|mcp|all] [url]",
	Short: "Download and cache updated JSON schema(s)",
	Long: `Download JSON schema(s) from canonical Agent Plugins URLs (or a custom URL) 
and cache them locally in ~/.apv/schemas/.`,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		targetType := updateType
		targetURL := ""

		if len(args) == 1 {
			if args[0] == "manifest" || args[0] == "mcp" || args[0] == "all" {
				targetType = args[0]
			} else {
				targetURL = args[0]
			}
		} else if len(args) == 2 {
			targetType = args[0]
			targetURL = args[1]
		}

		mgr, err := schema.NewManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Schema manager error: %v\n", err)
			os.Exit(2)
		}

		var updatedInfos []*schema.SchemaInfo

		if targetType == schema.SchemaTypeManifest || targetType == schema.SchemaTypeMCP {
			info, err := mgr.Update(targetType, targetURL)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error updating schema (%s): %v\n", targetType, err)
				os.Exit(2)
			}
			updatedInfos = append(updatedInfos, info)
		} else {
			// Update both schemas
			infoManifest, err := mgr.Update(schema.SchemaTypeManifest, "")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error updating manifest schema: %v\n", err)
				os.Exit(2)
			}
			updatedInfos = append(updatedInfos, infoManifest)

			infoMCP, err := mgr.Update(schema.SchemaTypeMCP, "")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error updating MCP schema: %v\n", err)
				os.Exit(2)
			}
			updatedInfos = append(updatedInfos, infoMCP)
		}

		if updateFormat == "json" {
			if len(updatedInfos) == 1 {
				out, _ := json.MarshalIndent(updatedInfos[0], "", "  ")
				fmt.Println(string(out))
			} else {
				out, _ := json.MarshalIndent(updatedInfos, "", "  ")
				fmt.Println(string(out))
			}
		} else {
			for _, info := range updatedInfos {
				fmt.Printf("✓ Schema (%s) successfully updated and cached.\n", info.Type)
				fmt.Printf("  ID:   %s\n", info.ID)
				fmt.Printf("  Path: %s\n", info.Path)
			}
		}
	},
}

func init() {
	schemaUpdateCmd.Flags().StringVarP(&updateFormat, "format", "f", "text", "Output format (text or json)")
	schemaUpdateCmd.Flags().StringVarP(&updateType, "type", "t", "all", "Schema type (all, manifest, or mcp)")

	_ = schemaUpdateCmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"text", "json"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = schemaUpdateCmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"all", "manifest", "mcp"}, cobra.ShellCompDirectiveNoFileComp
	})

	schemaCmd.AddCommand(schemaUpdateCmd)
}
