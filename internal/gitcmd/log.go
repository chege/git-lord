package gitcmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// CommitData represents information about a single commit.
type CommitData struct {
	Hash      string
	Author    string
	Email     string
	Timestamp int64
}

// GetCommitHistory retrieves a list of commits that touch the repository,
// optionally filtered by a since date.
func GetCommitHistory(since string) ([]CommitData, error) {
	repo, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, fmt.Errorf("failed to open repo: %v", err)
	}

	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get head: %v", err)
	}

	opts := &git.LogOptions{From: head.Hash()}

	if since != "" {
		// Basic parsing for "since" if they provide a standard date format
		if parsedTime, err := time.Parse("2006-01-02", since); err == nil {
			opts.Since = &parsedTime
		}
	}

	commitIter, err := repo.Log(opts)
	if err != nil {
		return nil, fmt.Errorf("git log failed: %v", err)
	}

	var commits []CommitData
	err = commitIter.ForEach(func(c *object.Commit) error {
		commits = append(commits, CommitData{
			Hash:      c.Hash.String(),
			Author:    c.Author.Name,
			Email:     strings.ToLower(c.Author.Email),
			Timestamp: c.Author.When.Unix(),
		})
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("commit iteration failed: %v", err)
	}

	return commits, nil
}
