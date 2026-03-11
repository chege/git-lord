package main_test

import (
	"bytes"
	"encoding/csv"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestRepo(t *testing.T) string {
	dir, err := os.MkdirTemp("", "git-lord-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Helper to run git commands
	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
		}
	}

	runGit("init")
	runGit("config", "user.name", "Alice")
	runGit("config", "user.email", "alice@example.com")

	// Commit 1 by Alice
	file1 := filepath.Join(dir, "file1.txt")
	if err := os.WriteFile(file1, []byte("line1\nline2\nline3\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit("add", "file1.txt")
	runGit("commit", "-m", "Initial commit from Alice")

	// Change author to Bob
	runGit("config", "user.name", "Bob")
	runGit("config", "user.email", "bob@example.com")

	// Commit 2 by Bob (modifies file1, creates file2)
	if err := os.WriteFile(file1, []byte("line1\nline2\nline3\nline4 from bob\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	file2 := filepath.Join(dir, "file2.txt")
	if err := os.WriteFile(file2, []byte("bob line1\nbob line2\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit("add", ".")
	runGit("commit", "-m", "Bob adds lines")

	return dir
}

func buildGitLord(t *testing.T) string {
	tempBin := filepath.Join(t.TempDir(), "git-lord")

	// Build the CLI binary
	cmd := exec.Command("go", "build", "-o", tempBin, "./cmd/git-lord")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build git-lord: %v\nOutput: %s", err, string(out))
	}
	return tempBin
}

func TestE2E_BasicMetrics(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer func() { _ = os.RemoveAll(repoDir) }()

	binPath := buildGitLord(t)

	// Run git-lord with CSV output for deterministic parsing
	cmd := exec.Command(binPath, "--format", "csv")
	cmd.Dir = repoDir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("git-lord execution failed: %v\nStderr: %s", err, stderr.String())
	}

	output := stdout.String()
	reader := csv.NewReader(strings.NewReader(output))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV: %v", err)
	}

	// Helper to find author row
	findAuthor := func(name string) []string {
		for _, row := range records {
			if row[0] == name {
				return row
			}
		}
		return nil
	}

	alice := findAuthor("Alice")
	if alice == nil {
		t.Fatalf("Alice not found in output: %s", output)
	}
	// Alice: loc=3, retention=100.0, coms=1, fils=1, exclusive=0 (shared file1.txt)
	if alice[1] != "3" || alice[2] != "100.0" || alice[3] != "1" || alice[4] != "1" || alice[5] != "0" {
		t.Errorf("Alice metrics incorrect: %v", alice)
	}

	bob := findAuthor("Bob")
	if bob == nil {
		t.Fatalf("Bob not found in output: %s", output)
	}
	// Bob: loc=3, retention=100.0, coms=1, fils=2, exclusive=1 (owns file2.txt)
	if bob[1] != "3" || bob[2] != "100.0" || bob[3] != "1" || bob[4] != "2" || bob[5] != "1" {
		t.Errorf("Bob metrics incorrect: %v", bob)
	}
}

func TestE2E_Pulse(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer func() { _ = os.RemoveAll(repoDir) }()

	binPath := buildGitLord(t)

	cmd := exec.Command(binPath, "pulse", "--format", "csv")
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git-lord pulse failed: %v\nOutput: %s", err, string(out))
	}

	output := string(out)
	// Pulse CSV header: Author,commits,additions,deletions,churn,files
	if !strings.Contains(output, "Author,commits,additions,deletions,churn,files") {
		t.Errorf("Pulse CSV header missing: %s", output)
	}

	if !strings.Contains(output, "Alice,1,+3,-0,3,1") {
		t.Errorf("Alice pulse metrics incorrect: %s", output)
	}
}

func TestE2E_JSONOutput(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer func() { _ = os.RemoveAll(repoDir) }()

	binPath := buildGitLord(t)

	cmd := exec.Command(binPath, "--format", "json")
	cmd.Dir = repoDir

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git-lord JSON failed: %v\nOutput: %s", err, string(out))
	}

	output := string(out)
	if !strings.Contains(output, `"TotalLoc": 6`) {
		t.Errorf("Expected TotalLoc: 6 in JSON, got:\n%s", output)
	}
	if !strings.Contains(output, `"Name": "Alice"`) {
		t.Errorf("Expected Alice in JSON")
	}
}
