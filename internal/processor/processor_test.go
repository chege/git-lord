package processor

import (
	"context"
	"runtime"
	"testing"

	"github.com/chege/git-lord/internal/gitcmd"
	"github.com/chege/git-lord/internal/models"
)

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
