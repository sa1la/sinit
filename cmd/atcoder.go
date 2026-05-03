package cmd

import (
	"fmt"

	"github.com/sa1la/sinit/utils/atcoder"
	"github.com/spf13/cobra"
)

func newAtcoderCmd(use, short, long string, lang atcoder.Lang) *cobra.Command {
	var contestID string
	var force bool
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		Run: func(cmd *cobra.Command, args []string) {
			if !atcoder.CheckValidDir() {
				return
			}
			if contestID == "" {
				fmt.Print("which contest?(etc: abc133/arc101): ")
				fmt.Scanln(&contestID)
			}
			fmt.Println("creating...")
			if err := atcoder.CreateContestsTasks(contestID, lang, force); err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("let's go.")
		},
	}
	cmd.Flags().StringVarP(&contestID, "contest", "c", "", "-c=abc376")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite existing sample files")
	return cmd
}

func init() {
	rootCmd.AddCommand(newAtcoderCmd(
		"ac",
		"Fetch algorithm problems and create directory structure for a contest. BTW, this is for golang.",
		"This command pulls algorithm problems based on the contest ID, generates a folder, and creates algorithm solution files within it. Example usage: sinit ac -c=abc375",
		atcoder.LangGo,
	))
	rootCmd.AddCommand(newAtcoderCmd(
		"acr",
		"Fetch AtCoder problems and scaffold a Rust source file for a contest.",
		"This command pulls algorithm problems based on the contest ID and writes a single .rs file with one stub function per problem. Example: sinit acr -c=abc375",
		atcoder.LangRust,
	))
}
