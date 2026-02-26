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
}

// GetCommitHistory retrieves a list of commits that touch the repository,
// optionally filtered by a since date.
func GetCommitHistory(since string) ([]CommitData, error) {
	args := []string{"log", "--format=%H|%aN|%aE|%at"}
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
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) != 4 {
			continue
		}

		var ts int64
		fmt.Sscanf(parts[3], "%d", &ts)

		commits = append(commits, CommitData{
			Hash:      parts[0],
			Author:    parts[1],
			Email:     strings.ToLower(parts[2]),
			Timestamp: ts,
		})
	}

	return commits, nil
}
