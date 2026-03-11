package gitcmd

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// IsValidRepo checks if the current directory is within a Git repository.
func IsValidRepo() error {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("not a git repository: %v", err)
	}
	return nil
}

// ListTrackedFiles returns a list of files currently tracked by Git.
func ListTrackedFiles() ([]string, error) {
	cmd := exec.Command("git", "ls-files")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git ls-files failed: %v", err)
	}

	lines := strings.Split(stdout.String(), "\n")
	var files []string
	for _, line := range lines {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}
