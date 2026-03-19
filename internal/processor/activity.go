package processor

import (
	"sort"
	"strings"

	"github.com/chege/git-lord/internal/gitcmd"
	"github.com/chege/git-lord/internal/models"
)

// ProcessPulse calculates activity metrics from history without using blame.
func ProcessPulse(commits []gitcmd.CommitData, sortMetric string, showProgress bool) []models.PulseStat {
	authors := make(map[string]*models.PulseStat)

	spinner := newSpinner("Analyzing commits", showProgress)
	defer spinner.Stop()

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
	}

	var stats []models.PulseStat
	for _, s := range authors {
		stats = append(stats, *s)
	}

	sort.SliceStable(stats, func(i, j int) bool {
		left := pulseSortValue(stats[i], sortMetric)
		right := pulseSortValue(stats[j], sortMetric)
		if left == right {
			return stats[i].Name < stats[j].Name
		}
		return left > right
	})

	return stats
}

func pulseSortValue(stat models.PulseStat, sortMetric string) int {
	switch strings.ToLower(sortMetric) {
	case "additions":
		return stat.Additions
	case "deletions":
		return stat.Deletions
	case "net":
		return stat.Net
	case "churn":
		return stat.Churn
	case "files", "fils":
		return stat.Files
	case "commits", "coms", "":
		fallthrough
	default:
		return stat.Commits
	}
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
