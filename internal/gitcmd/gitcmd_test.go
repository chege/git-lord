package gitcmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/chege/git-lord/internal/cache"
)

func setupTestRepo(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("failed to restore dir: %v", err)
		}
	})

	if err := os.MkdirAll(".git", 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	initCmd := exec.Command("git", "init")
	if err := initCmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	if err := exec.Command("git", "config", "user.email", "test@example.com").Run(); err != nil {
		t.Fatalf("failed to config user.email: %v", err)
	}
	if err := exec.Command("git", "config", "user.name", "Test User").Run(); err != nil {
		t.Fatalf("failed to config user.name: %v", err)
	}

	return tmpDir
}

func commitFile(t *testing.T, filename, content, message string, date time.Time) {
	t.Helper()

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if err := exec.Command("git", "add", filename).Run(); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE="+date.Format(time.RFC3339),
		"GIT_COMMITTER_DATE="+date.Format(time.RFC3339),
	)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}
}

func TestGetBlame(t *testing.T) {
	setupTestRepo(t)

	commitFile(t, "test.txt", "line1\nline2\nline3\n", "initial commit", time.Now())

	ctx := context.Background()
	data, err := GetBlame(ctx, "test.txt", nil, "")

	if err != nil {
		t.Errorf("GetBlame() error = %v", err)
	}

	if len(data.AuthorLines) == 0 {
		t.Error("GetBlame() returned empty AuthorLines")
	}

	totalLines := 0
	for _, lines := range data.AuthorLines {
		totalLines += len(lines)
	}
	if totalLines != 3 {
		t.Errorf("expected 3 lines, got %d", totalLines)
	}
}

func TestGetBlameWithCache(t *testing.T) {
	setupTestRepo(t)

	commitFile(t, "cached.txt", "content\n", "commit", time.Now())

	c := &cache.Cache{
		Version: cache.CacheVersion,
		Entries: make(map[string]cache.CacheEntry),
	}

	ctx := context.Background()

	data1, err := GetBlame(ctx, "cached.txt", c, "")
	if err != nil {
		t.Errorf("GetBlame() first call error = %v", err)
	}

	data2, err := GetBlame(ctx, "cached.txt", c, "")
	if err != nil {
		t.Errorf("GetBlame() second call error = %v", err)
	}

	if len(data1.AuthorLines) != len(data2.AuthorLines) {
		t.Error("cached and non-cached results differ")
	}
}

func TestGetBlameNonexistentFile(t *testing.T) {
	setupTestRepo(t)

	commitFile(t, "exists.txt", "content\n", "commit", time.Now())

	ctx := context.Background()
	data, err := GetBlame(ctx, "nonexistent.txt", nil, "")

	if err != nil {
		t.Errorf("GetBlame() should not error for nonexistent file, got: %v", err)
	}

	if len(data.AuthorLines) != 0 {
		t.Error("GetBlame() should return empty data for nonexistent file")
	}
}

func TestGetCommitHistory(t *testing.T) {
	setupTestRepo(t)

	now := time.Now()
	commitFile(t, "file1.txt", "content1\n", "first commit", now.Add(-48*time.Hour))
	commitFile(t, "file2.txt", "content2\n", "second commit", now.Add(-24*time.Hour))
	commitFile(t, "file1.txt", "updated content\n", "third commit", now.Add(-1*time.Hour))

	ctx := context.Background()
	commits, err := GetCommitHistory(ctx, "")

	if err != nil {
		t.Errorf("GetCommitHistory() error = %v", err)
	}

	if len(commits) != 3 {
		t.Errorf("expected 3 commits, got %d", len(commits))
	}

	for _, commit := range commits {
		if commit.Hash == "" {
			t.Error("commit missing hash")
		}
		if commit.Author == "" {
			t.Error("commit missing author")
		}
		if commit.Email == "" {
			t.Error("commit missing email")
		}
	}
}

func TestGetCommitHistoryWithSince(t *testing.T) {
	setupTestRepo(t)

	now := time.Now()
	commitFile(t, "old.txt", "old\n", "old commit", now.Add(-72*time.Hour))
	commitFile(t, "recent.txt", "recent\n", "recent commit", now.Add(-1*time.Hour))

	ctx := context.Background()
	since := now.Add(-48 * time.Hour).Format("2006-01-02")
	commits, err := GetCommitHistory(ctx, since)

	if err != nil {
		t.Errorf("GetCommitHistory() error = %v", err)
	}

	if len(commits) == 0 {
		t.Error("expected at least one commit after since date")
	}
}

func TestGetCommitHistoryEmptyRepo(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	defer func() {
		if err = os.Chdir(originalDir); err != nil {
			t.Logf("failed to restore dir: %v", err)
		}
	}()

	if err = os.MkdirAll(".git", 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	if err = exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("failed to init git: %v", err)
	}

	ctx := context.Background()
	commits, err := GetCommitHistory(ctx, "")

	if err == nil {
		t.Skip("skipping: GetCommitHistory may error on empty repo depending on git version")
	}

	if len(commits) != 0 {
		t.Errorf("expected 0 commits in empty repo, got %d", len(commits))
	}
}

func TestIsValidRepo(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "valid repo",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				if err := exec.Command("git", "init", tmpDir).Run(); err != nil {
					t.Fatalf("failed to init git: %v", err)
				}
				return tmpDir
			},
			wantErr: false,
		},
		{
			name: "not a repo",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("failed to get current dir: %v", err)
			}
			if err = os.Chdir(dir); err != nil {
				t.Fatalf("failed to change dir: %v", err)
			}
			defer func() {
				if err = os.Chdir(originalDir); err != nil {
					t.Logf("failed to restore dir: %v", err)
				}
			}()

			ctx := context.Background()
			err = IsValidRepo(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("IsValidRepo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListTrackedFiles(t *testing.T) {
	setupTestRepo(t)

	commitFile(t, "file1.txt", "content1\n", "commit1", time.Now())

	if err := os.MkdirAll("subdir", 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	commitFile(t, filepath.Join("subdir", "file2.txt"), "content2\n", "commit2", time.Now())

	ctx := context.Background()
	files, err := ListTrackedFiles(ctx)

	if err != nil {
		t.Errorf("ListTrackedFiles() error = %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}

	foundFile1 := false
	foundFile2 := false
	for _, f := range files {
		if f == "file1.txt" {
			foundFile1 = true
		}
		if f == filepath.Join("subdir", "file2.txt") {
			foundFile2 = true
		}
	}

	if !foundFile1 {
		t.Error("file1.txt not in tracked files")
	}
	if !foundFile2 {
		t.Error("file2.txt not in tracked files")
	}
}

func TestFileChurnChurn(t *testing.T) {
	fc := FileChurn{
		Path:      "test.go",
		Additions: 100,
		Deletions: 50,
	}

	if fc.Churn() != 150 {
		t.Errorf("FileChurn.Churn() = %d, want 150", fc.Churn())
	}
}
