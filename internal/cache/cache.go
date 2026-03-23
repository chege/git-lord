package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type Cache struct {
	Version string                `json:"version"`
	HEAD    string                `json:"head"`
	Entries map[string]CacheEntry `json:"entries"`
	mu      sync.RWMutex          `json:"-"`
}

type CacheEntry struct {
	BlobHash    string             `json:"blob_hash"`
	AuthorLines map[string][]int64 `json:"author_lines"`
}

const (
	CacheVersion = "1"
	CacheDir     = ".git/git-lord-cache"
	CacheFile    = "blame.json"
)

func Load(headHash string) (*Cache, error) {
	c := &Cache{
		Version: CacheVersion,
		HEAD:    headHash,
		Entries: make(map[string]CacheEntry),
	}

	cachePath := filepath.Join(CacheDir, CacheFile)

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return c, nil
	}

	var existing Cache
	if err := json.Unmarshal(data, &existing); err != nil {
		return c, nil
	}

	if existing.Version != CacheVersion {
		return c, nil
	}

	existing.HEAD = headHash

	return &existing, nil
}

func (c *Cache) Save() error {
	if err := os.MkdirAll(CacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	cachePath := filepath.Join(CacheDir, CacheFile)
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

func (c *Cache) Get(filePath, blobHash string) (map[string][]int64, bool) {
	if c == nil {
		return nil, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Entries == nil {
		return nil, false
	}

	key := filePath + ":" + blobHash
	entry, ok := c.Entries[key]
	if !ok {
		return nil, false
	}

	if entry.BlobHash != blobHash {
		return nil, false
	}

	return entry.AuthorLines, true
}

func (c *Cache) Set(filePath, blobHash string, authorLines map[string][]int64) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Entries == nil {
		c.Entries = make(map[string]CacheEntry)
	}

	key := filePath + ":" + blobHash
	c.Entries[key] = CacheEntry{
		BlobHash:    blobHash,
		AuthorLines: authorLines,
	}
}

func GetBlobHash(filePath string) (string, error) {
	cmd := exec.Command("git", "hash-object", filePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get blob hash for %s: %w", filePath, err)
	}
	return strings.TrimSpace(string(output)), nil
}

func GetHEAD() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
