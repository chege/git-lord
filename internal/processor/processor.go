package processor

import (
	"runtime"
	"sync"

	"github.com/christopher/git-lord/internal/gitcmd"
	"github.com/christopher/git-lord/internal/metrics"
)

type AuthorMetrics struct {
	Name    string
	Email   string
	Loc     int
	Commits int
	Files   int
	Hours   int
	Months  int
}

// GlobalMetrics holds the sum of all metrics.
type GlobalMetrics struct {
	TotalLoc     int
	TotalCommits int
	TotalFiles   int
	TotalHours   int
	TotalMonths  int
}

// Result holds the final processed data.
type Result struct {
	Authors map[string]*AuthorMetrics
	Global  GlobalMetrics
}

// ProcessRepository orchestrates the gathering and aggregation of metrics.
func ProcessRepository(files []string, commits []gitcmd.CommitData) Result {
	numCPU := runtime.NumCPU()
	filesChan := make(chan string, len(files))
	for _, f := range files {
		filesChan <- f
	}
	close(filesChan)

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
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	authors := make(map[string]*AuthorMetrics)
	var global GlobalMetrics

	// Merge commits and prepare timestamps for hours/months
	// We do this first to populate the Names for the Emails
	authorTimestamps := make(map[string][]int64)
	var allTimestamps []int64

	for _, commit := range commits {
		// Use Email as the unique identifier
		id := commit.Email
		if _, ok := authors[id]; !ok {
			authors[id] = &AuthorMetrics{
				Name:  commit.Author,
				Email: commit.Email,
			}
		}
		authors[id].Commits++
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
				authors[id] = &AuthorMetrics{
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

	return Result{
		Authors: authors,
		Global:  global,
	}
}
