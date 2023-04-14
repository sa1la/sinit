/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"sa1l.nubi/sinit/utils"
)

var baseType string
var projectPath string

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "A brief description of your command",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if err := utils.InitProject(projectPath); err != nil {
			fmt.Println("Init error!")
			return
		}
		fmt.Println("Project init successfully!")
	},
}

func init() {
	initCmd.Flags().StringVarP(&baseType, "type", "t", "git", "-t=xxx")
	initCmd.Flags().StringVarP(&projectPath, "project", "p", ".", "-p=xxx")
	rootCmd.AddCommand(initCmd)
}
