package gitcmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/chege/git-lord/internal/cache"
)

// BlameData holds the timestamps of lines per author for a specific file.
type BlameData struct {
	AuthorLines map[string][]int64
}

// GetBlame parses git blame natively overhead-free.
func GetBlame(ctx context.Context, filePath string, c *cache.Cache, blobHash string) (BlameData, error) {
	data := BlameData{
		AuthorLines: make(map[string][]int64),
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "blame", "-M", "-C", "--line-porcelain", filePath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if !strings.Contains(stderr.String(), "no such path") &&
			!strings.Contains(stderr.String(), "has no contents") {
			return data, fmt.Errorf("git blame failed for %s: %w, stderr: %s", filePath, err, stderr.String())
		}
	}

	scanner := bufio.NewScanner(&stdout)
	var currentHash, currentAuthorEmail string
	var currentTimestamp int64
	hashToEmail := make(map[string]string)
	hashToTs := make(map[string]int64)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "\t") {
			email := currentAuthorEmail
			if email == "" {
				email = hashToEmail[currentHash]
			}
			ts := currentTimestamp
			if ts == 0 {
				ts = hashToTs[currentHash]
			}

			if email != "" && !strings.Contains(email, "not.committed.yet") {
				data.AuthorLines[email] = append(data.AuthorLines[email], ts)
			}
			currentHash = ""
			currentAuthorEmail = ""
			currentTimestamp = 0
		} else if currentHash == "" {
			parts := strings.SplitN(line, " ", 2)
			currentHash = parts[0]
		} else if strings.HasPrefix(line, "author-mail ") {
			email := strings.TrimPrefix(line, "author-mail ")
			email = strings.TrimSpace(email)
			email = strings.Trim(email, "<>")
			currentAuthorEmail = strings.ToLower(email)
			hashToEmail[currentHash] = currentAuthorEmail
		} else if strings.HasPrefix(line, "committer-time ") {
			tsStr := strings.TrimPrefix(line, "committer-time ")
			_, _ = fmt.Sscanf(tsStr, "%d", &currentTimestamp)
			hashToTs[currentHash] = currentTimestamp
		}
	}

	if c != nil && blobHash != "" {
		c.Set(filePath, blobHash, data.AuthorLines)
	}

	return data, nil
}
