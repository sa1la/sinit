package cmd

import (
	"fmt"

	"github.com/sa1la/sinit/utils/atcoder"
	"github.com/spf13/cobra"
)

var atcoderRustCmd = &cobra.Command{
	Use:   "acr",
	Short: "Fetch AtCoder problems and scaffold a Rust source file for a contest.",
	Long:  "This command pulls algorithm problems based on the contest ID and writes a single .rs file with one stub function per problem. Example: sinit acr -c=abc375",
	Run: func(cmd *cobra.Command, args []string) {
		atcoder.CheckValidDir()
		if contestsID == "" {
			fmt.Print("which contest?(etc: abc133/arc101): ")
			fmt.Scanln(&contestsID)
		}
		fmt.Println("creating...")
		if err := atcoder.CreateContestsTasks(contestsID, atcoder.RUST); err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		fmt.Println("let's go.")
	},
}

func init() {
	rootCmd.AddCommand(atcoderRustCmd)

	atcoderRustCmd.PersistentFlags().StringVarP(&contestsID, "contest", "c", "", "-c=abc376")
}
