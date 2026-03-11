package processor

import (
	"fmt"
	"math"
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

// ProcessPulse calculates activity metrics from history without using blame.
func ProcessPulse(commits []gitcmd.CommitData, showProgress bool) []models.PulseStat {
	authors := make(map[string]*models.PulseStat)

	var bar *progressbar.ProgressBar
	if showProgress {
		bar = progressbar.NewOptions(len(commits),
			progressbar.OptionSetDescription("Analyzing commits"),
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

// ProcessRepository orchestrates the gathering and aggregation of metrics.
func ProcessRepository(files []string, commits []gitcmd.CommitData, showProgress bool, workerCount int) ResultExtended {
	if workerCount <= 0 {
		workerCount = runtime.NumCPU() * 2
	}
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
				blameData, err := gitcmd.GetBlame(file)
				if err == nil {
					resultsChan <- fileBlame{Path: file, Data: blameData}
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
				FirstCommit:    commit.Timestamp,
				LastCommit:     commit.Timestamp,
				ActiveDays:     make(map[string]bool),
				FileExtensions: make(map[string]bool),
			}
		}
		m := authors[id]
		m.Commits++
		m.LifetimeAdditions += commit.Additions
		m.LifetimeDeletions += commit.Deletions
		if commit.Timestamp < m.FirstCommit {
			m.FirstCommit = commit.Timestamp
		}
		if commit.Timestamp > m.LastCommit {
			m.LastCommit = commit.Timestamp
		}

		m.MessageWords += len(strings.Fields(commit.Message))
		day := time.Unix(commit.Timestamp, 0).Format("2006-01-02")
		m.ActiveDays[day] = true

		t := time.Unix(commit.Timestamp, 0)
		if t.Weekday() == time.Friday && t.Hour() >= 16 {
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
		authorTimestamps[id] = append(authorTimestamps[id], commit.Timestamp)
		allTimestamps = append(allTimestamps, commit.Timestamp)
	}

	// Merge blame results
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

// ProcessSilos finds high-risk knowledge silos.
func ProcessSilos(res ResultExtended, minLOC int) []models.SiloRecord {
	var silos []models.SiloRecord

	for path, owners := range res.FileOwners {
		total := 0
		var maxOwner string
		maxLoc := 0
		for email, loc := range owners {
			total += loc
			if loc > maxLoc {
				maxLoc = loc
				maxOwner = email
			}
		}

		if total < minLOC { 
			continue
		}

		ownership := (float64(maxLoc) / float64(total)) * 100.0
		if ownership >= 80.0 { 
			silos = append(silos, models.SiloRecord{
				Path:      path,
				LOC:       total,
				Owner:     maxOwner,
				Ownership: ownership,
			})
		}
	}

	sort.Slice(silos, func(i, j int) bool {
		return silos[i].LOC > silos[j].LOC 
	})

	if len(silos) > 15 {
		silos = silos[:15]
	}

	return silos
}

// ProcessTrends groups additions and deletions by month.
func ProcessTrends(commits []gitcmd.CommitData) []models.TrendStat {
	monthMap := make(map[string]*models.TrendStat)

	for _, c := range commits {
		period := time.Unix(c.Timestamp, 0).Format("2006-01")
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

func ProcessLegacy(res models.Result) []models.LegacyStat {
	yearMap := make(map[int]int)
	total := 0

	for _, a := range res.Authors {
		if a.OldestLineTs == 0 {
			continue
		}
		year := time.Unix(a.OldestLineTs, 0).Year()
		yearMap[year] += a.Loc
		total += a.Loc
	}

	var stats []models.LegacyStat
	for year, loc := range yearMap {
		pct := 0.0
		if total > 0 {
			pct = float64(loc) / float64(total) * 100
		}
		stats = append(stats, models.LegacyStat{
			Year: year,
			Loc:  loc,
			Pct:  pct,
		})
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Year < stats[j].Year
	})

	return stats
}

func calculateBusFactor(authors map[string]*models.AuthorMetrics, totalLoc int) int {
	if totalLoc == 0 {
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

func ProcessAwards(res models.Result) []models.Award {
	var awards []models.Award
	authors := res.Authors

	// 1. THE JANITOR
	var janitor *models.AuthorMetrics
	minNet := 0
	for _, a := range authors {
		net := a.LifetimeAdditions - a.LifetimeDeletions
		if a.LifetimeDeletions > 50 && net < minNet {
			minNet = net
			janitor = a
		}
	}
	if janitor != nil {
		awards = append(awards, models.Award{
			ID:          "janitor",
			Title:       "The Janitor",
			Emoji:       "🧹",
			Winner:      janitor.Name,
			Vibe:        "The refactoring hero cleaning up the kingdom.",
			Description: "Awarded to the person who has removed the most technical debt (Deletions > Additions).",
			Value:       fmt.Sprintf("%d Net Lines", minNet),
		})
	}

	// 2. THE INDIANA JONES
	var jones *models.AuthorMetrics
	oldest := int64(math.MaxInt64)
	for _, a := range authors {
		if a.OldestLineTs > 0 && a.OldestLineTs < oldest {
			oldest = a.OldestLineTs
			jones = a
		}
	}
	if jones != nil {
		awards = append(awards, models.Award{
			ID:          "indiana",
			Title:       "The Indiana Jones",
			Emoji:       "🤠",
			Winner:      jones.Name,
			Vibe:        "Brave enough to explore the ancient ruins.",
			Description: "Awarded to the author of the single oldest surviving line of code in the HEAD.",
			Value:       fmt.Sprintf("Code from %s", time.Unix(oldest, 0).Format("2006-01-02")),
		})
	}

	// 3. THE NOVELIST
	var novelist *models.AuthorMetrics
	maxWords := 0
	for _, a := range authors {
		if a.Commits >= 5 {
			avg := a.MessageWords / a.Commits
			if avg > maxWords {
				maxWords = avg
				novelist = a
			}
		}
	}
	if novelist != nil {
		awards = append(awards, models.Award{
			ID:          "novelist",
			Title:       "The Novelist",
			Emoji:       "📚",
			Winner:      novelist.Name,
			Vibe:        "Their git logs are available in hardcover.",
			Description: "Awarded for the highest average word count in commit messages.",
			Value:       fmt.Sprintf("%d words/commit", maxWords),
		})
	}

	// 4. THE SPEED DEMON
	var speedDemon *models.AuthorMetrics
	minInterval := float64(math.MaxInt64)
	for _, a := range authors {
		if len(a.CommitIntervals) >= 5 {
			var sum int64
			for _, v := range a.CommitIntervals {
				sum += v
			}
			avg := float64(sum) / float64(len(a.CommitIntervals))
			if avg < minInterval {
				minInterval = avg
				speedDemon = a
			}
		}
	}
	if speedDemon != nil {
		awards = append(awards, models.Award{
			ID:          "speed",
			Title:       "The Speed Demon",
			Emoji:       "🏎️",
			Winner:      speedDemon.Name,
			Vibe:        "Currently in a state of hyper-focus flow.",
			Description: "Awarded for the shortest average time between consecutive commits.",
			Value:       fmt.Sprintf("%.1f min avg", minInterval/60.0),
		})
	}

	// 5. THE POLISHED GEM
	var gem *models.AuthorMetrics
	maxRatio := 0.0
	for _, a := range authors {
		if a.Loc > 100 { // Only for significant code owners
			ratio := float64(a.Commits) / float64(a.Loc)
			if ratio > maxRatio {
				maxRatio = ratio
				gem = a
			}
		}
	}
	if gem != nil {
		awards = append(awards, models.Award{
			ID:          "gem",
			Title:       "The Polished Gem",
			Emoji:       "💎",
			Winner:      gem.Name,
			Vibe:        "For the developer who makes tiny, perfect commits.",
			Description: "Awarded for the highest ratio of commits to surviving lines of code.",
			Value:       fmt.Sprintf("%.2f coms/loc", maxRatio),
		})
	}

	// 6. THE GHOST OF CHRISTMAS PAST
	var ghost *models.AuthorMetrics
	maxGhostLoc := 0
	ninetyDaysAgo := time.Now().AddDate(0, 0, -90).Unix()
	for _, a := range authors {
		if a.LastCommit < ninetyDaysAgo {
			if a.Loc > maxGhostLoc {
				maxGhostLoc = a.Loc
				ghost = a
			}
		}
	}
	if ghost != nil {
		awards = append(awards, models.Award{
			ID:          "ghost",
			Title:       "The Ghost of Christmas Past",
			Emoji:       "👻",
			Winner:      ghost.Name,
			Vibe:        "Their spirit still haunts every file.",
			Description: "Identifies the largest knowledge silo from a contributor who hasn't been active in 90 days.",
			Value:       fmt.Sprintf("%d surviving LOC", maxGhostLoc),
		})
	}

	// 7. THE FRIDAY ROULETTE
	var gambler *models.AuthorMetrics
	maxFriday := 0
	for _, a := range authors {
		if a.FridayAfterFour > maxFriday {
			maxFriday = a.FridayAfterFour
			gambler = a
		}
	}
	if gambler != nil {
		awards = append(awards, models.Award{
			ID:          "friday",
			Title:       "The Friday Roulette",
			Emoji:       "🎲",
			Winner:      gambler.Name,
			Vibe:        "Living life on the edge of a broken production build.",
			Description: "Awarded for the most pushes made after 4:00 PM on a Friday.",
			Value:       fmt.Sprintf("%d late Friday pushes", maxFriday),
		})
	}

	// 8. THE DEEP THINKER
	var thinker *models.AuthorMetrics
	maxGap := 0
	for _, a := range authors {
		if a.MaxGap > maxGap && a.MaxGap > 7 {
			maxGap = a.MaxGap
			thinker = a
		}
	}
	if thinker != nil {
		awards = append(awards, models.Award{
			ID:          "thinker",
			Title:       "The Deep Thinker",
			Emoji:       "🧘",
			Winner:      thinker.Name,
			Vibe:        "Currently in a state of high-level strategic meditation.",
			Description: "Awarded for the longest streak of consecutive days with zero activity.",
			Value:       fmt.Sprintf("%d day gap", maxGap),
		})
	}

	// 9. THE STEALTH SPECIALIST
	var stealth *models.AuthorMetrics
	minVelocity := 100.0
	for _, a := range authors {
		activeDays := len(a.ActiveDays)
		if activeDays > 10 { // Threshold: Must be a long-term contributor
			velocity := float64(a.LifetimeAdditions+a.LifetimeDeletions) / float64(activeDays)
			if velocity < 1.5 {
				if velocity < minVelocity {
					minVelocity = velocity
					stealth = a
				}
			}
		}
	}
	if stealth != nil {
		awards = append(awards, models.Award{
			ID:          "stealth",
			Title:       "The Stealth Specialist",
			Emoji:       "🥷",
			Winner:      stealth.Name,
			Vibe:        "Operating in the shadows with surgical precision.",
			Description: "Awarded for having many active days but making very small, targeted changes.",
			Value:       fmt.Sprintf("%.1f changes/day", minVelocity),
		})
	}

	return awards
}
