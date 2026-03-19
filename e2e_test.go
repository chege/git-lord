package main_test

import (
	"bytes"
	"encoding/csv"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func testEnv(extra ...string) []string {
	env := make([]string, 0, len(os.Environ())+len(extra))
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "GIT_") {
			continue
		}
		env = append(env, entry)
	}
	return append(env, extra...)
}

func setupTestRepo(t *testing.T) string {
	dir, err := os.MkdirTemp("", "git-lord-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Helper to run git commands
	runGit := func(env []string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = testEnv(env...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
		}
	}

	runGit(nil, "init")
	runGit(nil, "config", "user.name", "Alice")
	runGit(nil, "config", "user.email", "alice@example.com")

	// Commit 1 by Alice (Older)
	file1 := filepath.Join(dir, "file1.txt")
	if err := os.WriteFile(file1, []byte("line1\nline2\nline3\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	fileAlice := filepath.Join(dir, "alice.txt")
	if err := os.WriteFile(fileAlice, []byte("alice legacy\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(nil, "add", ".")
	// Set an old date for Alice
	runGit([]string{
		"GIT_AUTHOR_NAME=Alice",
		"GIT_AUTHOR_EMAIL=alice@example.com",
		"GIT_COMMITTER_NAME=Alice",
		"GIT_COMMITTER_EMAIL=alice@example.com",
		"GIT_AUTHOR_DATE=2020-01-01T12:00:00",
		"GIT_COMMITTER_DATE=2020-01-01T12:00:00",
	}, "commit", "-m", "Initial commit from Alice")

	// Change author to Bob
	runGit(nil, "config", "user.name", "Bob")
	runGit(nil, "config", "user.email", "bob@example.com")

	// Commit 2 by Bob (Newer)
	// Bob overwrites Alice's line 3 to trigger a deletion
	if err := os.WriteFile(file1, []byte("line1\nline2\nline3 from bob\nline4 from bob\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	file2 := filepath.Join(dir, "file2.txt")
	if err := os.WriteFile(file2, []byte("bob line1\nbob line2\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(nil, "add", ".")
	runGit([]string{
		"GIT_AUTHOR_NAME=Bob",
		"GIT_AUTHOR_EMAIL=bob@example.com",
		"GIT_COMMITTER_NAME=Bob",
		"GIT_COMMITTER_EMAIL=bob@example.com",
		"GIT_AUTHOR_DATE=2026-01-01T12:00:00",
		"GIT_COMMITTER_DATE=2026-01-01T12:00:00",
	}, "commit", "-m", "Bob adds lines")

	return dir
}

func buildGitLord(t *testing.T) string {
	tempBin := filepath.Join(t.TempDir(), "git-lord")

	// Build the CLI binary
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", tempBin, "./cmd/git-lord")
	cmd.Env = testEnv()
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
	cmd := exec.Command(binPath, "--all", "--format", "csv")
	cmd.Dir = repoDir
	cmd.Env = testEnv()
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
	// Alice: loc=3 (2 in file1, 1 in alice.txt), coms=1, fils=2
	if alice[1] != "3" || alice[3] != "1" || alice[4] != "2" {
		t.Errorf("Alice metrics incorrect: %v", alice)
	}

	bob := findAuthor("Bob")
	if bob == nil {
		t.Fatalf("Bob not found in output: %s", output)
	}
	// Bob: loc=4 (2 in file1, 2 in file2), coms=1, fils=2
	if bob[1] != "4" || bob[3] != "1" || bob[4] != "2" {
		t.Errorf("Bob metrics incorrect: %v", bob)
	}
}

func TestE2E_Pulse(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer func() { _ = os.RemoveAll(repoDir) }()

	binPath := buildGitLord(t)

	cmd := exec.Command(binPath, "pulse", "--days", "100", "--format", "csv")
	cmd.Dir = repoDir
	cmd.Env = testEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git-lord pulse failed: %v\nOutput: %s", err, string(out))
	}

	output := string(out)
	if !strings.Contains(output, "Author,commits,additions,deletions,net,churn,files") {
		t.Errorf("Pulse CSV header missing: %s", output)
	}

	// Bob: 1 commit, +4 lines, -1 del, net 3, churn 5, 2 files
	if !strings.Contains(output, "Bob,1,+4,-1,3,5,2") {
		t.Errorf("Bob pulse metrics incorrect: %s", output)
	}
}

func TestE2E_PulseTableTotals(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer func() { _ = os.RemoveAll(repoDir) }()

	binPath := buildGitLord(t)

	cmd := exec.Command(binPath, "pulse", "--days", "10000", "--no-progress")
	cmd.Dir = repoDir
	cmd.Env = testEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git-lord pulse failed: %v\nOutput: %s", err, string(out))
	}

	output := string(out)
	plainOutput := ansiPattern.ReplaceAllString(output, "")
	footerPattern := regexp.MustCompile(`(?m)^│ TOTAL\s+│ 2\s+│ \+8\s+│ -1\s+│ \+7\s+│ 9\s+│ 4\s+│$`)
	if !footerPattern.MatchString(plainOutput) {
		t.Errorf("Pulse table totals missing or incorrect: %s", output)
	}
	if !strings.Contains(output, "ACTIVITY PULSE") {
		t.Errorf("Pulse table header missing: %s", output)
	}
}

func TestE2E_Awards(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer func() { _ = os.RemoveAll(repoDir) }()

	binPath := buildGitLord(t)

	cmd := exec.Command(binPath, "awards")
	cmd.Dir = repoDir
	cmd.Env = testEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git-lord awards failed: %v\nOutput: %s", err, string(out))
	}

	output := string(out)
	if !strings.Contains(output, "THE AWARDS CEREMONY") {
		t.Errorf("Awards ceremony title missing: %s", output)
	}
	// Alice should win Indiana Jones (first commit in 2020)
	if !strings.Contains(output, "THE INDIANA JONES") || !strings.Contains(output, "Alice") {
		t.Errorf("Alice should be Indiana Jones: %s", output)
	}
}

func TestE2E_Silos(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer func() { _ = os.RemoveAll(repoDir) }()

	binPath := buildGitLord(t)

	cmd := exec.Command(binPath, "silos", "--min-loc", "0")
	cmd.Dir = repoDir
	cmd.Env = testEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git-lord silos failed: %v\nOutput: %s", err, string(out))
	}

	output := string(out)
	// file2.txt should be a silo for Bob
	if !strings.Contains(output, "file2.txt") || !strings.Contains(output, "bob@example.com") {
		t.Errorf("Silos report missing expected file2.txt data: %s", output)
	}
}

func TestE2E_Trends(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer func() { _ = os.RemoveAll(repoDir) }()

	binPath := buildGitLord(t)

	cmd := exec.Command(binPath, "trends")
	cmd.Dir = repoDir
	cmd.Env = testEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git-lord trends failed: %v\nOutput: %s", err, string(out))
	}

	output := string(out)
	// Alice committed in 2020, Bob in 2026
	if !strings.Contains(output, "2020-01") || !strings.Contains(output, "2026-01") {
		t.Errorf("Trends report missing expected time periods: %s", output)
	}
}

func TestE2E_JSONOutput(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer func() { _ = os.RemoveAll(repoDir) }()

	binPath := buildGitLord(t)

	cmd := exec.Command(binPath, "--format", "json")
	cmd.Dir = repoDir
	cmd.Env = testEnv()

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git-lord JSON failed: %v\nOutput: %s", err, string(out))
	}

	output := string(out)
	// Alice(3) + Bob(4) = 7
	if !strings.Contains(output, `"TotalLoc": 7`) {
		t.Errorf("Expected TotalLoc: 7 in JSON, got:\n%s", output)
	}
	if !strings.Contains(output, `"Name": "Alice"`) {
		t.Errorf("Expected Alice in JSON")
	}
}
