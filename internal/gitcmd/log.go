package gitcmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CommitData represents information about a single commit.
type CommitData struct {
	Hash      string
	Author    string
	Email     string
	Date      time.Time
	Additions int
	Deletions int
	Files     int
	Message   string
	IsMerge   bool
}

// GetCommitHistory retrieves a list of commits that touch the repository,
// optionally filtered by a since date.
func GetCommitHistory(ctx context.Context, since string) ([]CommitData, error) {
	// %ai gives ISO author date: 2023-01-01 12:00:00 +0100
	// -M detects renames
	args := []string{"log", "-M", "--format=COMMIT|%H|%aN|%aE|%ai|%P|%s", "--numstat"}
	if since != "" {
		args = append(args, "--since="+since)
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
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
			if len(parts) < 5 {
				continue
			}

			// Parse ISO date
			t, err := time.Parse("2006-01-02 15:04:05 -0700", parts[4])
			if err != nil {
				// Fallback to now if parsing fails
				t = time.Now()
			}

			parents := ""
			msg := ""
			if len(parts) >= 6 {
				parents = parts[5]
			}
			if len(parts) >= 7 {
				msg = parts[6]
			}

			currentCommit = &CommitData{
				Hash:    parts[1],
				Author:  parts[2],
				Email:   strings.ToLower(parts[3]),
				Date:    t,
				IsMerge: strings.Contains(parents, " "),
				Message: msg,
			}
		} else if currentCommit != nil {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				var added, deleted int
				if parts[0] != "-" {
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
