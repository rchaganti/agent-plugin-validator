package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ravik/agent-plugin-validator/schema"
	"github.com/ravik/agent-plugin-validator/validator"
)

const Version = "1.0.0"

// Terminal color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

type GlobalOptions struct {
	SchemaOverride string
	Format         string
	Quiet          bool
	ColorMode      string
	UseColor       bool
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	arg1 := os.Args[1]

	switch arg1 {
	case "-h", "--help", "help":
		printUsage()
		os.Exit(0)
	case "-v", "--version", "version":
		fmt.Printf("apv (Agent Plugin Validator) v%s\n", Version)
		os.Exit(0)
	case "schema":
		runSchemaCommand(os.Args[2:])
	case "validate":
		runValidateCommand(os.Args[2:])
	default:
		if strings.HasPrefix(arg1, "-") || fileExists(arg1) {
			runValidateCommand(os.Args[1:])
		} else {
			fmt.Fprintf(os.Stderr, "Unknown command or file: %s\nRun 'apv --help' for usage.\n", arg1)
			os.Exit(2)
		}
	}
}

func printUsage() {
	fmt.Print(`apv - Agent Plugin Validator CLI (v1.0.0)

Usage:
  apv validate [flags] <file>     Validate a plugin.json manifest against schema
  apv schema show [flags]         Show information about the active schema
  apv schema update [url]         Download and cache an updated JSON schema
  apv schema reset                Revert to embedded default schema
  apv --version                   Display CLI version
  apv --help                      Show usage instructions

Validate Flags:
  -s, --schema <path|url>        Custom schema override (file or URL)
  -f, --format <text|json>       Output format (default: text)
  -q, --quiet                    Quiet mode (suppress output, use exit code only)
      --color <auto|always|never> Control color output (default: auto)

Examples:
  apv validate plugin.json
  apv validate --format json plugin.json
  apv validate --quiet - < plugin.json
  apv schema update https://agent-plugins.org/schemas/1.0.0/plugin.schema.json
`)
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

func runValidateCommand(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	var opts GlobalOptions

	fs.StringVar(&opts.SchemaOverride, "schema", "", "Schema override path or URL")
	fs.StringVar(&opts.SchemaOverride, "s", "", "Schema override path or URL (shorthand)")
	fs.StringVar(&opts.Format, "format", "text", "Output format (text or json)")
	fs.StringVar(&opts.Format, "f", "text", "Output format (shorthand)")
	fs.BoolVar(&opts.Quiet, "quiet", false, "Quiet mode")
	fs.BoolVar(&opts.Quiet, "q", false, "Quiet mode (shorthand)")
	fs.StringVar(&opts.ColorMode, "color", "auto", "Color mode (auto, always, never)")

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	opts.UseColor = resolveColor(opts.ColorMode)

	manifestPath := ""
	if fs.NArg() > 0 {
		manifestPath = fs.Arg(0)
	} else {
		if !opts.Quiet {
			fmt.Fprintln(os.Stderr, "Error: manifest file path or '-' (for stdin) is required.")
		}
		os.Exit(2)
	}

	// 1. Read manifest input
	var manifestData []byte
	var err error

	if manifestPath == "-" {
		manifestData, err = io.ReadAll(os.Stdin)
		if err != nil {
			if !opts.Quiet {
				fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			}
			os.Exit(2)
		}
	} else {
		manifestData, err = os.ReadFile(manifestPath)
		if err != nil {
			if !opts.Quiet {
				fmt.Fprintf(os.Stderr, "Error reading file '%s': %v\n", manifestPath, err)
			}
			os.Exit(2)
		}
	}

	// 2. Resolve schema
	mgr, err := schema.NewManager()
	if err != nil {
		if !opts.Quiet {
			fmt.Fprintf(os.Stderr, "Schema manager initialization error: %v\n", err)
		}
		os.Exit(2)
	}

	schemaInfo, err := mgr.Resolve(opts.SchemaOverride)
	if err != nil {
		if !opts.Quiet {
			fmt.Fprintf(os.Stderr, "Error resolving schema: %v\n", err)
		}
		os.Exit(2)
	}

	// 3. Validate manifest
	result, err := validator.Validate(manifestData, schemaInfo)
	if err != nil {
		if !opts.Quiet {
			fmt.Fprintf(os.Stderr, "Runtime validation error: %v\n", err)
		}
		os.Exit(2)
	}

	// 4. Output results
	if opts.Quiet {
		if result.Valid {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	if opts.Format == "json" {
		outJSON, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(outJSON))
	} else {
		printTextValidationResult(result, opts.UseColor, manifestPath)
	}

	if result.Valid {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
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

func runSchemaCommand(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: apv schema <show|update|reset>")
		os.Exit(2)
	}

	cmd := args[0]
	subArgs := args[1:]

	mgr, err := schema.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Schema manager error: %v\n", err)
		os.Exit(2)
	}

	switch cmd {
	case "show":
		fs := flag.NewFlagSet("schema show", flag.ExitOnError)
		fmtFlag := fs.String("format", "text", "Output format")
		fs.StringVar(fmtFlag, "f", "text", "Output format")
		fs.Parse(subArgs)

		info, err := mgr.Show()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting schema info: %v\n", err)
			os.Exit(2)
		}

		if *fmtFlag == "json" {
			out, _ := json.MarshalIndent(info, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Printf("Active Schema:\n")
			fmt.Printf("  Title:  %s\n", info.Title)
			fmt.Printf("  ID:     %s\n", info.ID)
			fmt.Printf("  Source: %s\n", info.Source)
			fmt.Printf("  Path:   %s\n", info.Path)
		}

	case "update":
		fs := flag.NewFlagSet("schema update", flag.ExitOnError)
		fmtFlag := fs.String("format", "text", "Output format")
		fs.StringVar(fmtFlag, "f", "text", "Output format")
		fs.Parse(subArgs)

		targetURL := ""
		if fs.NArg() > 0 {
			targetURL = fs.Arg(0)
		}

		info, err := mgr.Update(targetURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating schema: %v\n", err)
			os.Exit(2)
		}

		if *fmtFlag == "json" {
			out, _ := json.MarshalIndent(info, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Printf("✓ Schema successfully updated and cached.\n")
			fmt.Printf("  ID:   %s\n", info.ID)
			fmt.Printf("  Path: %s\n", info.Path)
		}

	case "reset":
		if err := mgr.Reset(); err != nil {
			fmt.Fprintf(os.Stderr, "Error resetting schema: %v\n", err)
			os.Exit(2)
		}
		fmt.Println("✓ Schema reset. Reverted to embedded default schema.")

	default:
		fmt.Fprintf(os.Stderr, "Unknown schema subcommand: %s\n", cmd)
		os.Exit(2)
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
