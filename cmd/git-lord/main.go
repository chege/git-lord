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
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: git-lord [command] [options]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  (default)  Compute all-time ownership leaderboard\n")
		fmt.Fprintf(os.Stderr, "  pulse      Compute recent activity metrics (velocity/churn)\n")
		fmt.Fprintf(os.Stderr, "  help       Show this help screen\n\n")
		fmt.Fprintf(os.Stderr, "Global Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nOutput Columns (Leaderboard):\n")
		fmt.Fprintf(os.Stderr, "  AUTHOR        Contributor's name.\n")
		fmt.Fprintf(os.Stderr, "  LOC           Current surviving lines of code attributed to the author.\n")
		fmt.Fprintf(os.Stderr, "  RETENTION     Percentage of lifetime additions that still exist.\n")
		fmt.Fprintf(os.Stderr, "  COMMITS       Total number of commits by the author.\n")
		fmt.Fprintf(os.Stderr, "  DISTRIBUTION  Percentage share of total [LOC / COMMITS / FILES].\n")
	}

	cfg := config{}
	flag.StringVar(&cfg.Sort, "sort", "loc", "Sort metric: loc, coms, fils, hrs")
	flag.StringVar(&cfg.Since, "since", "", "Filter by date (e.g., '2023-01-01', '2 weeks ago')")
	flag.StringVar(&cfg.Include, "include", "", "Include glob pattern")
	flag.StringVar(&cfg.Exclude, "exclude", "", "Exclude glob pattern")
	flag.StringVar(&cfg.Format, "format", "table", "Output format: table, json, csv")
	flag.BoolVar(&cfg.NoHours, "no-hours", false, "Disable hours calculation")
	flag.BoolVar(&cfg.NoProgress, "no-progress", false, "Disable progress bar")
	flag.IntVar(&cfg.Days, "days", 30, "Window in days for the 'pulse' command")

	flag.Parse()

	args := flag.Args()
	cmd := ""
	if len(args) > 0 {
		cmd = args[0]
	}

	// Handle 'help' command or sub-command help
	if cmd == "help" {
		flag.Usage()
		return
	}

	var err error
	switch cmd {
	case "pulse":
		// Pulse defaults to cfg.Days if cfg.Since is not specified
		if cfg.Since == "" {
			cfg.Since = fmt.Sprintf("%d days ago", cfg.Days)
		}
		err = runPulse(cfg)
	default:
		err = runLeaderboard(cfg)
	}

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
	format.PrintPulse(stats)
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
