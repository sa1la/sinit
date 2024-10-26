package project

import (
	"fmt"
	"os"
	"os/exec"
)

func InitProject(projectName string) error {
	// We need to change the current working directory to the new project directory
	if err := os.Chdir(projectName); err != nil {
		fmt.Println("Error changing directory to new project:", err)
		return err
	}
	// We need to run the following command to remove the git history and start a new repository
	resetCmd := exec.Command("rm", "-rf", ".git")
	if err := resetCmd.Run(); err != nil {
		fmt.Println("Error removing git history:", err)
		return err
	}

	// We need to initialize a new git repository
	initCmd := exec.Command("git", "init")
	if err := initCmd.Run(); err != nil {
		fmt.Println("Error initializing new git repository:", err)
		return err
	}

	// We need to make an initial commit
	addCmd := exec.Command("git", "add", ".")
	if err := addCmd.Run(); err != nil {
		fmt.Println("Error adding files to git repository:", err)
		return err
	}

	commitCmd := exec.Command("git", "commit", "-m", "Initial commit")
	if err := commitCmd.Run(); err != nil {
		fmt.Println("Error making initial commit:", err)
		return err
	}
	return nil
}
