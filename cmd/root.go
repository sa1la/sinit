package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sinit",
	Short: "sinit — a small CLI to bootstrap projects and AtCoder contests.",
	Long: `sinit is a small CLI helper that automates common chores:

  - "init"    resets a target project's git history and creates a clean initial commit.
  - "ac"      fetches problems for an AtCoder contest and scaffolds Go solution files.
  - "acr"     does the same as "ac" but emits a single Rust source file with stubs.
  - "run"     compiles and runs a solution against stored sample inputs.
  - "bundle"  generates a self-contained Go file ready for AtCoder submission.

Run "sinit <subcommand> --help" for details on each command.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
