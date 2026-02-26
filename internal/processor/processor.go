package processor

import (
	"runtime"
	"sync"

	"github.com/christopher/git-lord/internal/gitcmd"
	"github.com/christopher/git-lord/internal/metrics"
)

// AuthorMetrics holds all statistics for a single author.
type AuthorMetrics struct {
	Name    string
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

	// Merge blame results
	for result := range resultsChan {
		if len(result.AuthorLines) > 0 {
			global.TotalFiles++
		}
		for author, lines := range result.AuthorLines {
			if _, ok := authors[author]; !ok {
				authors[author] = &AuthorMetrics{Name: author}
			}
			authors[author].Loc += lines
			authors[author].Files++
			global.TotalLoc += lines
		}
	}

	// Merge commits and prepare timestamps for hours/months
	authorTimestamps := make(map[string][]int64)
	var allTimestamps []int64

	for _, commit := range commits {
		if _, ok := authors[commit.Author]; !ok {
			// Some authors might have commits but 0 surviving LOC.
			authors[commit.Author] = &AuthorMetrics{Name: commit.Author}
		}
		authors[commit.Author].Commits++
		global.TotalCommits++
		authorTimestamps[commit.Author] = append(authorTimestamps[commit.Author], commit.Timestamp)
		allTimestamps = append(allTimestamps, commit.Timestamp)
	}

	for author, timestamps := range authorTimestamps {
		hours := metrics.CalculateHours(timestamps, 60)
		months := metrics.CalculateMonths(timestamps)
		authors[author].Hours = hours
		authors[author].Months = months
		global.TotalHours += hours
	}

	global.TotalMonths = metrics.CalculateMonths(allTimestamps)

	return Result{
		Authors: authors,
		Global:  global,
	}
}
