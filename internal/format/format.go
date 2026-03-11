package format

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/christopher/git-lord/internal/models"
)

type column struct {
	Header    string
	ValueFunc func(p models.AuthorStat) string
	Footer    string
}

type pulseColumn struct {
	Header    string
	ValueFunc func(p models.PulseStat) string
}

func getPulseColumns() []pulseColumn {
	return []pulseColumn{
		{Header: "Author", ValueFunc: func(p models.PulseStat) string { return p.Name }},
		{Header: "Commits", ValueFunc: func(p models.PulseStat) string { return fmt.Sprintf("%d", p.Commits) }},
		{Header: "Additions", ValueFunc: func(p models.PulseStat) string { return fmt.Sprintf("+%d", p.Additions) }},
		{Header: "Deletions", ValueFunc: func(p models.PulseStat) string { return fmt.Sprintf("-%d", p.Deletions) }},
		{Header: "Churn", ValueFunc: func(p models.PulseStat) string { return fmt.Sprintf("%d", p.Churn) }},
		{Header: "Files", ValueFunc: func(p models.PulseStat) string { return fmt.Sprintf("%d", p.Files) }},
	}
}

func getColumns(global models.GlobalMetrics, noHours bool) []column {
	cols := []column{
		{Header: "Author", ValueFunc: func(p models.AuthorStat) string { return p.Name }, Footer: "Total"},
		{Header: "LOC", ValueFunc: func(p models.AuthorStat) string { return fmt.Sprintf("%d", p.Loc) }, Footer: fmt.Sprintf("%d", global.TotalLoc)},
		{Header: "Retention", ValueFunc: func(p models.AuthorStat) string { return fmt.Sprintf("%.1f%%", p.Retention) }},
		{Header: "Commits", ValueFunc: func(p models.AuthorStat) string { return fmt.Sprintf("%d", p.Commits) }, Footer: fmt.Sprintf("%d", global.TotalCommits)},
		{Header: "Files", ValueFunc: func(p models.AuthorStat) string { return fmt.Sprintf("%d", p.Files) }, Footer: fmt.Sprintf("%d", global.TotalFiles)},
		{Header: "Exclusive", ValueFunc: func(p models.AuthorStat) string { return fmt.Sprintf("%d", p.ExclusiveFiles) }, Footer: fmt.Sprintf("Bus Factor: %d", global.BusFactor)},
	}

	if !noHours {
		cols = append(cols,
			column{Header: "Hours", ValueFunc: func(p models.AuthorStat) string { return fmt.Sprintf("%d", p.Hours) }, Footer: fmt.Sprintf("%d", global.TotalHours)},
			column{Header: "Months", ValueFunc: func(p models.AuthorStat) string { return fmt.Sprintf("%d", p.Months) }, Footer: fmt.Sprintf("%d", global.TotalMonths)},
		)
	}

	cols = append(cols,
		column{Header: "First", ValueFunc: func(p models.AuthorStat) string { return time.Unix(p.FirstCommit, 0).Format("2006-01-02") }},
		column{Header: "Last", ValueFunc: func(p models.AuthorStat) string { return time.Unix(p.LastCommit, 0).Format("2006-01-02") }},
		column{Header: "Distribution", ValueFunc: func(p models.AuthorStat) string {
			return fmt.Sprintf("%.1f / %.1f / %.1f", p.LocDist, p.ComsDist, p.FilesDist)
		}, Footer: "100.0 / 100.0 / 100.0"},
	)

	return cols
}

// GenerateStats converts models.Result into a sorted slice of AuthorStat.
func GenerateStats(result models.Result, sortBy string) []models.AuthorStat {
	var stats []models.AuthorStat

	for _, m := range result.Authors {
		stat := models.AuthorStat{AuthorMetrics: *m}
		if result.Global.TotalLoc > 0 {
			stat.LocDist = (float64(stat.Loc) / float64(result.Global.TotalLoc)) * 100.0
		}
		if result.Global.TotalCommits > 0 {
			stat.ComsDist = (float64(stat.Commits) / float64(result.Global.TotalCommits)) * 100.0
		}
		if result.Global.TotalFiles > 0 {
			stat.FilesDist = (float64(stat.Files) / float64(result.Global.TotalFiles)) * 100.0
		}
		if stat.LifetimeAdditions > 0 {
			stat.Retention = (float64(stat.Loc) / float64(stat.LifetimeAdditions)) * 100.0
			if stat.Retention > 100.0 {
				stat.Retention = 100.0
			}
		}
		stats = append(stats, stat)
	}

	sort.SliceStable(stats, func(i, j int) bool {
		switch sortBy {
		case "coms":
			return stats[i].Commits > stats[j].Commits
		case "fils":
			return stats[i].Files > stats[j].Files
		case "hrs":
			return stats[i].Hours > stats[j].Hours
		case "loc":
			fallthrough
		default:
			return stats[i].Loc > stats[j].Loc
		}
	})

	return stats
}

// PrintTable formats the stats like git-fame.
func PrintTable(stats []models.AuthorStat, global models.GlobalMetrics, noHours bool) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.Style().Options.SeparateRows = false
	t.Style().Format.Header = text.FormatUpper
	t.Style().Color.Header = text.Colors{text.FgCyan, text.Bold}
	t.Style().Color.IndexColumn = text.Colors{text.FgHiCyan}

	cols := getColumns(global, noHours)

	headerRow := make(table.Row, len(cols))
	for i, c := range cols {
		headerRow[i] = c.Header
	}
	t.AppendHeader(headerRow)

	for _, p := range stats {
		row := make(table.Row, len(cols))
		for i, c := range cols {
			row[i] = c.ValueFunc(p)
		}
		t.AppendRow(row)
	}

	t.AppendSeparator()

	footerRow := make(table.Row, len(cols))
	for i, c := range cols {
		footerRow[i] = c.Footer
	}
	t.AppendFooter(footerRow)

	t.Render()
}

// PrintPulse formats the recent activity stats.
func PrintPulse(stats []models.PulseStat) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.Style().Options.SeparateRows = false
	t.Style().Format.Header = text.FormatUpper
	t.Style().Color.Header = text.Colors{text.FgCyan, text.Bold}

	cols := getPulseColumns()
	headerRow := make(table.Row, len(cols))
	for i, c := range cols {
		headerRow[i] = c.Header
	}
	t.AppendHeader(headerRow)

	for _, p := range stats {
		row := make(table.Row, len(cols))
		for i, c := range cols {
			row[i] = c.ValueFunc(p)
		}
		t.AppendRow(row)
	}

	t.Render()
}

// PrintJSON formats the stats to JSON.
func PrintJSON(stats []models.AuthorStat, global models.GlobalMetrics) error {
	data := struct {
		Authors []models.AuthorStat  `json:"authors"`
		Global  models.GlobalMetrics `json:"global"`
	}{
		Authors: stats,
		Global:  global,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// PrintPulseJSON formats the recent activity stats to JSON.
func PrintPulseJSON(stats []models.PulseStat) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(stats)
}

// PrintPulseCSV formats the recent activity stats to CSV.
func PrintPulseCSV(stats []models.PulseStat) error {
	w := csv.NewWriter(os.Stdout)
	header := []string{"Author", "commits", "additions", "deletions", "churn", "files"}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, p := range stats {
		row := []string{
			p.Name,
			fmt.Sprintf("%d", p.Commits),
			fmt.Sprintf("+%d", p.Additions),
			fmt.Sprintf("-%d", p.Deletions),
			fmt.Sprintf("%d", p.Churn),
			fmt.Sprintf("%d", p.Files),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// PrintCSV formats the stats to CSV.
func PrintCSV(stats []models.AuthorStat, noHours bool) error {
	w := csv.NewWriter(os.Stdout)

	// CSV header
	header := []string{"Author", "loc", "retention", "coms", "fils", "exclusive"}
	if !noHours {
		header = append(header, "hours", "months")
	}
	header = append(header, "first", "last", "loc_dist", "coms_dist", "fils_dist")
	if err := w.Write(header); err != nil {
		return err
	}

	for _, p := range stats {
		row := []string{
			p.Name,
			fmt.Sprintf("%d", p.Loc),
			fmt.Sprintf("%.1f", p.Retention),
			fmt.Sprintf("%d", p.Commits),
			fmt.Sprintf("%d", p.Files),
			fmt.Sprintf("%d", p.ExclusiveFiles),
		}
		if !noHours {
			row = append(row, fmt.Sprintf("%d", p.Hours), fmt.Sprintf("%d", p.Months))
		}
		row = append(row,
			time.Unix(p.FirstCommit, 0).Format("2006-01-02"),
			time.Unix(p.LastCommit, 0).Format("2006-01-02"),
			fmt.Sprintf("%.1f", p.LocDist),
			fmt.Sprintf("%.1f", p.ComsDist),
			fmt.Sprintf("%.1f", p.FilesDist),
		)
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}
