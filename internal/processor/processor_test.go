package processor

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/chege/git-lord/internal/gitcmd"
	"github.com/chege/git-lord/internal/models"
)

func TestProcessPulseSortMetrics(t *testing.T) {
	commits := []gitcmd.CommitData{
		{
			Author:    "Alice",
			Email:     "alice@example.com",
			Date:      time.Unix(100, 0),
			Additions: 3,
			Deletions: 0,
			Files:     1,
		},
		{
			Author:    "Alice",
			Email:     "alice@example.com",
			Date:      time.Unix(200, 0),
			Additions: 1,
			Deletions: 0,
			Files:     1,
		},
		{
			Author:    "Bob",
			Email:     "bob@example.com",
			Date:      time.Unix(300, 0),
			Additions: 10,
			Deletions: 9,
			Files:     5,
		},
		{
			Author:    "Carol",
			Email:     "carol@example.com",
			Date:      time.Unix(400, 0),
			Additions: 10,
			Deletions: 0,
			Files:     2,
		},
	}

	tests := []struct {
		name   string
		sort   string
		first  string
		second string
	}{
		{name: "default commits", sort: "", first: "Alice", second: "Bob"},
		{name: "sort additions", sort: "additions", first: "Bob", second: "Carol"},
		{name: "sort deletions", sort: "deletions", first: "Bob", second: "Alice"},
		{name: "sort net", sort: "net", first: "Carol", second: "Alice"},
		{name: "sort churn", sort: "churn", first: "Bob", second: "Carol"},
		{name: "sort files", sort: "files", first: "Bob", second: "Alice"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := ProcessPulse(commits, tt.sort, false)
			if len(stats) < 2 {
				t.Fatalf("expected at least two stats, got %d", len(stats))
			}
			if stats[0].Name != tt.first {
				t.Fatalf("expected first author %q, got %q", tt.first, stats[0].Name)
			}
			if stats[1].Name != tt.second {
				t.Fatalf("expected second author %q, got %q", tt.second, stats[1].Name)
			}
		})
	}
}

func TestCalculateBusFactor(t *testing.T) {
	tests := []struct {
		name     string
		authors  map[string]*models.AuthorMetrics
		totalLoc int
		expected int
	}{
		{
			name:     "Empty",
			authors:  map[string]*models.AuthorMetrics{},
			totalLoc: 0,
			expected: 0,
		},
		{
			name: "Single author owns all",
			authors: map[string]*models.AuthorMetrics{
				"a": {Loc: 100},
			},
			totalLoc: 100,
			expected: 1,
		},
		{
			name: "Two authors, one owns 60%",
			authors: map[string]*models.AuthorMetrics{
				"a": {Loc: 60},
				"b": {Loc: 40},
			},
			totalLoc: 100,
			expected: 1,
		},
		{
			name: "Three authors, 40/30/30",
			authors: map[string]*models.AuthorMetrics{
				"a": {Loc: 40},
				"b": {Loc: 30},
				"c": {Loc: 30},
			},
			totalLoc: 100,
			expected: 2, // 40 + 30 = 70 > 50
		},
		{
			name: "Equal distribution",
			authors: map[string]*models.AuthorMetrics{
				"a": {Loc: 20},
				"b": {Loc: 20},
				"c": {Loc: 20},
				"d": {Loc: 20},
				"e": {Loc: 20},
			},
			totalLoc: 100,
			expected: 3, // 20 + 20 + 20 = 60 > 50
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateBusFactor(tt.authors, tt.totalLoc)
			if got != tt.expected {
				t.Errorf("calculateBusFactor() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func BenchmarkProcessRepository(b *testing.B) {
	// Use some files for a quick but meaningful benchmark
	files := make([]string, 20)
	for i := 0; i < 20; i++ {
		files[i] = "internal/processor/processor.go"
	}
	commits := []gitcmd.CommitData{}

	numCPU := runtime.NumCPU()
	ctx := context.Background()

	b.Run("NumCPUx1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ProcessRepository(ctx, files, commits, false, numCPU)
		}
	})

	b.Run("NumCPUx2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ProcessRepository(ctx, files, commits, false, numCPU*2)
		}
	})

	b.Run("NumCPUx4", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ProcessRepository(ctx, files, commits, false, numCPU*4)
		}
	})

	b.Run("NumCPUx8", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ProcessRepository(ctx, files, commits, false, numCPU*8)
		}
	})
}
