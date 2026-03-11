package gitcmd

import (
	"fmt"

	"github.com/go-git/go-git/v5"
)

// IsValidRepo checks if the current directory is within a Git repository.
func IsValidRepo() error {
	_, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return fmt.Errorf("not a git repository: %v", err)
	}
	return nil
}

// ListTrackedFiles returns a list of files currently tracked by Git.
func ListTrackedFiles() ([]string, error) {
	repo, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, fmt.Errorf("failed to open repo: %v", err)
	}

	idx, err := repo.Storer.Index()
	if err != nil {
		return nil, fmt.Errorf("failed to get git index: %v", err)
	}

	var files []string
	for _, entry := range idx.Entries {
		files = append(files, entry.Name)
	}

	return files, nil
}
