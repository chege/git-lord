package processor

import (
	"context"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"

	"github.com/chege/git-lord/internal/gitcmd"
	"github.com/chege/git-lord/internal/metrics"
	"github.com/chege/git-lord/internal/models"
)

// ResultExtended holds extra data needed for specialized reports.
type ResultExtended struct {
	models.Result
	FileOwners map[string]map[string]int // file -> email -> lines
}

func newProgressBar(count int, desc string, show bool) *progressbar.ProgressBar {
	if !show {
		return nil
	}
	return progressbar.NewOptions(count,
		progressbar.OptionSetDescription(desc),
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

// ProcessRepository orchestrates the gathering and aggregation of metrics.
func ProcessRepository(ctx context.Context, files []string, commits []gitcmd.CommitData, showProgress bool, workerCount int) ResultExtended {
	if len(commits) == 0 && len(files) == 0 {
		return ResultExtended{}
	}

	if workerCount <= 0 {
		workerCount = runtime.NumCPU() * 2
	}
	filesChan := make(chan string, len(files))
	for _, f := range files {
		filesChan <- f
	}
	close(filesChan)

	bar := newProgressBar(len(files), "Blaming files", showProgress)

	type fileBlame struct {
		Path string
		Data gitcmd.BlameData
	}

	var wg sync.WaitGroup
	resultsChan := make(chan fileBlame, workerCount*2)

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range filesChan {
				select {
				case <-ctx.Done():
					return
				default:
					blameData, err := gitcmd.GetBlame(ctx, file)
					if err == nil {
						resultsChan <- fileBlame{Path: file, Data: blameData}
					}
					if bar != nil {
						_ = bar.Add(1)
					}
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
	fileOwners := make(map[string]map[string]int)
	var global models.GlobalMetrics

	authorTimestamps := make(map[string][]int64)
	var allTimestamps []int64

	for _, commit := range commits {
		id := commit.Email
		if _, ok := authors[id]; !ok {
			authors[id] = &models.AuthorMetrics{
				Name:           commit.Author,
				Email:          commit.Email,
				FirstCommit:    commit.Date.Unix(),
				LastCommit:     commit.Date.Unix(),
				ActiveDays:     make(map[string]bool),
				FileExtensions: make(map[string]bool),
			}
		}
		m := authors[id]
		m.Commits++
		m.LifetimeAdditions += commit.Additions
		m.LifetimeDeletions += commit.Deletions
		if commit.Date.Unix() < m.FirstCommit {
			m.FirstCommit = commit.Date.Unix()
		}
		if commit.Date.Unix() > m.LastCommit {
			m.LastCommit = commit.Date.Unix()
		}

		m.MessageWords += len(strings.Fields(commit.Message))
		day := commit.Date.Format("2006-01-02")
		m.ActiveDays[day] = true

		if commit.Date.Weekday() == time.Friday && commit.Date.Hour() >= 16 {
			m.FridayAfterFour++
		}

		lowerMsg := strings.ToLower(commit.Message)
		if strings.Contains(lowerMsg, "lint") || strings.Contains(lowerMsg, "style") {
			m.LintCommits++
		}

		if commit.IsMerge {
			m.MergeCommits++
		}

		global.TotalCommits++
		authorTimestamps[id] = append(authorTimestamps[id], commit.Date.Unix())
		allTimestamps = append(allTimestamps, commit.Date.Unix())
	}

	for res := range resultsChan {
		path := res.Path
		blame := res.Data
		if len(blame.AuthorLines) > 0 {
			global.TotalFiles++
		}
		
		fileOwners[path] = make(map[string]int)

		if len(blame.AuthorLines) == 1 {
			for email := range blame.AuthorLines {
				id := email
				if _, ok := authors[id]; !ok {
					authors[id] = &models.AuthorMetrics{Name: email, Email: email, ActiveDays: make(map[string]bool), FileExtensions: make(map[string]bool)}
				}
				authors[id].ExclusiveFiles++
			}
		}

		for email, timestamps := range blame.AuthorLines {
			id := email
			if _, ok := authors[id]; !ok {
				authors[id] = &models.AuthorMetrics{
					Name:           email,
					Email:          email,
					ActiveDays:     make(map[string]bool),
					FileExtensions: make(map[string]bool),
				}
			}
			m := authors[id]
			lineCount := len(timestamps)
			m.Loc += lineCount
			m.Files++
			global.TotalLoc += lineCount
			fileOwners[path][email] = lineCount

			for _, ts := range timestamps {
				if m.OldestLineTs == 0 || ts < m.OldestLineTs {
					m.OldestLineTs = ts
				}
			}
		}
	}

	for id, timestamps := range authorTimestamps {
		sort.SliceStable(timestamps, func(i, j int) bool { return timestamps[i] < timestamps[j] })
		
		hours := metrics.CalculateHours(timestamps, 60)
		months := metrics.CalculateMonths(timestamps)
		authors[id].Hours = hours
		authors[id].Months = months
		authors[id].MaxGap = metrics.CalculateMaxGap(timestamps)
		
		for i := 1; i < len(timestamps); i++ {
			diff := timestamps[i] - timestamps[i-1]
			if diff < 8*3600 { 
				authors[id].CommitIntervals = append(authors[id].CommitIntervals, diff)
			}
		}

		global.TotalHours += hours
	}

	global.TotalMonths = metrics.CalculateMonths(allTimestamps)
	global.BusFactor = calculateBusFactor(authors, global.TotalLoc)

	return ResultExtended{
		Result: models.Result{
			Authors: authors,
			Global:  global,
		},
		FileOwners: fileOwners,
	}
}

func calculateBusFactor(authors map[string]*models.AuthorMetrics, totalLoc int) int {
	if totalLoc <= 0 || len(authors) == 0 {
		return 0
	}

	var locs []int
	for _, a := range authors {
		locs = append(locs, a.Loc)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(locs)))

	count := 0
	runningSum := 0
	threshold := totalLoc / 2

	for _, loc := range locs {
		runningSum += loc
		count++
		if runningSum > threshold {
			break
		}
	}
	return count
}
