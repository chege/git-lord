package main_test

import (
	"bytes"
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
	os.WriteFile(file1, []byte("line1\nline2\nline3\n"), 0644)
	runGit("add", "file1.txt")
	runGit("commit", "-m", "Initial commit from Alice")

	// Change author to Bob
	runGit("config", "user.name", "Bob")
	runGit("config", "user.email", "bob@example.com")

	// Commit 2 by Bob (modifies file1, creates file2)
	os.WriteFile(file1, []byte("line1\nline2\nline3\nline4 from bob\n"), 0644)
	file2 := filepath.Join(dir, "file2.txt")
	os.WriteFile(file2, []byte("bob line1\nbob line2\n"), 0644)
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
	defer os.RemoveAll(repoDir)

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

	if !strings.Contains(output, "Alice") || !strings.Contains(output, "Bob") {
		t.Errorf("missing authors in output:\n%s", output)
	}

	// CSV format: Author,loc,coms,fils,hours,months,distribution
	if !strings.Contains(output, "Alice,3,1,1,1,1") {
		t.Errorf("Alice metrics incorrect. Expected 'Alice,3,1,1,1,1' in output:\n%s", output)
	}
	if !strings.Contains(output, "Bob,3,1,2,1,1") {
		t.Errorf("Bob metrics incorrect. Expected 'Bob,3,1,2,1,1' in output:\n%s", output)
	}
}

func TestE2E_JSONOutput(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

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
