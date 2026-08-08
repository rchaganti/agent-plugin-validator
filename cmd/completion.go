package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell autocompletion script",
	Long: `Generate shell autocompletion script for apv.

To load completions:

Bash:
  $ source <(apv completion bash)
  # To load completions for every new session, run:
  # Linux:   apv completion bash > /etc/bash_completion.d/apv
  # macOS:   apv completion bash > $(brew --prefix)/etc/bash_completion.d/apv

Zsh:
  # If shell completion is not already enabled in your environment:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  # To load completions for every session:
  $ apv completion zsh > "${fpath[1]}/_apv"

Fish:
  $ apv completion fish | source
  # To load completions for every session:
  $ apv completion fish > ~/.config/fish/completions/apv.fish

PowerShell:
  PS> apv completion powershell | Out-String | Invoke-Expression
  # To load completions for every session, add the output to your PowerShell profile ($PROFILE).
`,
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			_ = cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			_ = cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			_ = cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			_ = cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
