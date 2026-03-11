package gitcmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// IsValidRepo checks if the current directory is within a Git repository.
func IsValidRepo(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}
	return nil
}

// ListTrackedFiles returns a list of files currently tracked by Git.
func ListTrackedFiles(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-files")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git ls-files failed: %w", err)
	}

	var files []string
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}
