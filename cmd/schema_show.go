package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rchaganti/agent-plugin-validator/schema"
	"github.com/spf13/cobra"
)

var showFormat string
var showType string

var schemaShowCmd = &cobra.Command{
	Use:   "show [manifest|mcp]",
	Short: "Show details of active schema(s)",
	Long:  `Display title, $id, source, and file path of the active JSON schemas (manifest and mcp) used by apv.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		targetType := showType
		if len(args) > 0 {
			targetType = args[0]
		}

		mgr, err := schema.NewManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Schema manager error: %v\n", err)
			os.Exit(2)
		}

		var infos []*schema.SchemaInfo

		if targetType == schema.SchemaTypeManifest || targetType == schema.SchemaTypeMCP {
			info, err := mgr.Show(targetType)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting schema info for %s: %v\n", targetType, err)
				os.Exit(2)
			}
			infos = append(infos, info)
		} else {
			infoManifest, err := mgr.Show(schema.SchemaTypeManifest)
			if err == nil {
				infos = append(infos, infoManifest)
			}
			infoMCP, err := mgr.Show(schema.SchemaTypeMCP)
			if err == nil {
				infos = append(infos, infoMCP)
			}
		}

		if showFormat == "json" {
			if len(infos) == 1 {
				out, _ := json.MarshalIndent(infos[0], "", "  ")
				fmt.Println(string(out))
			} else {
				out, _ := json.MarshalIndent(infos, "", "  ")
				fmt.Println(string(out))
			}
		} else {
			if len(infos) == 1 {
				info := infos[0]
				fmt.Printf("Active Schema (%s):\n", info.Type)
				fmt.Printf("  Title:  %s\n", info.Title)
				fmt.Printf("  ID:     %s\n", info.ID)
				fmt.Printf("  Source: %s\n", info.Source)
				fmt.Printf("  Path:   %s\n", info.Path)
			} else {
				fmt.Println("Active Schemas:")
				for i, info := range infos {
					if i > 0 {
						fmt.Println()
					}
					fmt.Printf("[%s]\n", info.Type)
					fmt.Printf("  Title:  %s\n", info.Title)
					fmt.Printf("  ID:     %s\n", info.ID)
					fmt.Printf("  Source: %s\n", info.Source)
					fmt.Printf("  Path:   %s\n", info.Path)
				}
			}
		}
	},
}

func init() {
	schemaShowCmd.Flags().StringVarP(&showFormat, "format", "f", "text", "Output format (text or json)")
	schemaShowCmd.Flags().StringVarP(&showType, "type", "t", "all", "Schema type (all, manifest, or mcp)")

	_ = schemaShowCmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"text", "json"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = schemaShowCmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"all", "manifest", "mcp"}, cobra.ShellCompDirectiveNoFileComp
	})

	schemaCmd.AddCommand(schemaShowCmd)
}
