package processor

import (
	"fmt"
	"math"
	"time"

	"github.com/chege/git-lord/internal/models"
)

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
