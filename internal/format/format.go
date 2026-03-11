package format

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/christopher/git-lord/internal/processor"
)

// AuthorStat extends AuthorMetrics with calculated distribution percentages.
type AuthorStat struct {
	processor.AuthorMetrics
	LocDist   float64 `json:"loc_percent"`
	ComsDist  float64 `json:"commits_percent"`
	FilesDist float64 `json:"files_percent"`
}

// GenerateStats converts processor.Result into a sorted slice of AuthorStat.
func GenerateStats(result processor.Result, sortBy string) []AuthorStat {
	var stats []AuthorStat

	for _, m := range result.Authors {
		stat := AuthorStat{AuthorMetrics: *m}
		if result.Global.TotalLoc > 0 {
			stat.LocDist = (float64(stat.Loc) / float64(result.Global.TotalLoc)) * 100.0
		}
		if result.Global.TotalCommits > 0 {
			stat.ComsDist = (float64(stat.Commits) / float64(result.Global.TotalCommits)) * 100.0
		}
		if result.Global.TotalFiles > 0 {
			stat.FilesDist = (float64(stat.Files) / float64(result.Global.TotalFiles)) * 100.0
		}
		stats = append(stats, stat)
	}

	sort.SliceStable(stats, func(i, j int) bool {
		// Descending order sorting
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
func PrintTable(stats []AuthorStat, global processor.GlobalMetrics, noHours bool) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if noHours {
		_, _ = fmt.Fprintln(w, "Author\tloc\tcoms\tfils\tdistribution")
	} else {
		_, _ = fmt.Fprintln(w, "Author\tloc\tcoms\tfils\thours\tmonths\tdistribution")
	}

	for _, p := range stats {
		dist := fmt.Sprintf("%.1f / %.1f / %.1f", p.LocDist, p.ComsDist, p.FilesDist)
		if noHours {
			_, _ = fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%s\n",
				p.Name, p.Loc, p.Commits, p.Files, dist)
		} else {
			_, _ = fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\t%d\t%s\n",
				p.Name, p.Loc, p.Commits, p.Files, p.Hours, p.Months, dist)
		}
	}

	_, _ = fmt.Fprintln(w, "---------------------------------------------------------------------------------")
	if noHours {
		_, _ = fmt.Fprintf(w, "Total\t%d\t%d\t%d\t100.0 / 100.0 / 100.0\n",
			global.TotalLoc, global.TotalCommits, global.TotalFiles)
	} else {
		_, _ = fmt.Fprintf(w, "Total\t%d\t%d\t%d\t%d\t%d\t100.0 / 100.0 / 100.0\n",
			global.TotalLoc, global.TotalCommits, global.TotalFiles, global.TotalHours, global.TotalMonths)
	}
	_ = w.Flush()
}

// PrintJSON formats the stats to JSON.
func PrintJSON(stats []AuthorStat, global processor.GlobalMetrics) error {
	data := struct {
		Authors []AuthorStat            `json:"authors"`
		Global  processor.GlobalMetrics `json:"global"`
	}{
		Authors: stats,
		Global:  global,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// PrintCSV formats the stats to CSV.
func PrintCSV(stats []AuthorStat, noHours bool) error {
	w := csv.NewWriter(os.Stdout)
	var header []string
	if noHours {
		header = []string{"Author", "loc", "coms", "fils", "distribution"}
	} else {
		header = []string{"Author", "loc", "coms", "fils", "hours", "months", "distribution"}
	}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, p := range stats {
		dist := fmt.Sprintf("%.1f / %.1f / %.1f", p.LocDist, p.ComsDist, p.FilesDist)
		var row []string
		if noHours {
			row = []string{
				p.Name,
				fmt.Sprintf("%d", p.Loc),
				fmt.Sprintf("%d", p.Commits),
				fmt.Sprintf("%d", p.Files),
				dist,
			}
		} else {
			row = []string{
				p.Name,
				fmt.Sprintf("%d", p.Loc),
				fmt.Sprintf("%d", p.Commits),
				fmt.Sprintf("%d", p.Files),
				fmt.Sprintf("%d", p.Hours),
				fmt.Sprintf("%d", p.Months),
				dist,
			}
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}
