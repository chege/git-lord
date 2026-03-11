package gitcmd

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
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

	cmd := exec.Command("git", "blame", "--line-porcelain", filePath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// git blame exits with non-zero on some empty files, which we can ignore.
		// However, we should return other legitimate errors.
		if !strings.Contains(stderr.String(), "no such path") &&
			!strings.Contains(stderr.String(), "has no contents") {
			return data, fmt.Errorf("git blame failed for %s: %w, stderr: %s", filePath, err, stderr.String())
		}
	}

	scanner := bufio.NewScanner(&stdout)
	var currentHash, currentAuthorEmail string
	hashToEmail := make(map[string]string)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "\t") {
			// This is the actual code line
			email := currentAuthorEmail
			if email == "" {
				email = hashToEmail[currentHash]
			}
			if email != "" && !strings.Contains(email, "not.committed.yet") {
				data.AuthorLines[email]++
			}
			// Reset for the next blame block
			currentHash = ""
			currentAuthorEmail = ""
		} else if currentHash == "" {
			// First line of a block starts with the commit hash
			parts := strings.SplitN(line, " ", 2)
			currentHash = parts[0]
		} else if strings.HasPrefix(line, "author-mail ") {
			email := strings.TrimPrefix(line, "author-mail ")
			email = strings.Trim(email, "<>")
			currentAuthorEmail = strings.ToLower(email)
			hashToEmail[currentHash] = currentAuthorEmail
		}
	}

	return data, nil
}
