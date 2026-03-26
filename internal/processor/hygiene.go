package processor

import (
	"regexp"
	"sort"
	"strings"

	"github.com/chege/git-lord/internal/gitcmd"
	"github.com/chege/git-lord/internal/models"
)

var (
	conventionalPattern = regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|chore|ci|build|revert)(\(.+\))?: .+`)
	issueRefPattern     = regexp.MustCompile(`(#\d+|[A-Z]+-\d+|closes? #\d+|fixes? #\d+|refs? #\d+)`)
	vaguePatterns       = []string{"fix", "update", "wip", "tmp", "test", "changes", "work", "more", "cleanup", "refactor"}
)

func ProcessCommitHygiene(commits []gitcmd.CommitData) models.CommitHygieneReport {
	type authorStats struct {
		name            string
		email           string
		commits         int
		tooShort        int
		vague           int
		missingConv     int
		missingIssueRef int
		hasBody         int
		totalLength     int
	}

	statsByEmail := make(map[string]*authorStats)

	for _, commit := range commits {
		if commit.IsMerge {
			continue
		}

		email := commit.Email
		if _, ok := statsByEmail[email]; !ok {
			statsByEmail[email] = &authorStats{
				name:  commit.Author,
				email: email,
			}
		}

		stat := statsByEmail[email]
		stat.commits++

		msg := commit.Message
		stat.totalLength += len(msg)

		lines := strings.Split(msg, "\n")
		subject := lines[0]
		subjectLower := strings.ToLower(subject)

		if len(subject) < 15 || len(strings.Fields(subject)) < 3 {
			stat.tooShort++
		}

		isVague := false
		for _, pattern := range vaguePatterns {
			if subjectLower == pattern || strings.HasPrefix(subjectLower, pattern+" ") {
				isVague = true
				break
			}
		}
		if isVague {
			stat.vague++
		}

		if !conventionalPattern.MatchString(subject) {
			stat.missingConv++
		}

		if !issueRefPattern.MatchString(msg) {
			stat.missingIssueRef++
		}

		hasBodyContent := false
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) != "" {
				hasBodyContent = true
				break
			}
		}
		if hasBodyContent {
			stat.hasBody++
		}
	}

	var records []models.CommitHygieneRecord
	for _, stat := range statsByEmail {
		if stat.commits == 0 {
			continue
		}

		pct := func(count int) float64 {
			return float64(count) * 100.0 / float64(stat.commits)
		}

		record := models.CommitHygieneRecord{
			Author:              stat.name,
			Email:               stat.email,
			TotalCommits:        stat.commits,
			TooShort:            stat.tooShort,
			TooShortPct:         pct(stat.tooShort),
			VagueMessages:       stat.vague,
			VaguePct:            pct(stat.vague),
			MissingConventional: stat.missingConv,
			ConventionalPct:     100.0 - pct(stat.missingConv),
			MissingIssueRef:     stat.missingIssueRef,
			IssueRefPct:         100.0 - pct(stat.missingIssueRef),
			HasBody:             stat.hasBody,
			BodyPct:             pct(stat.hasBody),
			AvgMessageLength:    float64(stat.totalLength) / float64(stat.commits),
		}

		record.HygieneScore = calculateHygieneScore(record)
		records = append(records, record)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].HygieneScore > records[j].HygieneScore
	})

	return models.CommitHygieneReport{Authors: records}
}

func calculateHygieneScore(r models.CommitHygieneRecord) float64 {
	score := 100.0

	score -= r.TooShortPct * 0.25
	score -= r.VaguePct * 0.20
	score -= (100.0 - r.ConventionalPct) * 0.15
	score -= (100.0 - r.IssueRefPct) * 0.15
	score += r.BodyPct * 0.10

	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}
