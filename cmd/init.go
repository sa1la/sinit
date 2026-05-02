package cmd

import (
	"fmt"

	"github.com/sa1la/sinit/utils/project"
	"github.com/spf13/cobra"
)

var (
	projectPath string
	username    string
	email       string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Reset a project's git history and create a fresh initial commit.",
	Long:  `Removes the existing .git directory in the target project, re-initializes a new repository, configures the local user.name/user.email, and creates an initial commit. Example: sinit init -p=./my-project -u=alice -e=alice@example.com`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := project.InitProject(projectPath, username, email); err != nil {
			fmt.Println("Init error:", err)
			return
		}
		fmt.Println("Project init successfully!")
	},
}

func init() {
	initCmd.Flags().StringVarP(&projectPath, "project", "p", ".", "-p=path/to/project")
	initCmd.Flags().StringVarP(&username, "user", "u", "admin", "-u=username")
	initCmd.Flags().StringVarP(&email, "email", "e", "default@email.com", "-e=email@example.com")
	rootCmd.AddCommand(initCmd)
}
