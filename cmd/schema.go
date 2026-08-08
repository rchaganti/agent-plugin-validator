package cmd

import (
	"github.com/spf13/cobra"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Manage local JSON schemas",
	Long:  `Inspect, update, or reset the JSON schema used by apv for manifest validation.`,
}

func init() {
	rootCmd.AddCommand(schemaCmd)
}
