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

func TestProcessAwards(t *testing.T) {
	tests := []struct {
		name      string
		result    models.Result
		minAwards int
	}{
		{
			name: "empty authors",
			result: models.Result{
				Authors: map[string]*models.AuthorMetrics{},
			},
			minAwards: 0,
		},
		{
			name: "single author",
			result: models.Result{
				Authors: map[string]*models.AuthorMetrics{
					"alice@example.com": {
						Name:              "Alice",
						Loc:               100,
						Commits:           10,
						Files:             5,
						LifetimeAdditions: 200,
						LifetimeDeletions: 100,
						OldestLineTs:      time.Now().AddDate(0, -6, 0).Unix(),
						MessageWords:      50,
						ActiveDays:        map[string]bool{"2024-01-01": true},
						FileExtensions:    map[string]bool{".go": true},
					},
				},
			},
			minAwards: 1,
		},
		{
			name: "multiple authors",
			result: models.Result{
				Authors: map[string]*models.AuthorMetrics{
					"alice@example.com": {
						Name:              "Alice",
						Loc:               100,
						Commits:           20,
						Files:             5,
						ExclusiveFiles:    3,
						LifetimeAdditions: 300,
						LifetimeDeletions: 200,
						OldestLineTs:      time.Now().AddDate(-1, 0, 0).Unix(),
						MessageWords:      100,
						ActiveDays:        map[string]bool{"2024-01-01": true, "2024-01-02": true},
						FridayAfterFour:   5,
						LintCommits:       3,
						FileExtensions:    map[string]bool{".go": true, ".md": true},
						Months:            6,
					},
					"bob@example.com": {
						Name:              "Bob",
						Loc:               80,
						Commits:           15,
						Files:             8,
						ExclusiveFiles:    2,
						LifetimeAdditions: 150,
						LifetimeDeletions: 70,
						OldestLineTs:      time.Now().AddDate(0, -3, 0).Unix(),
						MessageWords:      75,
						ActiveDays:        map[string]bool{"2024-02-01": true},
						FridayAfterFour:   2,
						MergeCommits:      5,
						FileExtensions:    map[string]bool{".go": true, ".js": true, ".py": true},
						Months:            3,
						CommitIntervals:   []int64{300, 600, 900},
					},
				},
			},
			minAwards: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			awards := ProcessAwards(tt.result, false)
			if len(awards) < tt.minAwards {
				t.Errorf("expected at least %d awards, got %d", tt.minAwards, len(awards))
			}

			for _, award := range awards {
				if award.Title == "" {
					t.Error("award missing title")
				}
				if award.Winner == "" {
					t.Error("award missing winner")
				}
			}
		})
	}
}

func TestProcessLegacy(t *testing.T) {
	result := models.Result{
		Authors: map[string]*models.AuthorMetrics{
			"alice@example.com": {
				Name:         "Alice",
				OldestLineTs: time.Now().AddDate(-1, 0, 0).Unix(),
			},
		},
	}

	stats := ProcessLegacy(result)

	if len(stats) == 0 {
		t.Error("ProcessLegacy returned empty stats")
	}

	totalPct := 0.0
	for _, s := range stats {
		totalPct += s.Pct
		if s.Year < 1970 || s.Year > 2100 {
			t.Errorf("invalid year: %d", s.Year)
		}
	}

	if totalPct > 0 && (totalPct < 99.0 || totalPct > 101.0) {
		t.Errorf("percentages don't sum to ~100: %f", totalPct)
	}
}

func TestProcessSilos(t *testing.T) {
	tests := []struct {
		name     string
		result   ResultExtended
		minLOC   int
		minSilos int
		maxSilos int
	}{
		{
			name: "no silos",
			result: ResultExtended{
				FileOwners: map[string]map[string]int{
					"file1.go": {"alice@example.com": 30, "bob@example.com": 30},
				},
			},
			minLOC:   50,
			minSilos: 0,
			maxSilos: 0,
		},
		{
			name: "with silos",
			result: ResultExtended{
				FileOwners: map[string]map[string]int{
					"large_file.go": {"alice@example.com": 95, "bob@example.com": 5},
					"small_file.go": {"alice@example.com": 40},
				},
			},
			minLOC:   50,
			minSilos: 1,
			maxSilos: 1,
		},
		{
			name: "high ownership threshold",
			result: ResultExtended{
				FileOwners: map[string]map[string]int{
					"critical.go": {"alice@example.com": 98},
				},
			},
			minLOC:   50,
			minSilos: 1,
			maxSilos: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			silos := ProcessSilos(tt.result, tt.minLOC)

			if len(silos) < tt.minSilos {
				t.Errorf("expected at least %d silos, got %d", tt.minSilos, len(silos))
			}
			if len(silos) > tt.maxSilos {
				t.Errorf("expected at most %d silos, got %d", tt.maxSilos, len(silos))
			}

			for _, silo := range silos {
				if silo.LOC < tt.minLOC {
					t.Errorf("silo LOC %d below minimum %d", silo.LOC, tt.minLOC)
				}
				if silo.Ownership < 80.0 {
					t.Errorf("silo ownership %.1f below threshold", silo.Ownership)
				}
			}
		})
	}
}

func TestProcessTrends(t *testing.T) {
	commits := []gitcmd.CommitData{
		{
			Date:      time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			Additions: 100,
			Deletions: 50,
		},
		{
			Date:      time.Date(2024, 1, 20, 12, 0, 0, 0, time.UTC),
			Additions: 80,
			Deletions: 20,
		},
		{
			Date:      time.Date(2024, 2, 10, 12, 0, 0, 0, time.UTC),
			Additions: 200,
			Deletions: 100,
		},
	}

	trends := ProcessTrends(commits)

	if len(trends) == 0 {
		t.Error("ProcessTrends returned empty trends")
	}

	for _, trend := range trends {
		if trend.Period == "" {
			t.Error("trend missing period")
		}
		if trend.Net != trend.Additions-trend.Deletions {
			t.Errorf("trend net %d doesn't match additions - deletions", trend.Net)
		}
	}
}

func TestProcessCommitHygiene(t *testing.T) {
	tests := []struct {
		name        string
		commits     []gitcmd.CommitData
		minAuthors  int
		checkScores bool
	}{
		{
			name:        "empty commits",
			commits:     []gitcmd.CommitData{},
			minAuthors:  0,
			checkScores: false,
		},
		{
			name: "good hygiene commits",
			commits: []gitcmd.CommitData{
				{
					Author:  "Alice",
					Email:   "alice@example.com",
					Message: "feat(auth): add login functionality (#123)",
					IsMerge: false,
				},
				{
					Author:  "Alice",
					Email:   "alice@example.com",
					Message: "fix(api): resolve null pointer issue\n\nThis fixes the null pointer that occurred when...",
					IsMerge: false,
				},
			},
			minAuthors:  1,
			checkScores: true,
		},
		{
			name: "poor hygiene commits",
			commits: []gitcmd.CommitData{
				{
					Author:  "Bob",
					Email:   "bob@example.com",
					Message: "fix",
					IsMerge: false,
				},
				{
					Author:  "Bob",
					Email:   "bob@example.com",
					Message: "update",
					IsMerge: false,
				},
				{
					Author:  "Bob",
					Email:   "bob@example.com",
					Message: "wip",
					IsMerge: false,
				},
			},
			minAuthors:  1,
			checkScores: true,
		},
		{
			name: "merge commits excluded",
			commits: []gitcmd.CommitData{
				{
					Author:  "Alice",
					Email:   "alice@example.com",
					Message: "Merge pull request #100",
					IsMerge: true,
				},
				{
					Author:  "Alice",
					Email:   "alice@example.com",
					Message: "feat: actual feature",
					IsMerge: false,
				},
			},
			minAuthors:  1,
			checkScores: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := ProcessCommitHygiene(tt.commits)

			if len(report.Authors) < tt.minAuthors {
				t.Errorf("expected at least %d authors, got %d", tt.minAuthors, len(report.Authors))
			}

			if tt.checkScores && len(report.Authors) > 0 {
				for _, author := range report.Authors {
					if author.HygieneScore < 0 || author.HygieneScore > 100 {
						t.Errorf("invalid hygiene score: %.2f", author.HygieneScore)
					}
					if author.TotalCommits == 0 {
						t.Error("author has 0 total commits")
					}
				}
			}
		})
	}
}

func TestCalculateHygieneScore(t *testing.T) {
	tests := []struct {
		name     string
		record   models.CommitHygieneRecord
		minScore float64
		maxScore float64
	}{
		{
			name: "perfect hygiene",
			record: models.CommitHygieneRecord{
				TooShortPct:     0,
				VaguePct:        0,
				ConventionalPct: 100,
				IssueRefPct:     100,
				BodyPct:         100,
			},
			minScore: 90,
			maxScore: 110,
		},
		{
			name: "poor hygiene",
			record: models.CommitHygieneRecord{
				TooShortPct:     50,
				VaguePct:        50,
				ConventionalPct: 0,
				IssueRefPct:     0,
				BodyPct:         0,
			},
			minScore: 0,
			maxScore: 50,
		},
		{
			name: "average hygiene",
			record: models.CommitHygieneRecord{
				TooShortPct:     20,
				VaguePct:        10,
				ConventionalPct: 70,
				IssueRefPct:     60,
				BodyPct:         40,
			},
			minScore: 80,
			maxScore: 90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateHygieneScore(tt.record)

			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("hygiene score %.2f outside expected range [%.2f, %.2f]",
					score, tt.minScore, tt.maxScore)
			}

			if score < 0 || score > 100 {
				t.Errorf("hygiene score %.2f outside valid range [0, 100]", score)
			}
		})
	}
}

func TestScaleTo100(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		max   float64
		want  float64
	}{
		{"zero value", 0, 100, 0},
		{"zero max", 100, 0, 0},
		{"halfway", 50, 100, 50},
		{"full", 100, 100, 100},
		{"over max", 150, 100, 100},
		{"negative value", -10, 100, 0},
		{"negative max", 50, -100, 0},
		{"decimal", 33.3, 100, 33.3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scaleTo100(tt.value, tt.max)
			diff := got - tt.want
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.1 {
				t.Errorf("scaleTo100(%v, %v) = %v, want %v", tt.value, tt.max, got, tt.want)
			}
		})
	}
}

func TestClampScore(t *testing.T) {
	tests := []struct {
		name  string
		score float64
		want  int
	}{
		{"zero", 0, 0},
		{"negative", -10, 0},
		{"positive", 50, 50},
		{"max", 100, 100},
		{"over max", 150, 100},
		{"decimal rounding", 49.6, 50},
		{"decimal rounding down", 49.4, 49},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampScore(tt.score)
			if got != tt.want {
				t.Errorf("clampScore(%v) = %v, want %v", tt.score, got, tt.want)
			}
		})
	}
}

func TestHotspotRisk(t *testing.T) {
	tests := []struct {
		score int
		want  string
	}{
		{100, "CRITICAL"},
		{80, "CRITICAL"},
		{79, "HIGH"},
		{65, "HIGH"},
		{64, "MEDIUM"},
		{50, "MEDIUM"},
		{49, "WATCH"},
		{35, "WATCH"},
		{34, "LOW"},
		{0, "LOW"},
		{-10, "LOW"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := hotspotRisk(tt.score)
			if got != tt.want {
				t.Errorf("hotspotRisk(%d) = %q, want %q", tt.score, got, tt.want)
			}
		})
	}
}

func TestSpinner(t *testing.T) {
	t.Run("spinner with show=false", func(t *testing.T) {
		s := newSpinner("test", false)
		if s != nil {
			t.Error("expected nil spinner when show=false")
		}
	})

	t.Run("spinner with show=true", func(t *testing.T) {
		s := newSpinner("test", true)
		if s == nil {
			t.Error("expected non-nil spinner when show=true")
			return
		}

		// Stop should not panic
		s.Stop()
	})

	t.Run("spinner with show=true", func(t *testing.T) {
		s := newSpinner("test", true)
		if s == nil {
			t.Error("expected non-nil spinner when show=true")
			return
		}
		s.Stop()
	})

	t.Run("stop nil spinner", func(t *testing.T) {
		var s *Spinner
		s.Stop()
	})
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
