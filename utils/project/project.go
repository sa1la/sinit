package project

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func InitProject(projectName, username, email string) error {
	projectPath, err := filepath.Abs(projectName)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", projectName, err)
	}

	if err := os.RemoveAll(filepath.Join(projectPath, ".git")); err != nil {
		return fmt.Errorf("remove .git: %w", err)
	}

	steps := [][]string{
		{"init"},
		{"config", "--local", "user.name", username},
		{"config", "--local", "user.email", email},
		{"add", "."},
		{"commit", "-m", "Initial commit"},
	}
	for _, args := range steps {
		cmd := exec.Command("git", args...)
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("run git %v: %w", args, err)
		}
	}
	return nil
}
