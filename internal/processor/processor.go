package processor

import (
	"runtime"
	"sort"
	"sync"

	"github.com/schollz/progressbar/v3"

	"github.com/christopher/git-lord/internal/gitcmd"
	"github.com/christopher/git-lord/internal/metrics"
	"github.com/christopher/git-lord/internal/models"
)

// ProcessPulse calculates activity metrics from history without using blame.
func ProcessPulse(commits []gitcmd.CommitData) []models.PulseStat {
	authors := make(map[string]*models.PulseStat)

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
		authors[id].Churn += (c.Additions + c.Deletions)
		authors[id].Files += c.Files
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

// ProcessRepository orchestrates the gathering and aggregation of metrics.
func ProcessRepository(files []string, commits []gitcmd.CommitData, showProgress bool) models.Result {
	numCPU := runtime.NumCPU()
	filesChan := make(chan string, len(files))
	for _, f := range files {
		filesChan <- f
	}
	close(filesChan)

	var bar *progressbar.ProgressBar
	if showProgress {
		bar = progressbar.NewOptions(len(files),
			progressbar.OptionSetDescription("Blaming files"),
			progressbar.OptionSetWidth(15),
			progressbar.OptionShowCount(),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "=",
				SaucerHead:    ">",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}),
		)
	}

	var wg sync.WaitGroup
	resultsChan := make(chan gitcmd.BlameData, numCPU*2)

	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range filesChan {
				blameData, err := gitcmd.GetBlame(file)
				if err == nil {
					resultsChan <- blameData
				}
				if bar != nil {
					_ = bar.Add(1)
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		if bar != nil {
			_ = bar.Finish()
			_ = bar.Clear()
		}
		close(resultsChan)
	}()

	authors := make(map[string]*models.AuthorMetrics)
	var global models.GlobalMetrics

	// Merge commits and prepare timestamps for hours/months
	// We do this first to populate the Names for the Emails
	authorTimestamps := make(map[string][]int64)
	var allTimestamps []int64

	for _, commit := range commits {
		// Use Email as the unique identifier
		id := commit.Email
		if _, ok := authors[id]; !ok {
			authors[id] = &models.AuthorMetrics{
				Name:        commit.Author,
				Email:       commit.Email,
				FirstCommit: commit.Timestamp,
				LastCommit:  commit.Timestamp,
			}
		}
		authors[id].Commits++
		authors[id].LifetimeAdditions += commit.Additions
		if commit.Timestamp < authors[id].FirstCommit {
			authors[id].FirstCommit = commit.Timestamp
		}
		if commit.Timestamp > authors[id].LastCommit {
			authors[id].LastCommit = commit.Timestamp
		}
		global.TotalCommits++
		authorTimestamps[id] = append(authorTimestamps[id], commit.Timestamp)
		allTimestamps = append(allTimestamps, commit.Timestamp)
	}

	// Merge blame results
	for result := range resultsChan {
		if len(result.AuthorLines) > 0 {
			global.TotalFiles++
		}
		for email, lines := range result.AuthorLines {
			id := email
			if _, ok := authors[id]; !ok {
				// Fallback if we have lines but no commits
				authors[id] = &models.AuthorMetrics{
					Name:  email,
					Email: email,
				}
			}
			authors[id].Loc += lines
			authors[id].Files++
			global.TotalLoc += lines
		}
	}

	for id, timestamps := range authorTimestamps {
		hours := metrics.CalculateHours(timestamps, 60)
		months := metrics.CalculateMonths(timestamps)
		authors[id].Hours = hours
		authors[id].Months = months
		global.TotalHours += hours
	}

	global.TotalMonths = metrics.CalculateMonths(allTimestamps)

	return models.Result{
		Authors: authors,
		Global:  global,
	}
}
