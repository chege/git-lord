package processor

import (
	"testing"
	"time"

	"github.com/chege/git-lord/internal/gitcmd"
	"github.com/chege/git-lord/internal/models"
)

// branchHealthInput encapsulates all inputs needed for testing branch health processing.
type branchHealthInput struct {
	branches      []gitcmd.BranchInfo
	defaultBranch string
	daysThreshold int
	includeRemote bool
	// Mock function results for git operations
	mergedBranches map[string]bool
	aheadBehind    map[string]struct{ ahead, behind int }
}

// processBranchHealthForTest processes BranchInfo slices directly for testing.
// This extracts the core logic from ProcessBranchHealth without git command dependencies.
func processBranchHealthForTest(input branchHealthInput, now time.Time) models.BranchHealthReport {
	var records []models.BranchHealthRecord

	for _, branch := range input.branches {
		if branch.IsRemote && !input.includeRemote {
			continue
		}

		daysSinceLastCommit := int(now.Sub(branch.LastCommit).Hours() / 24)

		isMerged := input.mergedBranches[branch.Name]

		ahead := 0
		behind := 0
		if ab, ok := input.aheadBehind[branch.Name]; ok {
			ahead = ab.ahead
			behind = ab.behind
		}

		isStale := daysSinceLastCommit > input.daysThreshold
		isUnmerged := !isMerged
		isOrphaned := isStale && isUnmerged

		record := models.BranchHealthRecord{
			Name:                branch.Name,
			IsRemote:            branch.IsRemote,
			IsHead:              branch.IsHead,
			LastCommit:          branch.LastCommit,
			LastAuthor:          branch.LastAuthor,
			LastSubject:         branch.LastSubject,
			CommitCount:         branch.CommitCount,
			IsMerged:            isMerged,
			Behind:              behind,
			Ahead:               ahead,
			DaysSinceLastCommit: daysSinceLastCommit,
			StaleDays:           input.daysThreshold,
			IsStale:             isStale,
			IsUnmerged:          isUnmerged,
			IsOrphaned:          isOrphaned,
		}

		records = append(records, record)
	}

	for i := 0; i < len(records); i++ {
		for j := i + 1; j < len(records); j++ {
			if records[j].DaysSinceLastCommit > records[i].DaysSinceLastCommit {
				records[i], records[j] = records[j], records[i]
			}
		}
	}

	totalCount := len(records)
	staleCount := 0
	unmergedCount := 0
	orphanedCount := 0

	for _, record := range records {
		if record.IsStale {
			staleCount++
		}
		if record.IsUnmerged {
			unmergedCount++
		}
		if record.IsOrphaned {
			orphanedCount++
		}
	}

	return models.BranchHealthReport{
		Branches:      records,
		DefaultBranch: input.defaultBranch,
		TotalCount:    totalCount,
		StaleCount:    staleCount,
		UnmergedCount: unmergedCount,
		OrphanedCount: orphanedCount,
	}
}

func TestProcessBranchHealth_Empty(t *testing.T) {
	now := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)

	input := branchHealthInput{
		branches:       []gitcmd.BranchInfo{},
		defaultBranch:  "main",
		daysThreshold:  30,
		includeRemote:  false,
		mergedBranches: map[string]bool{},
		aheadBehind:    map[string]struct{ ahead, behind int }{},
	}

	report := processBranchHealthForTest(input, now)

	if len(report.Branches) != 0 {
		t.Errorf("expected empty branches slice, got %d branches", len(report.Branches))
	}
	if report.TotalCount != 0 {
		t.Errorf("expected TotalCount=0, got %d", report.TotalCount)
	}
	if report.StaleCount != 0 {
		t.Errorf("expected StaleCount=0, got %d", report.StaleCount)
	}
	if report.UnmergedCount != 0 {
		t.Errorf("expected UnmergedCount=0, got %d", report.UnmergedCount)
	}
	if report.OrphanedCount != 0 {
		t.Errorf("expected OrphanedCount=0, got %d", report.OrphanedCount)
	}
	if report.DefaultBranch != "main" {
		t.Errorf("expected DefaultBranch='main', got %q", report.DefaultBranch)
	}
}

func TestProcessBranchHealth_StaleDetection(t *testing.T) {
	now := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		daysThreshold int
		lastCommit    time.Time
		expectedStale bool
		expectedDays  int
	}{
		{
			name:          "fresh branch - exactly at threshold",
			daysThreshold: 30,
			lastCommit:    now.AddDate(0, 0, -30),
			expectedStale: false,
			expectedDays:  30,
		},
		{
			name:          "fresh branch - one day under threshold",
			daysThreshold: 30,
			lastCommit:    now.AddDate(0, 0, -29),
			expectedStale: false,
			expectedDays:  29,
		},
		{
			name:          "stale branch - one day over threshold",
			daysThreshold: 30,
			lastCommit:    now.AddDate(0, 0, -31),
			expectedStale: true,
			expectedDays:  31,
		},
		{
			name:          "very stale branch",
			daysThreshold: 30,
			lastCommit:    now.AddDate(0, 0, -100),
			expectedStale: true,
			expectedDays:  100,
		},
		{
			name:          "zero threshold - fresh branch",
			daysThreshold: 0,
			lastCommit:    now,
			expectedStale: false,
			expectedDays:  0,
		},
		{
			name:          "zero threshold - one day old",
			daysThreshold: 0,
			lastCommit:    now.AddDate(0, 0, -1),
			expectedStale: true,
			expectedDays:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := branchHealthInput{
				branches: []gitcmd.BranchInfo{
					{
						Name:        "test-branch",
						IsRemote:    false,
						IsHead:      false,
						LastCommit:  tt.lastCommit,
						LastAuthor:  "Test Author",
						LastSubject: "Test commit",
						CommitCount: 5,
					},
				},
				defaultBranch:  "main",
				daysThreshold:  tt.daysThreshold,
				includeRemote:  false,
				mergedBranches: map[string]bool{"test-branch": true},
				aheadBehind:    map[string]struct{ ahead, behind int }{},
			}

			report := processBranchHealthForTest(input, now)

			if len(report.Branches) != 1 {
				t.Fatalf("expected 1 branch, got %d", len(report.Branches))
			}

			record := report.Branches[0]
			if record.IsStale != tt.expectedStale {
				t.Errorf("expected IsStale=%v, got %v (days: %d, threshold: %d)",
					tt.expectedStale, record.IsStale, record.DaysSinceLastCommit, tt.daysThreshold)
			}
			if record.DaysSinceLastCommit != tt.expectedDays {
				t.Errorf("expected DaysSinceLastCommit=%d, got %d",
					tt.expectedDays, record.DaysSinceLastCommit)
			}
		})
	}
}

func TestProcessBranchHealth_MergedDetection(t *testing.T) {
	now := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name             string
		isMerged         bool
		expectedMerged   bool
		expectedUnmerged bool
	}{
		{
			name:             "merged branch",
			isMerged:         true,
			expectedMerged:   true,
			expectedUnmerged: false,
		},
		{
			name:             "unmerged branch",
			isMerged:         false,
			expectedMerged:   false,
			expectedUnmerged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := branchHealthInput{
				branches: []gitcmd.BranchInfo{
					{
						Name:        "feature-branch",
						IsRemote:    false,
						IsHead:      false,
						LastCommit:  now.AddDate(0, 0, -5),
						LastAuthor:  "Test Author",
						LastSubject: "Feature commit",
						CommitCount: 3,
					},
				},
				defaultBranch:  "main",
				daysThreshold:  30,
				includeRemote:  false,
				mergedBranches: map[string]bool{"feature-branch": tt.isMerged},
				aheadBehind:    map[string]struct{ ahead, behind int }{},
			}

			report := processBranchHealthForTest(input, now)

			if len(report.Branches) != 1 {
				t.Fatalf("expected 1 branch, got %d", len(report.Branches))
			}

			record := report.Branches[0]
			if record.IsMerged != tt.expectedMerged {
				t.Errorf("expected IsMerged=%v, got %v", tt.expectedMerged, record.IsMerged)
			}
			if record.IsUnmerged != tt.expectedUnmerged {
				t.Errorf("expected IsUnmerged=%v, got %v", tt.expectedUnmerged, record.IsUnmerged)
			}
		})
	}
}

func TestProcessBranchHealth_OrphanDetection(t *testing.T) {
	now := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name             string
		daysOld          int
		isMerged         bool
		expectedStale    bool
		expectedUnmerged bool
		expectedOrphaned bool
	}{
		{
			name:             "fresh and merged - not orphaned",
			daysOld:          5,
			isMerged:         true,
			expectedStale:    false,
			expectedUnmerged: false,
			expectedOrphaned: false,
		},
		{
			name:             "fresh and unmerged - not orphaned",
			daysOld:          5,
			isMerged:         false,
			expectedStale:    false,
			expectedUnmerged: true,
			expectedOrphaned: false,
		},
		{
			name:             "stale but merged - not orphaned",
			daysOld:          40,
			isMerged:         true,
			expectedStale:    true,
			expectedUnmerged: false,
			expectedOrphaned: false,
		},
		{
			name:             "stale and unmerged - IS orphaned",
			daysOld:          40,
			isMerged:         false,
			expectedStale:    true,
			expectedUnmerged: true,
			expectedOrphaned: true,
		},
		{
			name:             "very old and unmerged - IS orphaned",
			daysOld:          365,
			isMerged:         false,
			expectedStale:    true,
			expectedUnmerged: true,
			expectedOrphaned: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := branchHealthInput{
				branches: []gitcmd.BranchInfo{
					{
						Name:        "orphan-test",
						IsRemote:    false,
						IsHead:      false,
						LastCommit:  now.AddDate(0, 0, -tt.daysOld),
						LastAuthor:  "Test Author",
						LastSubject: "Test commit",
						CommitCount: 3,
					},
				},
				defaultBranch:  "main",
				daysThreshold:  30,
				includeRemote:  false,
				mergedBranches: map[string]bool{"orphan-test": tt.isMerged},
				aheadBehind:    map[string]struct{ ahead, behind int }{},
			}

			report := processBranchHealthForTest(input, now)

			if len(report.Branches) != 1 {
				t.Fatalf("expected 1 branch, got %d", len(report.Branches))
			}

			record := report.Branches[0]
			if record.IsStale != tt.expectedStale {
				t.Errorf("expected IsStale=%v, got %v", tt.expectedStale, record.IsStale)
			}
			if record.IsUnmerged != tt.expectedUnmerged {
				t.Errorf("expected IsUnmerged=%v, got %v", tt.expectedUnmerged, record.IsUnmerged)
			}
			if record.IsOrphaned != tt.expectedOrphaned {
				t.Errorf("expected IsOrphaned=%v, got %v", tt.expectedOrphaned, record.IsOrphaned)
			}
		})
	}
}

func TestProcessBranchHealth_DaysThreshold(t *testing.T) {
	now := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		daysThreshold int
		daysOld       int
		expectedStale bool
	}{
		{
			name:          "threshold 30 - fresh at 30 days",
			daysThreshold: 30,
			daysOld:       30,
			expectedStale: false,
		},
		{
			name:          "threshold 30 - stale at 31 days",
			daysThreshold: 30,
			daysOld:       31,
			expectedStale: true,
		},
		{
			name:          "threshold 90 - fresh at 90 days",
			daysThreshold: 90,
			daysOld:       90,
			expectedStale: false,
		},
		{
			name:          "threshold 90 - stale at 91 days",
			daysThreshold: 90,
			daysOld:       91,
			expectedStale: true,
		},
		{
			name:          "threshold 365 - fresh at 365 days",
			daysThreshold: 365,
			daysOld:       365,
			expectedStale: false,
		},
		{
			name:          "threshold 365 - stale at 366 days",
			daysThreshold: 365,
			daysOld:       366,
			expectedStale: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := branchHealthInput{
				branches: []gitcmd.BranchInfo{
					{
						Name:        "threshold-test",
						IsRemote:    false,
						IsHead:      false,
						LastCommit:  now.AddDate(0, 0, -tt.daysOld),
						LastAuthor:  "Test Author",
						LastSubject: "Test commit",
						CommitCount: 5,
					},
				},
				defaultBranch:  "main",
				daysThreshold:  tt.daysThreshold,
				includeRemote:  false,
				mergedBranches: map[string]bool{"threshold-test": false},
				aheadBehind:    map[string]struct{ ahead, behind int }{},
			}

			report := processBranchHealthForTest(input, now)

			if len(report.Branches) != 1 {
				t.Fatalf("expected 1 branch, got %d", len(report.Branches))
			}

			record := report.Branches[0]
			if record.IsStale != tt.expectedStale {
				t.Errorf("expected IsStale=%v, got %v (threshold=%d, days=%d)",
					tt.expectedStale, record.IsStale, tt.daysThreshold, tt.daysOld)
			}
			if record.StaleDays != tt.daysThreshold {
				t.Errorf("expected StaleDays=%d, got %d",
					tt.daysThreshold, record.StaleDays)
			}
		})
	}
}

func TestProcessBranchHealth_Sorting(t *testing.T) {
	now := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)

	input := branchHealthInput{
		branches: []gitcmd.BranchInfo{
			{
				Name:        "medium-old",
				IsRemote:    false,
				IsHead:      false,
				LastCommit:  now.AddDate(0, 0, -30),
				LastAuthor:  "Author A",
				LastSubject: "Medium old commit",
				CommitCount: 5,
			},
			{
				Name:        "very-old",
				IsRemote:    false,
				IsHead:      false,
				LastCommit:  now.AddDate(0, 0, -100),
				LastAuthor:  "Author B",
				LastSubject: "Very old commit",
				CommitCount: 10,
			},
			{
				Name:        "fresh",
				IsRemote:    false,
				IsHead:      false,
				LastCommit:  now.AddDate(0, 0, -5),
				LastAuthor:  "Author C",
				LastSubject: "Fresh commit",
				CommitCount: 2,
			},
			{
				Name:        "oldest",
				IsRemote:    false,
				IsHead:      false,
				LastCommit:  now.AddDate(0, 0, -200),
				LastAuthor:  "Author D",
				LastSubject: "Oldest commit",
				CommitCount: 15,
			},
		},
		defaultBranch:  "main",
		daysThreshold:  30,
		includeRemote:  false,
		mergedBranches: map[string]bool{},
		aheadBehind:    map[string]struct{ ahead, behind int }{},
	}

	report := processBranchHealthForTest(input, now)

	if len(report.Branches) != 4 {
		t.Fatalf("expected 4 branches, got %d", len(report.Branches))
	}

	expectedOrder := []string{"oldest", "very-old", "medium-old", "fresh"}
	expectedDays := []int{200, 100, 30, 5}

	for i, expectedName := range expectedOrder {
		if report.Branches[i].Name != expectedName {
			t.Errorf("position %d: expected %q, got %q",
				i, expectedName, report.Branches[i].Name)
		}
		if report.Branches[i].DaysSinceLastCommit != expectedDays[i] {
			t.Errorf("position %d: expected %d days, got %d",
				i, expectedDays[i], report.Branches[i].DaysSinceLastCommit)
		}
	}
}

func TestProcessBranchHealth_ReportCounters(t *testing.T) {
	now := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)

	input := branchHealthInput{
		branches: []gitcmd.BranchInfo{
			{
				Name:        "fresh-merged",
				IsRemote:    false,
				IsHead:      false,
				LastCommit:  now.AddDate(0, 0, -5),
				LastAuthor:  "Author",
				LastSubject: "Fresh merged",
				CommitCount: 3,
			},
			{
				Name:        "fresh-unmerged",
				IsRemote:    false,
				IsHead:      false,
				LastCommit:  now.AddDate(0, 0, -10),
				LastAuthor:  "Author",
				LastSubject: "Fresh unmerged",
				CommitCount: 3,
			},
			{
				Name:        "stale-merged",
				IsRemote:    false,
				IsHead:      false,
				LastCommit:  now.AddDate(0, 0, -40),
				LastAuthor:  "Author",
				LastSubject: "Stale merged",
				CommitCount: 5,
			},
			{
				Name:        "stale-unmerged-1",
				IsRemote:    false,
				IsHead:      false,
				LastCommit:  now.AddDate(0, 0, -50),
				LastAuthor:  "Author",
				LastSubject: "Stale unmerged 1",
				CommitCount: 5,
			},
			{
				Name:        "stale-unmerged-2",
				IsRemote:    false,
				IsHead:      false,
				LastCommit:  now.AddDate(0, 0, -60),
				LastAuthor:  "Author",
				LastSubject: "Stale unmerged 2",
				CommitCount: 5,
			},
		},
		defaultBranch: "main",
		daysThreshold: 30,
		includeRemote: false,
		mergedBranches: map[string]bool{
			"fresh-merged":     true,
			"fresh-unmerged":   false,
			"stale-merged":     true,
			"stale-unmerged-1": false,
			"stale-unmerged-2": false,
		},
		aheadBehind: map[string]struct{ ahead, behind int }{},
	}

	report := processBranchHealthForTest(input, now)

	if report.TotalCount != 5 {
		t.Errorf("expected TotalCount=5, got %d", report.TotalCount)
	}

	if report.StaleCount != 3 {
		t.Errorf("expected StaleCount=3, got %d", report.StaleCount)
	}

	if report.UnmergedCount != 3 {
		t.Errorf("expected UnmergedCount=3, got %d", report.UnmergedCount)
	}

	if report.OrphanedCount != 2 {
		t.Errorf("expected OrphanedCount=2, got %d", report.OrphanedCount)
	}
}

func TestProcessBranchHealth_RemoteFiltering(t *testing.T) {
	now := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		includeRemote bool
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "exclude remotes",
			includeRemote: false,
			expectedCount: 2,
			expectedNames: []string{"local-branch-1", "local-branch-2"},
		},
		{
			name:          "include remotes",
			includeRemote: true,
			expectedCount: 4,
			expectedNames: []string{"local-branch-1", "local-branch-2", "origin/remote-1", "origin/remote-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := branchHealthInput{
				branches: []gitcmd.BranchInfo{
					{
						Name:        "local-branch-1",
						IsRemote:    false,
						IsHead:      false,
						LastCommit:  now,
						LastAuthor:  "Author",
						LastSubject: "Local 1",
						CommitCount: 1,
					},
					{
						Name:        "origin/remote-1",
						IsRemote:    true,
						IsHead:      false,
						LastCommit:  now,
						LastAuthor:  "Author",
						LastSubject: "Remote 1",
						CommitCount: 1,
					},
					{
						Name:        "local-branch-2",
						IsRemote:    false,
						IsHead:      false,
						LastCommit:  now,
						LastAuthor:  "Author",
						LastSubject: "Local 2",
						CommitCount: 1,
					},
					{
						Name:        "origin/remote-2",
						IsRemote:    true,
						IsHead:      false,
						LastCommit:  now,
						LastAuthor:  "Author",
						LastSubject: "Remote 2",
						CommitCount: 1,
					},
				},
				defaultBranch:  "main",
				daysThreshold:  30,
				includeRemote:  tt.includeRemote,
				mergedBranches: map[string]bool{},
				aheadBehind:    map[string]struct{ ahead, behind int }{},
			}

			report := processBranchHealthForTest(input, now)

			if report.TotalCount != tt.expectedCount {
				t.Errorf("expected TotalCount=%d, got %d", tt.expectedCount, report.TotalCount)
			}
			if len(report.Branches) != tt.expectedCount {
				t.Errorf("expected %d branches, got %d", tt.expectedCount, len(report.Branches))
			}

			nameSet := make(map[string]bool)
			for _, b := range report.Branches {
				nameSet[b.Name] = true
			}
			for _, expectedName := range tt.expectedNames {
				if !nameSet[expectedName] {
					t.Errorf("expected branch %q not found in report", expectedName)
				}
			}
		})
	}
}

func TestProcessBranchHealth_AheadBehind(t *testing.T) {
	now := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)

	input := branchHealthInput{
		branches: []gitcmd.BranchInfo{
			{
				Name:        "behind-branch",
				IsRemote:    false,
				IsHead:      false,
				LastCommit:  now.AddDate(0, 0, -5),
				LastAuthor:  "Author",
				LastSubject: "Behind",
				CommitCount: 3,
			},
			{
				Name:        "ahead-branch",
				IsRemote:    false,
				IsHead:      false,
				LastCommit:  now.AddDate(0, 0, -5),
				LastAuthor:  "Author",
				LastSubject: "Ahead",
				CommitCount: 5,
			},
			{
				Name:        "synced-branch",
				IsRemote:    false,
				IsHead:      false,
				LastCommit:  now.AddDate(0, 0, -5),
				LastAuthor:  "Author",
				LastSubject: "Synced",
				CommitCount: 2,
			},
		},
		defaultBranch:  "main",
		daysThreshold:  30,
		includeRemote:  false,
		mergedBranches: map[string]bool{},
		aheadBehind: map[string]struct{ ahead, behind int }{
			"behind-branch": {ahead: 0, behind: 5},
			"ahead-branch":  {ahead: 3, behind: 0},
			"synced-branch": {ahead: 0, behind: 0},
		},
	}

	report := processBranchHealthForTest(input, now)

	if len(report.Branches) != 3 {
		t.Fatalf("expected 3 branches, got %d", len(report.Branches))
	}

	for _, b := range report.Branches {
		switch b.Name {
		case "behind-branch":
			if b.Ahead != 0 || b.Behind != 5 {
				t.Errorf("behind-branch: expected Ahead=0, Behind=5, got Ahead=%d, Behind=%d",
					b.Ahead, b.Behind)
			}
		case "ahead-branch":
			if b.Ahead != 3 || b.Behind != 0 {
				t.Errorf("ahead-branch: expected Ahead=3, Behind=0, got Ahead=%d, Behind=%d",
					b.Ahead, b.Behind)
			}
		case "synced-branch":
			if b.Ahead != 0 || b.Behind != 0 {
				t.Errorf("synced-branch: expected Ahead=0, Behind=0, got Ahead=%d, Behind=%d",
					b.Ahead, b.Behind)
			}
		}
	}
}

func TestProcessBranchHealth_RecordFields(t *testing.T) {
	now := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	commitTime := now.AddDate(0, 0, -15)

	input := branchHealthInput{
		branches: []gitcmd.BranchInfo{
			{
				Name:        "test/feature-branch",
				IsRemote:    false,
				IsHead:      true,
				LastCommit:  commitTime,
				LastAuthor:  "John Doe",
				LastSubject: "Add new feature",
				CommitCount: 42,
			},
		},
		defaultBranch: "main",
		daysThreshold: 30,
		includeRemote: false,
		mergedBranches: map[string]bool{
			"test/feature-branch": false,
		},
		aheadBehind: map[string]struct{ ahead, behind int }{
			"test/feature-branch": {ahead: 5, behind: 2},
		},
	}

	report := processBranchHealthForTest(input, now)

	if len(report.Branches) != 1 {
		t.Fatalf("expected 1 branch, got %d", len(report.Branches))
	}

	record := report.Branches[0]

	// Verify all fields are correctly copied/transformed
	if record.Name != "test/feature-branch" {
		t.Errorf("expected Name='test/feature-branch', got %q", record.Name)
	}
	if record.IsRemote != false {
		t.Errorf("expected IsRemote=false, got %v", record.IsRemote)
	}
	if record.IsHead != true {
		t.Errorf("expected IsHead=true, got %v", record.IsHead)
	}
	if !record.LastCommit.Equal(commitTime) {
		t.Errorf("expected LastCommit=%v, got %v", commitTime, record.LastCommit)
	}
	if record.LastAuthor != "John Doe" {
		t.Errorf("expected LastAuthor='John Doe', got %q", record.LastAuthor)
	}
	if record.LastSubject != "Add new feature" {
		t.Errorf("expected LastSubject='Add new feature', got %q", record.LastSubject)
	}
	if record.CommitCount != 42 {
		t.Errorf("expected CommitCount=42, got %d", record.CommitCount)
	}
	if record.IsMerged != false {
		t.Errorf("expected IsMerged=false, got %v", record.IsMerged)
	}
	if record.Ahead != 5 {
		t.Errorf("expected Ahead=5, got %d", record.Ahead)
	}
	if record.Behind != 2 {
		t.Errorf("expected Behind=2, got %d", record.Behind)
	}
	if record.DaysSinceLastCommit != 15 {
		t.Errorf("expected DaysSinceLastCommit=15, got %d", record.DaysSinceLastCommit)
	}
	if record.StaleDays != 30 {
		t.Errorf("expected StaleDays=30, got %d", record.StaleDays)
	}
	if record.IsStale != false {
		t.Errorf("expected IsStale=false, got %v", record.IsStale)
	}
	if record.IsUnmerged != true {
		t.Errorf("expected IsUnmerged=true, got %v", record.IsUnmerged)
	}
	if record.IsOrphaned != false {
		t.Errorf("expected IsOrphaned=false, got %v", record.IsOrphaned)
	}
}
