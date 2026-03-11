package processor

import (
	"testing"

	"github.com/christopher/git-lord/internal/models"
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
