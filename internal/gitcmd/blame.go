package gitcmd

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

// BlameData holds the counts of lines per author for a specific file.
type BlameData struct {
	AuthorLines map[string]int
}

// GetBlame parses git blame --line-porcelain output for a given file.
func GetBlame(filePath string) (BlameData, error) {
	cmd := exec.Command("git", "blame", "--line-porcelain", filePath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return BlameData{}, fmt.Errorf("failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return BlameData{}, fmt.Errorf("failed to start git blame: %v", err)
	}

	data := BlameData{
		AuthorLines: make(map[string]int),
	}
	hashToAuthor := make(map[string]string)

	var currentHash string
	var currentAuthor string

	scanner := bufio.NewScanner(stdout)
	// Buffer might need to be increased if lines are ridiculously long,
	// but default 64kb line length in scanner is usually fine for source code.
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		if line[0] == '\t' {
			// This is the actual file contents (prefixed with tab by git blame)
			content := strings.TrimSpace(line[1:])
			// Requirement: Count non-empty lines
			if content != "" {
				author, ok := hashToAuthor[currentHash]
				if !ok {
					author = currentAuthor
				}
				data.AuthorLines[author]++
			}
			// Reset for the next blame block
			currentHash = ""
			currentAuthor = ""
		} else if currentHash == "" {
			// First line of a block starts with the commit hash
			parts := strings.SplitN(line, " ", 2)
			currentHash = parts[0]
		} else if strings.HasPrefix(line, "author ") {
			currentAuthor = strings.TrimPrefix(line, "author ")
			hashToAuthor[currentHash] = currentAuthor
		}
	}

	// Wait for completion, Ignore error as git blame exits with non-zero on some empty files
	_ = cmd.Wait()

	return data, nil
}
