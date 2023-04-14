/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"sa1l.nubi/sinit/utils"
)

var (
	projectName string
)

const templateRepo = "https://github.com/sa1L-A/vue-template.git"

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "一个用于快速创建项目的指令",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		// First, we need to check if the project name is provided
		if projectName == "" {
			fmt.Print("Please enter a project name: ")
			fmt.Scanln(&projectName)

		}
		err := utils.CloneProject(projectName, templateRepo)
		if err != nil {
			fmt.Println("Generate project err:", err)
			return
		}
		// We need to read the package.json file in the new project directory
		packageJSONPath := filepath.Join(projectName, "package.json")
		packageJSONData, err := os.ReadFile(packageJSONPath)
		if err != nil {
			fmt.Println("Error reading package.json file:", err)
			return
		}

		// We need to replace the "name" field in the package.json file with the provided project name
		packageJSONData = bytes.Replace(packageJSONData, []byte("\"name\": \"vue-template\""), []byte(fmt.Sprintf("\"name\": \"%s\"", projectName)), -1)

		// We need to write the updated package.json file back to the new project directory
		if err := os.WriteFile(packageJSONPath, packageJSONData, 0644); err != nil {
			fmt.Println("Error writing package.json file:", err)
			return
		}

		if err := utils.InitProject(projectName); err != nil {
			return
		}
		fmt.Println("Project created successfully!")
	},
}

func init() {
	createCmd.Flags().StringVarP(&projectName, "name", "n", "", "Help message for name")
	rootCmd.AddCommand(createCmd)
}
