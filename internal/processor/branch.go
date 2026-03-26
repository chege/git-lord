package processor

import (
	"context"
	"sort"
	"time"

	"github.com/chege/git-lord/internal/gitcmd"
	"github.com/chege/git-lord/internal/models"
)

// ProcessBranchHealth analyzes all branches and returns a comprehensive health report.
func ProcessBranchHealth(ctx context.Context, daysThreshold int, includeRemote bool) models.BranchHealthReport {
	branches, err := gitcmd.ListBranches(ctx)
	if err != nil {
		return models.BranchHealthReport{
			Branches: []models.BranchHealthRecord{},
		}
	}

	defaultBranch, err := gitcmd.GetDefaultBranch(ctx)
	if err != nil {
		defaultBranch = "main"
	}

	var records []models.BranchHealthRecord
	now := time.Now()

	for _, branch := range branches {
		if branch.IsRemote && !includeRemote {
			continue
		}

		daysSinceLastCommit := int(now.Sub(branch.LastCommit).Hours() / 24)

		isMerged, err := gitcmd.IsBranchMerged(ctx, branch.Name, defaultBranch)
		if err != nil {
			isMerged = false
		}

		ahead, behind, err := gitcmd.GetBranchAheadBehind(ctx, branch.Name, defaultBranch)
		if err != nil {
			ahead = 0
			behind = 0
		}

		isStale := daysSinceLastCommit > daysThreshold
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
			StaleDays:           daysThreshold,
			IsStale:             isStale,
			IsUnmerged:          isUnmerged,
			IsOrphaned:          isOrphaned,
		}

		records = append(records, record)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].DaysSinceLastCommit > records[j].DaysSinceLastCommit
	})

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
		DefaultBranch: defaultBranch,
		TotalCount:    totalCount,
		StaleCount:    staleCount,
		UnmergedCount: unmergedCount,
		OrphanedCount: orphanedCount,
	}
}
