package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var Version = "1.1.0"

var (
	colorMode string
	useColor  bool
)

// Terminal color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

var rootCmd = &cobra.Command{
	Use:   "apv",
	Short: "apv - Agent Plugin Validator CLI",
	Long: `apv is a schema-driven CLI tool for validating Agent Plugin manifests (plugin.json) 
against the open Agent Plugins v1.0.0 specification.`,
	Version: Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		useColor = resolveColor(colorMode)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(2)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&colorMode, "color", "auto", "Control color output (auto, always, never)")
	_ = rootCmd.RegisterFlagCompletionFunc("color", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"auto", "always", "never"}, cobra.ShellCompDirectiveNoFileComp
	})

	rootCmd.SetVersionTemplate(fmt.Sprintf("apv (Agent Plugin Validator) v%s\n", Version))
}

func resolveColor(colorMode string) bool {
	switch strings.ToLower(colorMode) {
	case "always":
		return true
	case "never":
		return false
	case "auto", "":
		if os.Getenv("NO_COLOR") != "" {
			return false
		}
		fi, err := os.Stdout.Stat()
		if err != nil {
			return false
		}
		return (fi.Mode() & os.ModeCharDevice) != 0
	default:
		return false
	}
}
