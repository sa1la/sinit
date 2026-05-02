package project

import (
	"fmt"
	"os"
	"os/exec"
)

func InitProject(projectName, username, email string) error {
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(projectName); err != nil {
		return fmt.Errorf("error changing directory to %s: %v", projectName, err)
	}

	if err := os.RemoveAll(".git"); err != nil {
		return fmt.Errorf("error removing git history: %v", err)
	}

	steps := []struct {
		name string
		args []string
	}{
		{"git", []string{"init"}},
		{"git", []string{"config", "--local", "user.name", username}},
		{"git", []string{"config", "--local", "user.email", email}},
		{"git", []string{"add", "."}},
		{"git", []string{"commit", "-m", "Initial commit"}},
	}
	for _, s := range steps {
		if err := exec.Command(s.name, s.args...).Run(); err != nil {
			return fmt.Errorf("error running %s %v: %v", s.name, s.args, err)
		}
	}
	return nil
}
