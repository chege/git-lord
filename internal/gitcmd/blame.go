package gitcmd

import (
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5"
)

// BlameData holds the counts of lines per author for a specific file.
type BlameData struct {
	AuthorLines map[string]int
}

// GetBlame parsers git blame natively overhead-free.
func GetBlame(filePath string) (BlameData, error) {
	data := BlameData{
		AuthorLines: make(map[string]int),
	}

	repo, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return data, fmt.Errorf("failed to open repo: %v", err)
	}

	head, err := repo.Head()
	if err != nil {
		return data, fmt.Errorf("failed to get head: %v", err)
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return data, fmt.Errorf("failed to get commit: %v", err)
	}

	res, err := git.Blame(commit, filePath)
	if err != nil {
		return data, err
	}

	for _, line := range res.Lines {
		if strings.TrimSpace(line.Text) != "" {
			data.AuthorLines[line.Author]++
		}
	}

	return data, nil
}
