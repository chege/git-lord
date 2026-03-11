package gitcmd

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// CommitData represents information about a single commit.
type CommitData struct {
	Hash      string
	Author    string
	Email     string
	Timestamp int64
	Additions int
	Deletions int
	Files     int
}

// GetCommitHistory retrieves a list of commits that touch the repository,
// optionally filtered by a since date.
func GetCommitHistory(since string) ([]CommitData, error) {
	// Use --numstat to get additions/deletions per file
	args := []string{"log", "--format=COMMIT|%H|%aN|%aE|%at", "--numstat"}
	if since != "" {
		args = append(args, "--since="+since)
	}

	cmd := exec.Command("git", args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git log failed: %v", err)
	}

	var commits []CommitData
	var currentCommit *CommitData

	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "COMMIT|") {
			if currentCommit != nil {
				commits = append(commits, *currentCommit)
			}
			parts := strings.Split(line, "|")
			if len(parts) != 5 {
				continue
			}

			var ts int64
			_, _ = fmt.Sscanf(parts[4], "%d", &ts)

			currentCommit = &CommitData{
				Hash:      parts[1],
				Author:    parts[2],
				Email:     strings.ToLower(parts[3]),
				Timestamp: ts,
			}
		} else if currentCommit != nil {
			// Parse numstat lines: "added deleted path"
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				var added, deleted int
				if parts[0] != "-" { // Handle binary files
					_, _ = fmt.Sscanf(parts[0], "%d", &added)
				}
				if parts[1] != "-" {
					_, _ = fmt.Sscanf(parts[1], "%d", &deleted)
				}
				currentCommit.Additions += added
				currentCommit.Deletions += deleted
				currentCommit.Files++
			}
		}
	}

	if currentCommit != nil {
		commits = append(commits, *currentCommit)
	}

	return commits, nil
}
