/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"sa1l.nubi/sinit/utils/atcoder"
)

// atcoderCmd represents the atcoder command
var atcoderCmd = &cobra.Command{
	Use:   "ac",
	Short: "Fetch algorithm problems and create directory structure for a contest.",
	Long:  `This command pulls algorithm problems based on the contest ID, generates a folder, and creates algorithm solution files within it. Example usage: sinit ac -c=abc375`,
	Run: func(cmd *cobra.Command, args []string) {
		atcoder.CheckValidDir()
		if contestsID == "" {
			fmt.Print("which contest?(etc: abc133/arc101): ")
			fmt.Scanln(&contestsID)
		}
		fmt.Println("creating...")
		if err := atcoder.CreateContestsTasks(contestsID); err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		fmt.Println("let's go.")
	},
}
var contestsID string

func init() {
	rootCmd.AddCommand(atcoderCmd)

	atcoderCmd.PersistentFlags().StringVarP(&contestsID, "contest", "c", "", "-c=abc376")
}
