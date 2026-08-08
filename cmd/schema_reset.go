package cmd

import (
	"fmt"
	"os"

	"github.com/rchaganti/agent-plugin-validator/schema"
	"github.com/spf13/cobra"
)

var schemaResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Revert to the embedded default schema",
	Long:  `Delete the locally cached schema, reverting apv to use the embedded default schema.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		mgr, err := schema.NewManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Schema manager error: %v\n", err)
			os.Exit(2)
		}

		if err := mgr.Reset(); err != nil {
			fmt.Fprintf(os.Stderr, "Error resetting schema: %v\n", err)
			os.Exit(2)
		}
		fmt.Println("✓ Schema reset. Reverted to embedded default schema.")
	},
}

func init() {
	schemaCmd.AddCommand(schemaResetCmd)
}
