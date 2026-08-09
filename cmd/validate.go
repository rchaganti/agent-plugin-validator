package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/rchaganti/agent-plugin-validator/schema"
	"github.com/rchaganti/agent-plugin-validator/validator"
	"github.com/spf13/cobra"
)

var (
	schemaOverride string
	schemaTypeFlag string
	outputFormat   string
	quietMode      bool
)

var validateCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate a plugin.json or mcp.json file against JSON schema",
	Long: `Validate an Agent Plugin manifest (plugin.json) or MCP configuration (mcp.json) 
against the canonical Agent Plugins v1.0.0 JSON schema.
Pass '-' as the file argument to read from standard input.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		manifestPath := args[0]

		// 1. Read manifest input
		var manifestData []byte
		var err error

		if manifestPath == "-" {
			manifestData, err = io.ReadAll(os.Stdin)
			if err != nil {
				if !quietMode {
					fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
				}
				os.Exit(2)
			}
		} else {
			manifestData, err = os.ReadFile(manifestPath)
			if err != nil {
				if !quietMode {
					fmt.Fprintf(os.Stderr, "Error reading file '%s': %v\n", manifestPath, err)
				}
				os.Exit(2)
			}
		}

		// 2. Resolve schema
		mgr, err := schema.NewManager()
		if err != nil {
			if !quietMode {
				fmt.Fprintf(os.Stderr, "Schema manager initialization error: %v\n", err)
			}
			os.Exit(2)
		}

		targetType := schemaTypeFlag
		if targetType == "" || targetType == "auto" {
			targetType = schema.DetectType(manifestData, manifestPath)
		}

		schemaInfo, err := mgr.Resolve(targetType, schemaOverride)
		if err != nil {
			if !quietMode {
				fmt.Fprintf(os.Stderr, "Error resolving schema: %v\n", err)
			}
			os.Exit(2)
		}

		// 3. Validate manifest
		result, err := validator.Validate(manifestData, schemaInfo)
		if err != nil {
			if !quietMode {
				fmt.Fprintf(os.Stderr, "Runtime validation error: %v\n", err)
			}
			os.Exit(2)
		}

		// 4. Output results
		if quietMode {
			if result.Valid {
				os.Exit(0)
			} else {
				os.Exit(1)
			}
		}

		if outputFormat == "json" {
			outJSON, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(outJSON))
		} else {
			printTextValidationResult(result, useColor, manifestPath)
		}

		if result.Valid {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	},
}

func init() {
	validateCmd.Flags().StringVarP(&schemaOverride, "schema", "s", "", "Custom schema override (path or URL)")
	validateCmd.Flags().StringVarP(&schemaTypeFlag, "type", "t", "auto", "Schema type: auto, manifest, mcp")
	validateCmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text or json)")
	validateCmd.Flags().BoolVarP(&quietMode, "quiet", "q", false, "Quiet mode (suppress output, exit code only)")

	// Shell autocompletion setup
	_ = validateCmd.MarkFlagFilename("schema", "json")
	_ = validateCmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"auto", "manifest", "mcp"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = validateCmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"text", "json"}, cobra.ShellCompDirectiveNoFileComp
	})

	rootCmd.AddCommand(validateCmd)
}

func printTextValidationResult(res *validator.Result, useColor bool, manifestPath string) {
	cCheck := "✓"
	cX := "✗"
	cWarn := "⚠"
	cBold := ""
	cReset := ""
	cRed := ""
	cYellow := ""
	cCyan := ""

	if useColor {
		cCheck = colorGreen + "✓" + colorReset
		cX = colorRed + "✗" + colorReset
		cWarn = colorYellow + "⚠" + colorReset
		cBold = colorBold
		cReset = colorReset
		cRed = colorRed
		cYellow = colorYellow
		cCyan = colorCyan
	}

	schemaDesc := fmt.Sprintf("%s (%s)", res.Schema.Title, res.Schema.Source)
	if res.Schema.Path != "" && res.Schema.Path != "(embedded)" {
		schemaDesc = fmt.Sprintf("%s (%s: %s)", res.Schema.Title, res.Schema.Source, res.Schema.Path)
	}

	fmt.Printf("%s Using schema: %s%s%s\n", cCheck, cCyan, schemaDesc, cReset)

	if res.Valid {
		if len(res.Warnings) > 0 {
			fmt.Printf("%s %s%s is VALID%s (with %d warning%s)\n\n",
				cCheck, cBold, manifestPath, cReset, len(res.Warnings), plural(len(res.Warnings)))
			for _, warnIssue := range res.Warnings {
				fmt.Printf("  %s %s%-14s%s %s\n", cWarn, cYellow, warnIssue.Path, cReset, warnIssue.Message)
			}
		} else {
			fmt.Printf("%s %s%s is VALID%s\n", cCheck, cBold, manifestPath, cReset)
		}
	} else {
		fmt.Printf("%s %s%s is INVALID (%d error%s",
			cX, cBold, manifestPath, len(res.Errors), plural(len(res.Errors)))
		if len(res.Warnings) > 0 {
			fmt.Printf(", %d warning%s", len(res.Warnings), plural(len(res.Warnings)))
		}
		fmt.Printf(")%s\n\n", cReset)

		for _, errIssue := range res.Errors {
			fmt.Printf("  %s %s%-14s%s %s\n", cX, cRed, errIssue.Path, cReset, errIssue.Message)
		}
		for _, warnIssue := range res.Warnings {
			fmt.Printf("  %s %s%-14s%s %s\n", cWarn, cYellow, warnIssue.Path, cReset, warnIssue.Message)
		}
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
