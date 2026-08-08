package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rchaganti/agent-plugin-validator/schema"
	"github.com/spf13/cobra"
)

var showFormat string

var schemaShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show details of the active schema",
	Long:  `Display title, $id, source, and file path of the active JSON schema used by apv.`,
	Run: func(cmd *cobra.Command, args []string) {
		mgr, err := schema.NewManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Schema manager error: %v\n", err)
			os.Exit(2)
		}

		info, err := mgr.Show()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting schema info: %v\n", err)
			os.Exit(2)
		}

		if showFormat == "json" {
			out, _ := json.MarshalIndent(info, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Printf("Active Schema:\n")
			fmt.Printf("  Title:  %s\n", info.Title)
			fmt.Printf("  ID:     %s\n", info.ID)
			fmt.Printf("  Source: %s\n", info.Source)
			fmt.Printf("  Path:   %s\n", info.Path)
		}
	},
}

func init() {
	schemaShowCmd.Flags().StringVarP(&showFormat, "format", "f", "text", "Output format (text or json)")
	_ = schemaShowCmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"text", "json"}, cobra.ShellCompDirectiveNoFileComp
	})
	schemaCmd.AddCommand(schemaShowCmd)
}
