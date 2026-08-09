package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rchaganti/agent-plugin-validator/schema"
	"github.com/rchaganti/agent-plugin-validator/validator"
	"github.com/spf13/cobra"
)

var (
	schemaOverride         string
	schemaManifestOverride string
	schemaMCPOverride      string
	schemaTypeFlag         string
	outputFormat           string
	quietMode              bool
)

var validateCmd = &cobra.Command{
	Use:   "validate <file|dir>",
	Short: "Validate plugin.json or mcp.json file or folder against JSON schema",
	Long: `Validate an Agent Plugin manifest (plugin.json) or MCP configuration (mcp.json) 
against the canonical Agent Plugins v1.0.0 JSON schema.
If a directory path is passed, apv automatically discovers and validates plugin.json and mcp.json files inside it.
Pass '-' as the file argument to read from standard input.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		targetPath := args[0]

		mgr, err := schema.NewManager()
		if err != nil {
			if !quietMode {
				fmt.Fprintf(os.Stderr, "Schema manager initialization error: %v\n", err)
			}
			os.Exit(2)
		}

		// Check if target is stdin
		if targetPath == "-" {
			manifestData, err := io.ReadAll(os.Stdin)
			if err != nil {
				if !quietMode {
					fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
				}
				os.Exit(2)
			}
			runSingleValidation(manifestData, "-", mgr)
			return
		}

		// Check file info (is it a directory or a file?)
		fileInfo, statErr := os.Stat(targetPath)
		if statErr != nil {
			if !quietMode {
				fmt.Fprintf(os.Stderr, "Error accessing '%s': %v\n", targetPath, statErr)
			}
			os.Exit(2)
		}

		if fileInfo.IsDir() {
			runDirectoryValidation(targetPath, mgr)
		} else {
			manifestData, err := os.ReadFile(targetPath)
			if err != nil {
				if !quietMode {
					fmt.Fprintf(os.Stderr, "Error reading file '%s': %v\n", targetPath, err)
				}
				os.Exit(2)
			}
			runSingleValidation(manifestData, targetPath, mgr)
		}
	},
}

func getOverrideFor(schemaType string) string {
	// 1. Check dedicated flags
	if schemaType == schema.SchemaTypeManifest && schemaManifestOverride != "" {
		return schemaManifestOverride
	}
	if schemaType == schema.SchemaTypeMCP && schemaMCPOverride != "" {
		return schemaMCPOverride
	}

	// 2. Check key=value format in --schema (e.g., --schema manifest=./p.json,mcp=./m.json or --schema manifest=./p.json)
	if strings.Contains(schemaOverride, "=") {
		parts := strings.Split(schemaOverride, ",")
		for _, part := range parts {
			kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(kv) == 2 {
				k := strings.ToLower(strings.TrimSpace(kv[0]))
				v := strings.TrimSpace(kv[1])
				if k == schemaType {
					return v
				}
			}
		}
	}

	// 3. Fallback to generic --schema override if not key=value
	if schemaOverride != "" && !strings.Contains(schemaOverride, "=") {
		return schemaOverride
	}

	return ""
}

func runSingleValidation(manifestData []byte, path string, mgr *schema.Manager) {
	targetType := schemaTypeFlag
	if targetType == "" || targetType == "auto" {
		targetType = schema.DetectType(manifestData, path)
	}

	override := getOverrideFor(targetType)
	schemaInfo, err := mgr.Resolve(targetType, override)
	if err != nil {
		if !quietMode {
			fmt.Fprintf(os.Stderr, "Error resolving schema: %v\n", err)
		}
		os.Exit(2)
	}

	result, err := validator.Validate(manifestData, schemaInfo)
	if err != nil {
		if !quietMode {
			fmt.Fprintf(os.Stderr, "Runtime validation error: %v\n", err)
		}
		os.Exit(2)
	}

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
		printTextValidationResult(result, useColor, path)
	}

	if result.Valid {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

func runDirectoryValidation(dirPath string, mgr *schema.Manager) {
	pluginCandidate := filepath.Join(dirPath, "plugin.json")
	mcpCandidate := filepath.Join(dirPath, "mcp.json")

	var filesToValidate []string
	if fileExists(pluginCandidate) {
		filesToValidate = append(filesToValidate, pluginCandidate)
	}
	if fileExists(mcpCandidate) {
		filesToValidate = append(filesToValidate, mcpCandidate)
	}

	if len(filesToValidate) == 0 {
		if quietMode {
			os.Exit(0)
		}
		if outputFormat == "json" {
			outJSON, _ := json.MarshalIndent(map[string]interface{}{
				"valid":   true,
				"scanned": 0,
				"message": fmt.Sprintf("No plugin.json or mcp.json found in directory '%s'", dirPath),
				"results": []interface{}{},
			}, "", "  ")
			fmt.Println(string(outJSON))
		} else {
			cBlue := ""
			cReset := ""
			if useColor {
				cBlue = colorCyan
				cReset = colorReset
			}
			fmt.Printf("ℹ %sNo plugin.json or mcp.json found in directory '%s'%s\n", cBlue, dirPath, cReset)
		}
		os.Exit(0)
	}

	allValid := true
	var results []*validator.Result

	for _, filePath := range filesToValidate {
		data, err := os.ReadFile(filePath)
		if err != nil {
			if !quietMode {
				fmt.Fprintf(os.Stderr, "Error reading file '%s': %v\n", filePath, err)
			}
			os.Exit(2)
		}

		targetType := schemaTypeFlag
		if targetType == "" || targetType == "auto" {
			targetType = schema.DetectType(data, filePath)
		}

		override := getOverrideFor(targetType)
		schemaInfo, err := mgr.Resolve(targetType, override)
		if err != nil {
			if !quietMode {
				fmt.Fprintf(os.Stderr, "Error resolving schema: %v\n", err)
			}
			os.Exit(2)
		}

		res, err := validator.Validate(data, schemaInfo)
		if err != nil {
			if !quietMode {
				fmt.Fprintf(os.Stderr, "Runtime validation error for '%s': %v\n", filePath, err)
			}
			os.Exit(2)
		}

		if !res.Valid {
			allValid = false
		}
		results = append(results, res)
	}

	if quietMode {
		if allValid {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	if outputFormat == "json" {
		outJSON, _ := json.MarshalIndent(map[string]interface{}{
			"valid":   allValid,
			"scanned": len(filesToValidate),
			"results": results,
		}, "", "  ")
		fmt.Println(string(outJSON))
	} else {
		for i, res := range results {
			if i > 0 {
				fmt.Println()
			}
			printTextValidationResult(res, useColor, filesToValidate[i])
		}
	}

	if allValid {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func init() {
	validateCmd.Flags().StringVarP(&schemaOverride, "schema", "s", "", "Custom schema override (path, URL, or key=value, e.g. manifest=p.json,mcp=m.json)")
	validateCmd.Flags().StringVar(&schemaManifestOverride, "schema-manifest", "", "Custom Manifest schema override (path or URL)")
	validateCmd.Flags().StringVar(&schemaMCPOverride, "schema-mcp", "", "Custom MCP schema override (path or URL)")
	validateCmd.Flags().StringVarP(&schemaTypeFlag, "type", "t", "auto", "Schema type: auto, manifest, mcp")
	validateCmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text or json)")
	validateCmd.Flags().BoolVarP(&quietMode, "quiet", "q", false, "Quiet mode (suppress output, exit code only)")

	// Shell autocompletion setup
	_ = validateCmd.MarkFlagFilename("schema", "json")
	_ = validateCmd.MarkFlagFilename("schema-manifest", "json")
	_ = validateCmd.MarkFlagFilename("schema-mcp", "json")
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
			fmt.Printf("%s %s%s is VALID%s (with %d warning%s)\n",
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
