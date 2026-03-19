package processor

import (
	"math"
	"sort"
	"time"

	"github.com/chege/git-lord/internal/gitcmd"
	"github.com/chege/git-lord/internal/models"
)

const (
	defaultHotspotWindowDays = 30
	hotspotMinLOC            = 50
)

// ProcessHotspots ranks tracked files by recent churn, ownership concentration, and size.
func ProcessHotspots(res ResultExtended, commits []gitcmd.CommitData, windowDays int, now time.Time) models.HotspotReport {
	if windowDays <= 0 {
		windowDays = defaultHotspotWindowDays
	}
	if now.IsZero() {
		now = time.Now()
	}

	report := models.HotspotReport{WindowDays: windowDays}
	churnByFile := hotspotChurnByFile(res.FileOwners, commits, now.AddDate(0, 0, -windowDays))

	for path, owners := range res.FileOwners {
		loc := totalOwnedLines(owners)
		if loc < hotspotMinLOC {
			continue
		}

		churn, ok := churnByFile[path]
		if !ok || churn.Churn() <= 0 {
			continue
		}

		primaryOwner, ownerLines := primaryOwner(owners)
		ownershipPct := 0.0
		if loc > 0 {
			ownershipPct = float64(ownerLines) / float64(loc) * 100
		}

		churnScore := clampScore(0.7*scaleTo100(float64(churn.Churn()), 200) + 0.3*scaleTo100(float64(churn.RecentCommits), 8))
		ownershipScore := 0
		if ownershipPct > 50 {
			ownershipScore = clampScore((ownershipPct - 50) / 45 * 100)
		}
		sizeScore := clampScore(scaleTo100(float64(loc), 800))
		score := clampScore(0.5*float64(churnScore) + 0.3*float64(ownershipScore) + 0.2*float64(sizeScore))

		report.Hotspots = append(report.Hotspots, models.HotspotRecord{
			Path:           path,
			Score:          score,
			Risk:           hotspotRisk(score),
			LOC:            loc,
			RecentChurn:    churn.Churn(),
			RecentCommits:  churn.RecentCommits,
			PrimaryOwner:   primaryOwner,
			OwnerLines:     ownerLines,
			OwnershipPct:   ownershipPct,
			ActiveOwners:   len(owners),
			ChurnScore:     churnScore,
			OwnershipScore: ownershipScore,
			SizeScore:      sizeScore,
		})
	}

	sort.Slice(report.Hotspots, func(i, j int) bool {
		left := report.Hotspots[i]
		right := report.Hotspots[j]
		if left.Score != right.Score {
			return left.Score > right.Score
		}
		if left.RecentChurn != right.RecentChurn {
			return left.RecentChurn > right.RecentChurn
		}
		if left.OwnershipPct != right.OwnershipPct {
			return left.OwnershipPct > right.OwnershipPct
		}
		if left.LOC != right.LOC {
			return left.LOC > right.LOC
		}
		return left.Path < right.Path
	})

	return report
}

func hotspotChurnByFile(tracked map[string]map[string]int, commits []gitcmd.CommitData, cutoff time.Time) map[string]gitcmd.FileChurn {
	churnByFile := make(map[string]gitcmd.FileChurn)
	trackedPaths := make(map[string]struct{}, len(tracked))
	for path := range tracked {
		trackedPaths[path] = struct{}{}
	}

	for _, commit := range commits {
		if commit.Date.Before(cutoff) {
			continue
		}

		seenInCommit := make(map[string]struct{})
		for _, stat := range commit.FileStats {
			if _, ok := trackedPaths[stat.Path]; !ok {
				continue
			}
			if stat.Additions+stat.Deletions <= 0 {
				continue
			}

			churn := churnByFile[stat.Path]
			churn.Path = stat.Path
			churn.Additions += stat.Additions
			churn.Deletions += stat.Deletions
			if _, counted := seenInCommit[stat.Path]; !counted {
				churn.RecentCommits++
				seenInCommit[stat.Path] = struct{}{}
			}
			churnByFile[stat.Path] = churn
		}
	}

	return churnByFile
}

func totalOwnedLines(owners map[string]int) int {
	total := 0
	for _, loc := range owners {
		total += loc
	}
	return total
}

func primaryOwner(owners map[string]int) (string, int) {
	var owner string
	maxLines := 0
	for email, loc := range owners {
		if loc > maxLines || (loc == maxLines && (owner == "" || email < owner)) {
			owner = email
			maxLines = loc
		}
	}
	return owner, maxLines
}

func scaleTo100(value, max float64) float64 {
	if max <= 0 {
		return 0
	}
	if value <= 0 {
		return 0
	}
	if value >= max {
		return 100
	}
	return value / max * 100
}

func clampScore(score float64) int {
	if score <= 0 {
		return 0
	}
	if score >= 100 {
		return 100
	}
	return int(math.Round(score))
}

func hotspotRisk(score int) string {
	switch {
	case score >= 80:
		return "CRITICAL"
	case score >= 65:
		return "HIGH"
	case score >= 50:
		return "MEDIUM"
	case score >= 35:
		return "WATCH"
	default:
		return "LOW"
	}
}
