package processor

import (
	"sort"

	"github.com/chege/git-lord/internal/gitcmd"
	"github.com/chege/git-lord/internal/models"
)

// ProcessPulse calculates activity metrics from history without using blame.
func ProcessPulse(commits []gitcmd.CommitData, showProgress bool) []models.PulseStat {
	authors := make(map[string]*models.PulseStat)

	bar := newProgressBar(len(commits), "Analyzing commits", showProgress)

	for _, c := range commits {
		id := c.Email
		if _, ok := authors[id]; !ok {
			authors[id] = &models.PulseStat{
				Name: c.Author,
			}
		}
		authors[id].Commits++
		authors[id].Additions += c.Additions
		authors[id].Deletions += c.Deletions
		authors[id].Net += (c.Additions - c.Deletions)
		authors[id].Churn += (c.Additions + c.Deletions)
		authors[id].Files += c.Files

		if bar != nil {
			_ = bar.Add(1)
		}
	}

	if bar != nil {
		_ = bar.Finish()
		_ = bar.Clear()
	}

	var stats []models.PulseStat
	for _, s := range authors {
		stats = append(stats, *s)
	}

	sort.SliceStable(stats, func(i, j int) bool {
		return stats[i].Commits > stats[j].Commits
	})

	return stats
}

// ProcessTrends groups additions and deletions by month.
func ProcessTrends(commits []gitcmd.CommitData) []models.TrendStat {
	monthMap := make(map[string]*models.TrendStat)

	for _, c := range commits {
		period := c.Date.Format("2006-01")
		if _, ok := monthMap[period]; !ok {
			monthMap[period] = &models.TrendStat{Period: period}
		}
		monthMap[period].Additions += c.Additions
		monthMap[period].Deletions += c.Deletions
		monthMap[period].Net += (c.Additions - c.Deletions)
	}

	var trends []models.TrendStat
	for _, t := range monthMap {
		trends = append(trends, *t)
	}

	sort.Slice(trends, func(i, j int) bool {
		return trends[i].Period < trends[j].Period
	})

	return trends
}
