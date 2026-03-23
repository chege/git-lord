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

func TestProcessHotspots(t *testing.T) {
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	res := ResultExtended{
		FileOwners: map[string]map[string]int{
			"hot/a.go": {
				"alice@example.com": 80,
				"bob@example.com":   20,
			},
			"hot/b.go": {
				"carol@example.com": 54,
				"dave@example.com":  6,
			},
			"warm/c.go": {
				"erin@example.com":  30,
				"frank@example.com": 30,
			},
			"tiny/d.go": {
				"gina@example.com": 40,
				"hank@example.com": 5,
			},
		},
	}

	commits := []gitcmd.CommitData{
		{
			Date:      now.AddDate(0, 0, -3),
			FileStats: []gitcmd.FileStat{{Path: "hot/a.go", Additions: 70, Deletions: 20}, {Path: "hot/b.go", Additions: 40, Deletions: 10}},
		},
		{
			Date:      now.AddDate(0, 0, -2),
			FileStats: []gitcmd.FileStat{{Path: "hot/a.go", Additions: 25, Deletions: 5}},
		},
		{
			Date:      now.AddDate(0, 0, -1),
			FileStats: []gitcmd.FileStat{{Path: "hot/b.go", Additions: 35, Deletions: 5}, {Path: "warm/c.go", Additions: 30, Deletions: 10}},
		},
		{
			Date:      now.AddDate(0, 0, -40),
			FileStats: []gitcmd.FileStat{{Path: "warm/c.go", Additions: 500, Deletions: 200}},
		},
		{
			Date:      now.AddDate(0, 0, -1),
			FileStats: []gitcmd.FileStat{{Path: "deleted/e.go", Additions: 80, Deletions: 10}, {Path: "tiny/d.go", Additions: 15, Deletions: 3}},
		},
	}

	report := ProcessHotspots(res, commits, 30, now)
	if report.WindowDays != 30 {
		t.Fatalf("expected window days 30, got %d", report.WindowDays)
	}
	if len(report.Hotspots) != 3 {
		t.Fatalf("expected 3 hotspots, got %d", len(report.Hotspots))
	}

	first := report.Hotspots[0]
	if first.Path != "hot/a.go" {
		t.Fatalf("expected hot/a.go ranked first, got %q", first.Path)
	}
	if first.Score != 48 || first.Risk != "WATCH" {
		t.Fatalf("expected hot/a.go score/risk 48 WATCH, got %d %s", first.Score, first.Risk)
	}
	if first.RecentChurn != 120 || first.RecentCommits != 2 {
		t.Fatalf("expected hot/a.go churn 120/2, got %d/%d", first.RecentChurn, first.RecentCommits)
	}
	if first.PrimaryOwner != "alice@example.com" || first.OwnerLines != 80 {
		t.Fatalf("unexpected primary owner for hot/a.go: %s %d", first.PrimaryOwner, first.OwnerLines)
	}
	if first.ChurnScore != 50 || first.OwnershipScore != 67 || first.SizeScore != 13 {
		t.Fatalf("unexpected component scores for hot/a.go: %d %d %d", first.ChurnScore, first.OwnershipScore, first.SizeScore)
	}

	second := report.Hotspots[1]
	if second.Path != "hot/b.go" {
		t.Fatalf("expected hot/b.go ranked second, got %q", second.Path)
	}
	if second.Score != 48 || second.Risk != "WATCH" {
		t.Fatalf("expected hot/b.go score/risk 48 WATCH, got %d %s", second.Score, second.Risk)
	}

	third := report.Hotspots[2]
	if third.Path != "warm/c.go" {
		t.Fatalf("expected warm/c.go ranked third, got %q", third.Path)
	}
	if third.Score != 11 || third.Risk != "LOW" {
		t.Fatalf("expected warm/c.go score/risk 11 LOW, got %d %s", third.Score, third.Risk)
	}
}

func TestProcessHotspotsDefaultWindowAndTieBreakers(t *testing.T) {
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	res := ResultExtended{
		FileOwners: map[string]map[string]int{
			"alpha.go": {"a@example.com": 50, "b@example.com": 10},
			"beta.go":  {"a@example.com": 50, "b@example.com": 10},
		},
	}
	commits := []gitcmd.CommitData{
		{Date: now.AddDate(0, 0, -29), FileStats: []gitcmd.FileStat{{Path: "beta.go", Additions: 30, Deletions: 10}}},
		{Date: now.AddDate(0, 0, -29), FileStats: []gitcmd.FileStat{{Path: "alpha.go", Additions: 30, Deletions: 10}}},
	}

	report := ProcessHotspots(res, commits, 0, now)
	if report.WindowDays != 30 {
		t.Fatalf("expected default window of 30 days, got %d", report.WindowDays)
	}
	if len(report.Hotspots) != 2 {
		t.Fatalf("expected 2 hotspots, got %d", len(report.Hotspots))
	}
	if report.Hotspots[0].Path != "alpha.go" {
		t.Fatalf("expected alpha.go first on deterministic path tie-break, got %q", report.Hotspots[0].Path)
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
			_ = ProcessRepository(ctx, files, commits, false, numCPU, nil)
		}
	})

	b.Run("NumCPUx2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ProcessRepository(ctx, files, commits, false, numCPU*2, nil)
		}
	})

	b.Run("NumCPUx4", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ProcessRepository(ctx, files, commits, false, numCPU*4, nil)
		}
	})

	b.Run("NumCPUx8", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ProcessRepository(ctx, files, commits, false, numCPU*8, nil)
		}
	})
}
