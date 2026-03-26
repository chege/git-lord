package cache

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		headHash string
		setup    func(t *testing.T, cacheDir string)
		wantErr  bool
	}{
		{
			name:     "new cache with no existing file",
			headHash: "abc123",
			setup:    func(t *testing.T, cacheDir string) {},
			wantErr:  false,
		},
		{
			name:     "cache with invalid JSON",
			headHash: "def456",
			setup: func(t *testing.T, cacheDir string) {
				if err := os.MkdirAll(cacheDir, 0755); err != nil {
					t.Fatalf("failed to create cache dir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(cacheDir, CacheFile), []byte("invalid json"), 0644); err != nil {
					t.Fatalf("failed to write invalid cache: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name:     "cache with wrong version",
			headHash: "ghi789",
			setup: func(t *testing.T, cacheDir string) {
				if err := os.MkdirAll(cacheDir, 0755); err != nil {
					t.Fatalf("failed to create cache dir: %v", err)
				}
				content := `{"version": "2", "head": "old", "entries": {}}`
				if err := os.WriteFile(filepath.Join(cacheDir, CacheFile), []byte(content), 0644); err != nil {
					t.Fatalf("failed to write cache: %v", err)
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			originalDir, _ := os.Getwd()
			_ = os.Chdir(tmpDir)
			defer func() { _ = os.Chdir(originalDir) }()

			if err := os.MkdirAll(".git", 0755); err != nil {
				t.Fatalf("failed to create .git dir: %v", err)
			}

			tt.setup(t, CacheDir)

			c, err := Load(tt.headHash)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if c == nil {
				t.Fatal("Load() returned nil cache")
			}

			if c.Version != CacheVersion {
				t.Errorf("expected version %s, got %s", CacheVersion, c.Version)
			}

			if c.HEAD != tt.headHash {
				t.Errorf("expected HEAD %s, got %s", tt.headHash, c.HEAD)
			}

			if c.Entries == nil {
				t.Error("expected non-nil Entries map")
			}
		})
	}
}

func TestCacheSave(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.MkdirAll(".git", 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	c := &Cache{
		Version: CacheVersion,
		HEAD:    "test-head",
		Entries: map[string]CacheEntry{
			"file1:hash1": {
				BlobHash: "hash1",
				AuthorLines: map[string][]int64{
					"alice@example.com": {1000000, 1000001},
				},
			},
		},
	}

	if err := c.Save(); err != nil {
		t.Errorf("Save() error = %v", err)
	}

	cachePath := filepath.Join(CacheDir, CacheFile)
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("Save() did not create cache file")
	}

	loaded, err := Load("test-head")
	if err != nil {
		t.Errorf("Load() after Save() error = %v", err)
	}

	if loaded.HEAD != c.HEAD {
		t.Errorf("loaded HEAD = %s, want %s", loaded.HEAD, c.HEAD)
	}

	if len(loaded.Entries) != len(c.Entries) {
		t.Errorf("loaded entries count = %d, want %d", len(loaded.Entries), len(c.Entries))
	}
}

func TestCacheGet(t *testing.T) {
	tests := []struct {
		name     string
		cache    *Cache
		filePath string
		blobHash string
		wantOk   bool
	}{
		{
			name:     "nil cache",
			cache:    nil,
			filePath: "file1",
			blobHash: "hash1",
			wantOk:   false,
		},
		{
			name: "cache with nil entries",
			cache: &Cache{
				Entries: nil,
			},
			filePath: "file1",
			blobHash: "hash1",
			wantOk:   false,
		},
		{
			name: "cache hit",
			cache: &Cache{
				Entries: map[string]CacheEntry{
					"file1:hash1": {
						BlobHash: "hash1",
						AuthorLines: map[string][]int64{
							"alice@example.com": {1000000},
						},
					},
				},
			},
			filePath: "file1",
			blobHash: "hash1",
			wantOk:   true,
		},
		{
			name: "cache miss - wrong hash",
			cache: &Cache{
				Entries: map[string]CacheEntry{
					"file1:hash1": {
						BlobHash: "oldhash",
						AuthorLines: map[string][]int64{
							"alice@example.com": {1000000},
						},
					},
				},
			},
			filePath: "file1",
			blobHash: "hash1",
			wantOk:   false,
		},
		{
			name: "cache miss - key not found",
			cache: &Cache{
				Entries: map[string]CacheEntry{},
			},
			filePath: "file1",
			blobHash: "hash1",
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.cache.Get(tt.filePath, tt.blobHash)
			if ok != tt.wantOk {
				t.Errorf("Get() ok = %v, want %v", ok, tt.wantOk)
			}
			if tt.wantOk && got == nil {
				t.Error("Get() returned nil data on cache hit")
			}
		})
	}
}

func TestCacheSet(t *testing.T) {
	tests := []struct {
		name        string
		cache       *Cache
		filePath    string
		blobHash    string
		authorLines map[string][]int64
	}{
		{
			name:        "nil cache",
			cache:       nil,
			filePath:    "file1",
			blobHash:    "hash1",
			authorLines: map[string][]int64{"alice@example.com": {1000000}},
		},
		{
			name: "set with nil entries",
			cache: &Cache{
				Entries: nil,
			},
			filePath:    "file1",
			blobHash:    "hash1",
			authorLines: map[string][]int64{"alice@example.com": {1000000}},
		},
		{
			name: "set new entry",
			cache: &Cache{
				Entries: make(map[string]CacheEntry),
			},
			filePath:    "file1",
			blobHash:    "hash1",
			authorLines: map[string][]int64{"alice@example.com": {1000000, 1000001}},
		},
		{
			name: "update existing entry",
			cache: &Cache{
				Entries: map[string]CacheEntry{
					"file1:hash1": {
						BlobHash:    "oldhash",
						AuthorLines: map[string][]int64{"bob@example.com": {2000000}},
					},
				},
			},
			filePath:    "file1",
			blobHash:    "hash1",
			authorLines: map[string][]int64{"alice@example.com": {1000000}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cache.Set(tt.filePath, tt.blobHash, tt.authorLines)

			if tt.cache != nil {
				key := tt.filePath + ":" + tt.blobHash
				entry, ok := tt.cache.Entries[key]
				if !ok && tt.name != "nil cache" {
					t.Error("Set() did not add entry to cache")
				}
				if ok {
					if entry.BlobHash != tt.blobHash {
						t.Errorf("BlobHash = %s, want %s", entry.BlobHash, tt.blobHash)
					}
				}
			}
		})
	}
}

func TestGetBlobHash(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.MkdirAll(".git", 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	if err := os.WriteFile("test.txt", []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	hash, err := GetBlobHash("test.txt")
	if err != nil {
		t.Errorf("GetBlobHash() error = %v", err)
	}
	if hash == "" {
		t.Error("GetBlobHash() returned empty hash")
	}
}

func TestGetBlobHashesBatch(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.MkdirAll(".git", 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	tests := []struct {
		name      string
		filePaths []string
		wantCount int
	}{
		{
			name:      "empty file list",
			filePaths: []string{},
			wantCount: 0,
		},
		{
			name:      "single file",
			filePaths: []string{"test1.txt"},
			wantCount: 1,
		},
		{
			name:      "multiple files",
			filePaths: []string{"test1.txt", "test2.txt"},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, path := range tt.filePaths {
				if err := os.WriteFile(path, []byte("content of "+path), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
			}

			hashes, err := GetBlobHashesBatch(tt.filePaths)
			if err != nil {
				t.Errorf("GetBlobHashesBatch() error = %v", err)
			}

			if len(hashes) != tt.wantCount {
				t.Errorf("GetBlobHashesBatch() returned %d hashes, want %d", len(hashes), tt.wantCount)
			}

			for _, path := range tt.filePaths {
				if _, ok := hashes[path]; !ok && len(tt.filePaths) > 0 {
					t.Errorf("GetBlobHashesBatch() missing hash for %s", path)
				}
			}
		})
	}
}

func TestGetHEAD(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.MkdirAll(".git", 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	initCmd := exec.Command("git", "init")
	if err := initCmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	if err := os.WriteFile("test.txt", []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	addCmd := exec.Command("git", "add", "test.txt")
	if err := addCmd.Run(); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	commitCmd := exec.Command("git", "commit", "-m", "initial commit")
	commitCmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	head, err := GetHEAD()
	if err != nil {
		t.Errorf("GetHEAD() error = %v", err)
	}
	if head == "" {
		t.Error("GetHEAD() returned empty hash")
	}
}

func TestSaveErrorCases(t *testing.T) {
	t.Run("cache save creates directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(originalDir) }()

		c := &Cache{
			Version: CacheVersion,
			HEAD:    "test",
			Entries: make(map[string]CacheEntry),
		}

		err := c.Save()
		if err != nil {
			t.Errorf("Save() error = %v", err)
		}

		if _, err := os.Stat(CacheDir); os.IsNotExist(err) {
			t.Error("Save() did not create cache directory")
		}
	})
}

func TestGetBlobHashErrors(t *testing.T) {
	t.Run("nonexistent file", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(originalDir) }()

		if err := os.MkdirAll(".git", 0755); err != nil {
			t.Fatalf("failed to create .git dir: %v", err)
		}
		if err := exec.Command("git", "init").Run(); err != nil {
			t.Fatalf("failed to init git: %v", err)
		}

		_, err := GetBlobHash("nonexistent.txt")
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestGetHEADError(t *testing.T) {
	t.Run("not a git repo", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(originalDir) }()

		_, err := GetHEAD()
		if err == nil {
			t.Error("expected error when not in git repo")
		}
	})
}

func TestGetBlobHashesBatchFallback(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.MkdirAll(".git", 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("failed to init git: %v", err)
	}

	if err := os.WriteFile("file1.txt", []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	hashes, err := GetBlobHashesBatch([]string{"file1.txt"})
	if err != nil {
		t.Errorf("GetBlobHashesBatch() error = %v", err)
	}

	if len(hashes) != 1 {
		t.Errorf("expected 1 hash, got %d", len(hashes))
	}
}
