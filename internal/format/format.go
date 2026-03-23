package format

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/chege/git-lord/internal/models"
)

// PrintReportHeader displays a styled summary of the analysis context.
func PrintReportHeader(name string, since string, files int, commits int) {
	window := "All-time"
	if since != "" {
		window = since
	}

	fmt.Printf("%s\n", text.Colors{text.FgHiCyan, text.Bold}.Sprint("👑 GIT-LORD | "+strings.ToUpper(name)))
	fmt.Printf("%s %s\n", text.Colors{text.FgHiBlack}.Sprint("📅 Window:  "), window)

	counts := fmt.Sprintf("%d commits", commits)
	if files > 0 {
		counts += fmt.Sprintf(" across %d files", files)
	}
	fmt.Printf("%s %s\n\n", text.Colors{text.FgHiBlack}.Sprint("📦 Analyzed:"), counts)
}

type column struct {
	Header    string
	ValueFunc func(p models.AuthorStat) string
	Footer    string
}

type pulseColumn struct {
	Header    string
	ValueFunc func(p models.PulseStat) string
	Footer    func(total models.PulseStat) string
}

func formatPulseNet(net int) string {
	color := text.FgGreen
	sign := "+"
	if net < 0 {
		color = text.FgRed
		sign = ""
	}
	return color.Sprint(fmt.Sprintf("%s%d", sign, net))
}

func getPulseColumns() []pulseColumn {
	return []pulseColumn{
		{Header: "Author", ValueFunc: func(p models.PulseStat) string { return p.Name }, Footer: func(total models.PulseStat) string { return "Total" }},
		{Header: "Commits", ValueFunc: func(p models.PulseStat) string { return fmt.Sprintf("%d", p.Commits) }, Footer: func(total models.PulseStat) string { return fmt.Sprintf("%d", total.Commits) }},
		{Header: "Additions", ValueFunc: func(p models.PulseStat) string { return fmt.Sprintf("+%d", p.Additions) }, Footer: func(total models.PulseStat) string { return fmt.Sprintf("+%d", total.Additions) }},
		{Header: "Deletions", ValueFunc: func(p models.PulseStat) string { return fmt.Sprintf("-%d", p.Deletions) }, Footer: func(total models.PulseStat) string { return fmt.Sprintf("-%d", total.Deletions) }},
		{Header: "Net", ValueFunc: func(p models.PulseStat) string { return formatPulseNet(p.Net) }, Footer: func(total models.PulseStat) string { return formatPulseNet(total.Net) }},
		{Header: "Churn", ValueFunc: func(p models.PulseStat) string { return fmt.Sprintf("%d", p.Churn) }, Footer: func(total models.PulseStat) string { return fmt.Sprintf("%d", total.Churn) }},
		{Header: "Files", ValueFunc: func(p models.PulseStat) string { return fmt.Sprintf("%d", p.Files) }, Footer: func(total models.PulseStat) string { return fmt.Sprintf("%d", total.Files) }},
	}
}

func summarizePulse(stats []models.PulseStat) models.PulseStat {
	var total models.PulseStat
	for _, stat := range stats {
		total.Commits += stat.Commits
		total.Additions += stat.Additions
		total.Deletions += stat.Deletions
		total.Net += stat.Net
		total.Churn += stat.Churn
		total.Files += stat.Files
	}
	return total
}

func getColumns(global models.GlobalMetrics, cfg models.Config) []column {
	cols := []column{
		{Header: "Author", ValueFunc: func(p models.AuthorStat) string { return p.Name }, Footer: "Total"},
		{Header: "LOC", ValueFunc: func(p models.AuthorStat) string { return fmt.Sprintf("%d", p.Loc) }, Footer: fmt.Sprintf("%d", global.TotalLoc)},
	}

	if cfg.ShowAll || cfg.ShowSilos {
		cols = append(cols,
			column{Header: "Retention", ValueFunc: func(p models.AuthorStat) string { return fmt.Sprintf("%.1f%%", p.Retention) }},
		)
	}

	cols = append(cols,
		column{Header: "Commits", ValueFunc: func(p models.AuthorStat) string { return fmt.Sprintf("%d", p.Commits) }, Footer: fmt.Sprintf("%d", global.TotalCommits)},
		column{Header: "Files", ValueFunc: func(p models.AuthorStat) string { return fmt.Sprintf("%d", p.Files) }, Footer: fmt.Sprintf("%d", global.TotalFiles)},
	)

	if cfg.ShowAll || cfg.ShowSilos {
		cols = append(cols,
			column{Header: "Exclusive", ValueFunc: func(p models.AuthorStat) string { return fmt.Sprintf("%d", p.ExclusiveFiles) }, Footer: fmt.Sprintf("Bus Factor: %d", global.BusFactor)},
		)
	}

	if !cfg.NoHours && (cfg.ShowAll || cfg.ShowSocial) {
		cols = append(cols,
			column{Header: "Hours", ValueFunc: func(p models.AuthorStat) string { return fmt.Sprintf("%d", p.Hours) }, Footer: fmt.Sprintf("%d", global.TotalHours)},
			column{Header: "Months", ValueFunc: func(p models.AuthorStat) string { return fmt.Sprintf("%d", p.Months) }, Footer: fmt.Sprintf("%d", global.TotalMonths)},
			column{Header: "Max Gap", ValueFunc: func(p models.AuthorStat) string { return fmt.Sprintf("%dd", p.MaxGap) }},
		)
	}

	if cfg.ShowAll || cfg.ShowSocial {
		cols = append(cols,
			column{Header: "First", ValueFunc: func(p models.AuthorStat) string { return time.Unix(p.FirstCommit, 0).Format("2006-01-02") }},
			column{Header: "Last", ValueFunc: func(p models.AuthorStat) string { return time.Unix(p.LastCommit, 0).Format("2006-01-02") }},
			column{Header: "Badges", ValueFunc: func(p models.AuthorStat) string { return strings.Join(p.Badges, " ") }},
		)
	}

	cols = append(cols,
		column{Header: "Distribution", ValueFunc: func(p models.AuthorStat) string {
			return fmt.Sprintf("%.1f / %.1f / %.1f", p.LocDist, p.ComsDist, p.FilesDist)
		}, Footer: "100.0 / 100.0 / 100.0"},
	)

	return cols
}

func newTableWriter(headerColor text.Color) table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.Style().Options.SeparateRows = false
	t.Style().Format.Header = text.FormatUpper
	t.Style().Color.Header = text.Colors{headerColor, text.Bold}
	return t
}

// GenerateStats converts models.Result into a sorted slice of AuthorStat.
func GenerateStats(result models.Result, cfg models.Config) []models.AuthorStat {
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

		// Award "Code Janitor" badge 🧹
		if stat.LifetimeDeletions > models.JanitorDeletionThreshold && stat.LifetimeDeletions > stat.LifetimeAdditions {
			stat.Badges = append(stat.Badges, "🧹")
		}

		stats = append(stats, stat)
	}

	sort.SliceStable(stats, func(i, j int) bool {
		switch cfg.Sort {
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
func PrintTable(stats []models.AuthorStat, global models.GlobalMetrics, cfg models.Config) {
	t := newTableWriter(text.FgCyan)
	t.Style().Color.IndexColumn = text.Colors{text.FgHiCyan}

	cols := getColumns(global, cfg)

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
	t := newTableWriter(text.FgCyan)

	cols := getPulseColumns()
	total := summarizePulse(stats)
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
		footerRow[i] = c.Footer(total)
	}
	t.AppendFooter(footerRow)

	t.Render()
}

// PrintSilos formats the knowledge silo report.
func PrintSilos(silos []models.SiloRecord) {
	if len(silos) == 0 {
		fmt.Println("\n✅ No high-risk silos found. Knowledge is well distributed!")
		return
	}

	t := newTableWriter(text.FgRed)

	t.AppendHeader(table.Row{"Risk Level", "File Path", "LOC", "Primary Owner", "Ownership"})

	for _, s := range silos {
		risk := "HIGH"
		if s.Ownership >= 95.0 {
			risk = "CRITICAL"
		}
		t.AppendRow(table.Row{
			risk,
			s.Path,
			s.LOC,
			s.Owner,
			fmt.Sprintf("%.1f%%", s.Ownership),
		})
	}

	t.Render()
}

// PrintTrends formats the repository growth trends.
func PrintTrends(trends []models.TrendStat) {
	t := newTableWriter(text.FgCyan)

	t.AppendHeader(table.Row{"Month", "Additions", "Deletions", "Net Change"})

	for _, tr := range trends {
		netColor := text.FgGreen
		if tr.Net < 0 {
			netColor = text.FgRed
		}
		t.AppendRow(table.Row{
			tr.Period,
			fmt.Sprintf("+%d", tr.Additions),
			fmt.Sprintf("-%d", tr.Deletions),
			netColor.Sprint(fmt.Sprintf("%+d", tr.Net)),
		})
	}

	t.Render()
}

// PrintLegacy formats the code age report.
func PrintLegacy(stats []models.LegacyStat) {
	t := newTableWriter(text.FgCyan)

	t.AppendHeader(table.Row{"Year", "LOC", "Share (%)"})

	for _, s := range stats {
		t.AppendRow(table.Row{
			s.Year,
			s.Loc,
			fmt.Sprintf("%.1f%%", s.Pct),
		})
	}

	t.Render()
}

// PrintAwards formats the award ceremony.
func PrintAwards(awards []models.Award) {
	fmt.Printf("\n🏆 THE AWARDS CEREMONY 🏆\n\n")

	for _, a := range awards {
		header := fmt.Sprintf("%s %s", a.Emoji, strings.ToUpper(a.Title))
		fmt.Println(text.Colors{text.FgHiYellow, text.Bold}.Sprint(header))
		fmt.Printf("   Winner:  %s\n", text.Colors{text.FgCyan}.Sprint(a.Winner))
		fmt.Printf("   Meaning: %s\n", text.Colors{text.Faint}.Sprint(a.Description))
		fmt.Printf("   Vibe:    %s\n", text.Colors{text.Faint, text.Italic}.Sprint(a.Vibe))
		fmt.Printf("   Stat:    %s\n\n", text.Colors{text.FgHiGreen}.Sprint(a.Value))
	}
}

// PrintAwardsJSON formats awards to JSON.
func PrintAwardsJSON(awards []models.Award) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(awards)
}

// PrintAwardsCSV formats awards to CSV.
func PrintAwardsCSV(awards []models.Award) error {
	w := csv.NewWriter(os.Stdout)
	header := []string{"Award", "Winner", "Meaning", "Vibe", "Stat"}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, a := range awards {
		row := []string{a.Title, a.Winner, a.Description, a.Vibe, a.Value}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
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
	header := []string{"Author", "commits", "additions", "deletions", "net", "churn", "files"}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, p := range stats {
		row := []string{
			p.Name,
			fmt.Sprintf("%d", p.Commits),
			fmt.Sprintf("+%d", p.Additions),
			fmt.Sprintf("-%d", p.Deletions),
			fmt.Sprintf("%d", p.Net),
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
func PrintCSV(stats []models.AuthorStat, cfg models.Config) error {
	w := csv.NewWriter(os.Stdout)

	header := []string{"Author", "loc"}
	if cfg.ShowAll || cfg.ShowSilos {
		header = append(header, "retention")
	}
	header = append(header, "coms", "fils")
	if cfg.ShowAll || cfg.ShowSilos {
		header = append(header, "exclusive")
	}
	if !cfg.NoHours && (cfg.ShowAll || cfg.ShowSocial) {
		header = append(header, "hours", "months", "max_gap")
	}
	if cfg.ShowAll || cfg.ShowSocial {
		header = append(header, "first", "last", "badges")
	}
	header = append(header, "loc_dist", "coms_dist", "fils_dist")

	if err := w.Write(header); err != nil {
		return err
	}

	for _, p := range stats {
		row := []string{p.Name, fmt.Sprintf("%d", p.Loc)}
		if cfg.ShowAll || cfg.ShowSilos {
			row = append(row, fmt.Sprintf("%.1f", p.Retention))
		}
		row = append(row, fmt.Sprintf("%d", p.Commits), fmt.Sprintf("%d", p.Files))
		if cfg.ShowAll || cfg.ShowSilos {
			row = append(row, fmt.Sprintf("%d", p.ExclusiveFiles))
		}
		if !cfg.NoHours && (cfg.ShowAll || cfg.ShowSocial) {
			row = append(row, fmt.Sprintf("%d", p.Hours), fmt.Sprintf("%d", p.Months), fmt.Sprintf("%d", p.MaxGap))
		}
		if cfg.ShowAll || cfg.ShowSocial {
			row = append(row,
				time.Unix(p.FirstCommit, 0).Format("2006-01-02"),
				time.Unix(p.LastCommit, 0).Format("2006-01-02"),
				strings.Join(p.Badges, " "),
			)
		}
		row = append(row,
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

// PrintHotspots formats the hotspot analysis report.
func PrintHotspots(report models.HotspotReport) {
	// Filter hotspots with score >= 35
	var filtered []models.HotspotRecord
	for _, h := range report.Hotspots {
		if h.Score >= 35 {
			filtered = append(filtered, h)
		}
	}

	if len(filtered) == 0 {
		fmt.Println("\n✅ No high-risk hotspots found in this window.")
		return
	}

	// Take top 15
	if len(filtered) > 15 {
		filtered = filtered[:15]
	}

	t := newTableWriter(text.FgRed)

	t.AppendHeader(table.Row{"Risk", "Path", "Score", "LOC", "Recent Churn", "Primary Owner", "Ownership%"})

	for _, h := range filtered {
		var riskColor text.Color
		switch h.Risk {
		case "CRITICAL":
			riskColor = text.FgRed
		case "HIGH":
			riskColor = text.FgHiRed
		case "MEDIUM":
			riskColor = text.FgYellow
		case "WATCH":
			riskColor = text.FgBlue
		default:
			riskColor = text.FgGreen
		}

		t.AppendRow(table.Row{
			riskColor.Sprint(h.Risk),
			h.Path,
			h.Score,
			h.LOC,
			h.RecentChurn,
			h.PrimaryOwner,
			fmt.Sprintf("%.1f%%", h.OwnershipPct),
		})
	}

	t.Render()
}

// PrintHotspotsJSON formats hotspot report to JSON.
func PrintHotspotsJSON(report models.HotspotReport) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// PrintHotspotsCSV formats hotspot report to CSV.
func PrintHotspotsCSV(report models.HotspotReport) error {
	w := csv.NewWriter(os.Stdout)
	header := []string{"path", "score", "risk", "loc", "recent_churn", "recent_commits", "primary_owner", "owner_lines", "ownership_pct", "active_owners", "churn_score", "ownership_score", "size_score"}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, h := range report.Hotspots {
		row := []string{
			h.Path,
			fmt.Sprintf("%d", h.Score),
			h.Risk,
			fmt.Sprintf("%d", h.LOC),
			fmt.Sprintf("%d", h.RecentChurn),
			fmt.Sprintf("%d", h.RecentCommits),
			h.PrimaryOwner,
			fmt.Sprintf("%d", h.OwnerLines),
			fmt.Sprintf("%.2f", h.OwnershipPct),
			fmt.Sprintf("%d", h.ActiveOwners),
			fmt.Sprintf("%d", h.ChurnScore),
			fmt.Sprintf("%d", h.OwnershipScore),
			fmt.Sprintf("%d", h.SizeScore),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}
