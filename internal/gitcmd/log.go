package gitcmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// FileStat represents per-file churn within a commit.
type FileStat struct {
	Path      string
	Additions int
	Deletions int
}

// CommitData represents information about a single commit.
type CommitData struct {
	Hash           string
	Author         string
	Email          string
	Date           time.Time
	Additions      int
	Deletions      int
	Files          int
	FileExtensions map[string]bool
	FileStats      []FileStat
	Message        string
	IsMerge        bool
}

// FileChurn captures recent churn for a tracked file.
type FileChurn struct {
	Path          string
	Additions     int
	Deletions     int
	RecentCommits int
}

// Churn returns total changed lines in the window.
func (f FileChurn) Churn() int {
	return f.Additions + f.Deletions
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
				Hash:           parts[1],
				Author:         parts[2],
				Email:          strings.ToLower(parts[3]),
				Date:           t,
				FileExtensions: make(map[string]bool),
				IsMerge:        strings.Contains(parents, " "),
				Message:        msg,
			}
		} else if currentCommit != nil {
			parts := strings.SplitN(line, "\t", 3)
			if len(parts) >= 3 {
				path := normalizeNumstatPath(parts[2])
				var added, deleted int
				isBinary := parts[0] == "-" || parts[1] == "-"
				if parts[0] != "-" {
					_, _ = fmt.Sscanf(parts[0], "%d", &added)
				}
				if parts[1] != "-" {
					_, _ = fmt.Sscanf(parts[1], "%d", &deleted)
				}
				currentCommit.Additions += added
				currentCommit.Deletions += deleted
				currentCommit.Files++

				if !isBinary && path != "" {
					currentCommit.FileStats = append(currentCommit.FileStats, FileStat{
						Path:      path,
						Additions: added,
						Deletions: deleted,
					})
				}

				ext := strings.ToLower(filepath.Ext(path))
				if ext == "" {
					ext = "[no extension]"
				}
				currentCommit.FileExtensions[ext] = true
			}
		}
	}

	if currentCommit != nil {
		commits = append(commits, *currentCommit)
	}

	return commits, nil
}

func normalizeNumstatPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if !strings.Contains(raw, "=>") {
		return raw
	}

	start := strings.Index(raw, "{")
	end := strings.Index(raw, "}")
	if start >= 0 && end > start {
		prefix := raw[:start]
		body := raw[start+1 : end]
		suffix := raw[end+1:]
		parts := strings.SplitN(body, "=>", 2)
		if len(parts) == 2 {
			return prefix + strings.TrimSpace(parts[1]) + suffix
		}
	}

	parts := strings.Split(raw, "=>")
	if len(parts) == 0 {
		return raw
	}

	return strings.TrimSpace(parts[len(parts)-1])
}
