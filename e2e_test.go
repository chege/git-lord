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

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
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

	commitAs := func(name, email, date, msg string) {
		env := []string{
			"GIT_AUTHOR_NAME=" + name,
			"GIT_AUTHOR_EMAIL=" + email,
			"GIT_COMMITTER_NAME=" + name,
			"GIT_COMMITTER_EMAIL=" + email,
			"GIT_AUTHOR_DATE=" + date,
			"GIT_COMMITTER_DATE=" + date,
		}
		runGit(env, "commit", "-m", msg)
	}

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
	commitAs("Alice", "alice@example.com", "2020-01-01T12:00:00", "Initial commit from Alice")

	// Commit 2 by Bob (Newer)
	if err := os.WriteFile(file1, []byte("line1\nline2\nline3\nline4 from bob\nline5 from bob\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	file2 := filepath.Join(dir, "file2.txt")
	if err := os.WriteFile(file2, []byte("bob line1\nbob line2\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	bobNotes := filepath.Join(dir, "bob_notes.txt")
	if err := os.WriteFile(bobNotes, []byte("bob note\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(nil, "add", ".")
	commitAs("Bob", "bob@example.com", "2026-01-01T12:00:00", "Bob adds lines")

	// Commit 3 by Bob in a different month and file type.
	bobScript := filepath.Join(dir, "deploy.sh")
	if err := os.WriteFile(bobScript, []byte("#!/bin/sh\necho deploy\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(bobNotes, []byte("bob note updated\nbob extra\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	bobOps := filepath.Join(dir, "ops.txt")
	if err := os.WriteFile(bobOps, []byte("ops\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(nil, "add", ".")
	commitAs("Bob", "bob@example.com", "2026-02-01T12:00:00", "Bob adds deploy script")

	caraGo := filepath.Join(dir, "service.go")
	if err := os.WriteFile(caraGo, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(nil, "add", ".")
	commitAs("Cara", "cara@example.com", "2026-03-01T12:00:00", "Cara adds Go service")

	caraJSON := filepath.Join(dir, "config.json")
	if err := os.WriteFile(caraJSON, []byte("{\n  \"ok\": true\n}\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(nil, "add", ".")
	commitAs("Cara", "cara@example.com", "2026-03-02T12:00:00", "Cara adds config")

	caraMD := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(caraMD, []byte("# Notes\n\nShip it.\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(nil, "add", ".")
	commitAs("Cara", "cara@example.com", "2026-03-03T12:00:00", "Cara adds notes")

	if err := os.WriteFile(caraMD, []byte("# Notes\n\nKeep shipping.\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(nil, "add", ".")
	commitAs("Cara", "cara@example.com", "2026-03-04T12:00:00", "Cara refreshes notes")

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
	// Alice: loc=4 (3 in file1, 1 in alice.txt), coms=1, fils=2
	if alice[1] != "4" || alice[3] != "1" || alice[4] != "2" {
		t.Errorf("Alice metrics incorrect: %v", alice)
	}

	bob := findAuthor("Bob")
	if bob == nil {
		t.Fatalf("Bob not found in output: %s", output)
	}
	// Bob: loc=9, coms=2, fils=5
	if bob[1] != "9" || bob[3] != "2" || bob[4] != "5" {
		t.Errorf("Bob metrics incorrect: %v", bob)
	}

	cara := findAuthor("Cara")
	if cara == nil {
		t.Fatalf("Cara not found in output: %s", output)
	}
	if cara[1] != "9" || cara[3] != "4" || cara[4] != "3" {
		t.Errorf("Cara metrics incorrect: %v", cara)
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

	// Bob: 2 commits, +10 lines, -1 del, net 9, churn 11, 6 files changed
	if !strings.Contains(output, "Bob,2,+10,-1,9,11,6") {
		t.Errorf("Bob pulse metrics incorrect: %s", output)
	}
}

func TestE2E_PulseSortByChurn(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer func() { _ = os.RemoveAll(repoDir) }()

	binPath := buildGitLord(t)

	cmd := exec.Command(binPath, "pulse", "--days", "5000", "--format", "csv", "--sort", "churn")
	cmd.Dir = repoDir
	cmd.Env = testEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git-lord pulse --sort churn failed: %v\nOutput: %s", err, string(out))
	}

	reader := csv.NewReader(strings.NewReader(string(out)))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse pulse csv: %v", err)
	}

	if len(records) < 4 {
		t.Fatalf("expected header plus three rows, got: %v", records)
	}

	if records[1][0] != "Bob" {
		t.Fatalf("expected Bob first when sorting by churn, got %q", records[1][0])
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
	footerPattern := regexp.MustCompile(`(?m)^│ TOTAL\s+│ 7\s+│ \+24\s+│ -2\s+│ \+22\s+│ 26\s+│ 12\s+│$`)
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

	cmd := exec.Command(binPath, "awards", "--no-progress")
	cmd.Dir = repoDir
	cmd.Env = testEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git-lord awards failed: %v\nOutput: %s", err, string(out))
	}

	output := stripANSI(string(out))
	if !strings.Contains(output, "THE AWARDS CEREMONY") {
		t.Errorf("Awards ceremony title missing: %s", output)
	}
	if !strings.Contains(output, "THE INDIANA JONES") || !strings.Contains(output, "Winner:  Alice") {
		t.Errorf("Alice should be Indiana Jones: %s", output)
	}
	if !strings.Contains(output, "THE EVERGREEN") || !strings.Contains(output, "Winner:  Alice") {
		t.Errorf("Alice should be Evergreen: %s", output)
	}
	if !strings.Contains(output, "THE LANDLORD") || !strings.Contains(output, "Winner:  Bob") {
		t.Errorf("Bob should be Landlord: %s", output)
	}
	if !strings.Contains(output, "THE MARATHONER") || !strings.Contains(output, "Winner:  Bob") {
		t.Errorf("Bob should be Marathoner: %s", output)
	}
	if !strings.Contains(output, "THE POLYGLOT") || !strings.Contains(output, "Winner:  Cara") {
		t.Errorf("Cara should be Polyglot: %s", output)
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
	if !strings.Contains(output, `"TotalLoc": 22`) {
		t.Errorf("Expected TotalLoc: 22 in JSON, got:\n%s", output)
	}
	if !strings.Contains(output, `"Name": "Alice"`) {
		t.Errorf("Expected Alice in JSON")
	}
}
