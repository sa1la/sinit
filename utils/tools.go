package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

// clone git repo (templateRepo) to current path , name projectName
func CloneProject(projectName, templateRepo string) error {
	tempDir, err := ioutil.TempDir("", "template")
	if err != nil {
		fmt.Println("Error creating temporary directory:", err)
		return err
	}
	defer os.RemoveAll(tempDir)

	newCmd := exec.Command("git", "clone", templateRepo, tempDir)
	if err := newCmd.Run(); err != nil {
		fmt.Println("Error cloning template repository:", err)
		return err
	}
	// Now we need to copy the files from the template directory to the new project directory
	if err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return os.MkdirAll(filepath.Join(projectName, relPath), info.Mode())
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		return ioutil.WriteFile(filepath.Join(projectName, relPath), data, info.Mode())
	}); err != nil {
		fmt.Println("Error copying template files:", err)
		return err
	}
	return nil
}

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
