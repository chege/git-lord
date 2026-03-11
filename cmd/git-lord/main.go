package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/christopher/git-lord/internal/format"
	"github.com/christopher/git-lord/internal/gitcmd"
	"github.com/christopher/git-lord/internal/processor"
)

type config struct {
	Sort       string
	Since      string
	Include    string
	Exclude    string
	Format     string
	NoHours    bool
	NoProgress bool
	Days       int
}

func main() {
	// Root flag set (default/leaderboard)
	rootFs := flag.NewFlagSet("git-lord", flag.ExitOnError)
	cfg := config{}
	setupFlags(rootFs, &cfg)

	// Pulse flag set
	pulseFs := flag.NewFlagSet("pulse", flag.ExitOnError)
	setupFlags(pulseFs, &cfg)

	rootFs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: git-lord [command] [options]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  (default)  Compute all-time ownership leaderboard\n")
		fmt.Fprintf(os.Stderr, "  pulse      Compute recent activity metrics (velocity/churn)\n")
		fmt.Fprintf(os.Stderr, "  help       Show this help screen\n\n")
		fmt.Fprintf(os.Stderr, "Global Options:\n")
		rootFs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nOutput Columns (Leaderboard):\n")
		fmt.Fprintf(os.Stderr, "  AUTHOR        Contributor's name.\n")
		fmt.Fprintf(os.Stderr, "  LOC           Current surviving lines of code attributed to the author.\n")
		fmt.Fprintf(os.Stderr, "  RETENTION     Percentage of lifetime additions that still exist.\n")
		fmt.Fprintf(os.Stderr, "  COMMITS       Total number of commits by the author.\n")
		fmt.Fprintf(os.Stderr, "  DISTRIBUTION  Percentage share of total [LOC / COMMITS / FILES].\n")
	}
	pulseFs.Usage = rootFs.Usage

	if len(os.Args) < 2 {
		_ = rootFs.Parse(os.Args[1:])
		handleError(runLeaderboard(cfg))
		return
	}

	cmd := os.Args[1]
	switch cmd {
	case "pulse":
		_ = pulseFs.Parse(os.Args[2:])
		if cfg.Since == "" {
			cfg.Since = fmt.Sprintf("%d days ago", cfg.Days)
		}
		handleError(runPulse(cfg))
	case "help", "-h", "--help":
		rootFs.Usage()
	default:
		// Default to leaderboard if not a known command
		_ = rootFs.Parse(os.Args[1:])
		handleError(runLeaderboard(cfg))
	}
}

func setupFlags(fs *flag.FlagSet, cfg *config) {
	fs.StringVar(&cfg.Sort, "sort", "loc", "Sort metric: loc, coms, fils, hrs")
	fs.StringVar(&cfg.Since, "since", "", "Filter by date (e.g., '2023-01-01', '2 weeks ago')")
	fs.StringVar(&cfg.Include, "include", "", "Include glob pattern")
	fs.StringVar(&cfg.Exclude, "exclude", "", "Exclude glob pattern")
	fs.StringVar(&cfg.Format, "format", "table", "Output format: table, json, csv")
	fs.BoolVar(&cfg.NoHours, "no-hours", false, "Disable hours calculation")
	fs.BoolVar(&cfg.NoProgress, "no-progress", false, "Disable progress bar")
	fs.IntVar(&cfg.Days, "days", 30, "Window in days for the 'pulse' command")
}

func handleError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runLeaderboard(cfg config) error {
	if err := gitcmd.IsValidRepo(); err != nil {
		return err
	}

	files, err := gitcmd.ListTrackedFiles()
	if err != nil {
		return err
	}

	filteredFiles, err := filterFiles(files, cfg.Include, cfg.Exclude)
	if err != nil {
		return err
	}

	commits, err := gitcmd.GetCommitHistory(cfg.Since)
	if err != nil {
		return err
	}

	showProgress := !cfg.NoProgress
	if cfg.Format == "json" || cfg.Format == "csv" {
		showProgress = false
	}

	result := processor.ProcessRepository(filteredFiles, commits, showProgress)
	stats := format.GenerateStats(result, cfg.Sort)

	switch cfg.Format {
	case "json":
		return format.PrintJSON(stats, result.Global)
	case "csv":
		return format.PrintCSV(stats, cfg.NoHours)
	default:
		format.PrintTable(stats, result.Global, cfg.NoHours)
	}
	return nil
}

func runPulse(cfg config) error {
	if err := gitcmd.IsValidRepo(); err != nil {
		return err
	}

	commits, err := gitcmd.GetCommitHistory(cfg.Since)
	if err != nil {
		return err
	}

	stats := processor.ProcessPulse(commits)

	switch cfg.Format {
	case "json":
		return format.PrintPulseJSON(stats)
	case "csv":
		return format.PrintPulseCSV(stats)
	default:
		format.PrintPulse(stats)
	}
	return nil
}

func filterFiles(files []string, include, exclude string) ([]string, error) {
	var filtered []string
	for _, f := range files {
		if include != "" {
			match, err := filepath.Match(include, f)
			if err != nil {
				return nil, fmt.Errorf("invalid include pattern: %w", err)
			}
			if !match {
				continue
			}
		}
		if exclude != "" {
			match, err := filepath.Match(exclude, f)
			if err != nil {
				return nil, fmt.Errorf("invalid exclude pattern: %w", err)
			}
			if match {
				continue
			}
		}
		filtered = append(filtered, f)
	}
	return filtered, nil
}
