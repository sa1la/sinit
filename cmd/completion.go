package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionNoDesc bool

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate an autocompletion script for sinit in the specified shell.

See each sub-command's help for details on how to use the generated script.

To load completions:

Bash:
  source <(sinit completion bash)

  # To load completions for each session, execute once:
  # Linux:
  sinit completion bash > /etc/bash_completion.d/sinit
  # macOS:
  sinit completion bash > $(brew --prefix)/etc/bash_completion.d/sinit

Zsh:
  source <(sinit completion zsh)

  # To load completions for each session, execute once:
  sinit completion zsh > "${fpath[1]}/_sinit"

Fish:
  sinit completion fish | source

  # To load completions for each session, execute once:
  sinit completion fish > ~/.config/fish/completions/sinit.fish

PowerShell:
  sinit completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  sinit completion powershell > sinit.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			if completionNoDesc {
				return rootCmd.GenZshCompletionNoDesc(os.Stdout)
			}
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, !completionNoDesc)
		case "powershell":
			if completionNoDesc {
				return rootCmd.GenPowerShellCompletion(os.Stdout)
			}
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return fmt.Errorf("unsupported shell: %s", args[0])
		}
	},
}

func init() {
	completionCmd.Flags().BoolVar(&completionNoDesc, "no-descriptions", false, "disable completion descriptions")
	rootCmd.AddCommand(completionCmd)
}
