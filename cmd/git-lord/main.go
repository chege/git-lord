package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chege/git-lord/internal/format"
	"github.com/chege/git-lord/internal/gitcmd"
	"github.com/chege/git-lord/internal/models"
	"github.com/chege/git-lord/internal/processor"
)

const version = "v0.1.0"

func main() {
	cfg := models.Config{}

	// Root flag set (default/leaderboard)
	rootFs := flag.NewFlagSet("git-lord", flag.ExitOnError)
	setupGlobalFlags(rootFs, &cfg)
	rootFs.BoolVar(&cfg.ShowAll, "all", false, "Show all columns")
	rootFs.BoolVar(&cfg.ShowSilos, "silos", false, "Show exclusivity and silo columns")
	rootFs.BoolVar(&cfg.ShowSocial, "social", false, "Show badges and behavioral columns")

	// Subcommands
	pulseFs := flag.NewFlagSet("pulse", flag.ExitOnError)
	setupGlobalFlags(pulseFs, &cfg)

	awardFs := flag.NewFlagSet("awards", flag.ExitOnError)
	setupGlobalFlags(awardFs, &cfg)

	legacyFs := flag.NewFlagSet("legacy", flag.ExitOnError)
	setupGlobalFlags(legacyFs, &cfg)

	siloFs := flag.NewFlagSet("silos", flag.ExitOnError)
	setupGlobalFlags(siloFs, &cfg)
	siloFs.IntVar(&cfg.MinLOC, "min-loc", 50, "Minimum LOC for a file to be considered a silo")

	trendFs := flag.NewFlagSet("trends", flag.ExitOnError)
	setupGlobalFlags(trendFs, &cfg)

	rootFs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: git-lord [command] [options]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  (default)  Compute all-time ownership leaderboard\n")
		fmt.Fprintf(os.Stderr, "  pulse      Compute recent activity metrics (velocity/churn)\n")
		fmt.Fprintf(os.Stderr, "  awards     Hold an award ceremony for the contributors\n")
		fmt.Fprintf(os.Stderr, "  legacy     Show breakdown of code age by year\n")
		fmt.Fprintf(os.Stderr, "  silos      Show high-risk files with low bus factor\n")
		fmt.Fprintf(os.Stderr, "  trends     Show repository growth trends by month\n")
		fmt.Fprintf(os.Stderr, "  help       Show this help screen\n\n")
		fmt.Fprintf(os.Stderr, "Leaderboard Options:\n")
		rootFs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nTip: Use 'git lord <command> --help' for subcommand specific info.\n")
	}

	pulseFs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: git-lord pulse [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		pulseFs.PrintDefaults()
	}

	awardFs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: git-lord awards [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		awardFs.PrintDefaults()
	}

	ctx := context.Background()

	if len(os.Args) < 2 {
		_ = rootFs.Parse(os.Args[1:])
		if cfg.Version {
			fmt.Printf("git-lord %s\n", version)
			return
		}
		handleError(runLeaderboard(ctx, cfg))
		return
	}

	cmd := os.Args[1]
	switch cmd {
	case "pulse":
		_ = pulseFs.Parse(os.Args[2:])
		if cfg.Version {
			fmt.Printf("git-lord %s\n", version)
			return
		}
		if cfg.Since == "" {
			cfg.Since = fmt.Sprintf("%d days ago", cfg.Days)
		}
		handleError(runPulse(ctx, cfg))
	case "awards":
		_ = awardFs.Parse(os.Args[2:])
		if cfg.Version {
			fmt.Printf("git-lord %s\n", version)
			return
		}
		handleError(runAwards(ctx, cfg))
	case "legacy":
		_ = legacyFs.Parse(os.Args[2:])
		if cfg.Version {
			fmt.Printf("git-lord %s\n", version)
			return
		}
		handleError(runLegacy(ctx, cfg))
	case "silos":
		_ = siloFs.Parse(os.Args[2:])
		if cfg.Version {
			fmt.Printf("git-lord %s\n", version)
			return
		}
		handleError(runSilos(ctx, cfg))
	case "trends":
		_ = trendFs.Parse(os.Args[2:])
		if cfg.Version {
			fmt.Printf("git-lord %s\n", version)
			return
		}
		handleError(runTrends(ctx, cfg))
	case "help":
		rootFs.Usage()
	case "-h", "--help":
		rootFs.Usage()
	case "-v", "--version":
		fmt.Printf("git-lord %s\n", version)
	default:
		if cmd[0] == '-' {
			_ = rootFs.Parse(os.Args[1:])
			if cfg.Version {
				fmt.Printf("git-lord %s\n", version)
				return
			}
			handleError(runLeaderboard(ctx, cfg))
		} else {
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
			rootFs.Usage()
			os.Exit(1)
		}
	}
}

func setupGlobalFlags(fs *flag.FlagSet, cfg *models.Config) {
	fs.StringVar(&cfg.Sort, "sort", "loc", "Sort metric: loc, coms, fils, hrs")
	fs.StringVar(&cfg.Since, "since", "", "Filter by date")
	fs.StringVar(&cfg.Include, "include", "", "Include glob pattern")
	fs.StringVar(&cfg.Exclude, "exclude", "", "Exclude glob pattern")
	fs.StringVar(&cfg.Format, "format", "table", "Output format: table, json, csv")
	fs.BoolVar(&cfg.NoHours, "no-hours", false, "Disable hours calculation")
	fs.BoolVar(&cfg.NoProgress, "no-progress", false, "Disable progress bar")
	fs.IntVar(&cfg.Days, "days", 30, "Window in days for the 'pulse' command")
	fs.BoolVar(&cfg.Version, "version", false, "Show version information")
}

func handleError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func gatherData(ctx context.Context, cfg models.Config, needFiles bool) ([]string, []gitcmd.CommitData, error) {
	if err := gitcmd.IsValidRepo(ctx); err != nil {
		return nil, nil, err
	}

	var filteredFiles []string
	if needFiles {
		files, err := gitcmd.ListTrackedFiles(ctx)
		if err != nil {
			return nil, nil, err
		}

		filteredFiles, err = filterFiles(files, cfg.Include, cfg.Exclude)
		if err != nil {
			return nil, nil, err
		}
	}

	commits, err := gitcmd.GetCommitHistory(ctx, cfg.Since)
	if err != nil {
		return nil, nil, err
	}

	return filteredFiles, commits, nil
}

func runLeaderboard(ctx context.Context, cfg models.Config) error {
	filteredFiles, commits, err := gatherData(ctx, cfg, true)
	if err != nil {
		return err
	}

	showProgress := !cfg.NoProgress
	if cfg.Format == "json" || cfg.Format == "csv" {
		showProgress = false
	}

	if cfg.Format == "table" {
		format.PrintReportHeader("Ownership Leaderboard", cfg.Since, len(filteredFiles), len(commits))
	}

	result := processor.ProcessRepository(ctx, filteredFiles, commits, showProgress, 0)
	stats := format.GenerateStats(result.Result, cfg)

	switch cfg.Format {
	case "json":
		return format.PrintJSON(stats, result.Global)
	case "csv":
		return format.PrintCSV(stats, cfg)
	default:
		format.PrintTable(stats, result.Global, cfg)
	}
	return nil
}

func runPulse(ctx context.Context, cfg models.Config) error {
	_, commits, err := gatherData(ctx, cfg, false)
	if err != nil {
		return err
	}

	showProgress := !cfg.NoProgress
	if cfg.Format == "json" || cfg.Format == "csv" {
		showProgress = false
	}

	if cfg.Format == "table" {
		format.PrintReportHeader("Activity Pulse", cfg.Since, 0, len(commits))
	}

	stats := processor.ProcessPulse(commits, showProgress)

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

func runLegacy(ctx context.Context, cfg models.Config) error {
	filteredFiles, commits, err := gatherData(ctx, cfg, true)
	if err != nil {
		return err
	}

	if cfg.Format == "table" {
		format.PrintReportHeader("Legacy Report", cfg.Since, len(filteredFiles), len(commits))
	}

	result := processor.ProcessRepository(ctx, filteredFiles, commits, !cfg.NoProgress, 0)
	stats := processor.ProcessLegacy(result.Result)
	format.PrintLegacy(stats)
	return nil
}

func runSilos(ctx context.Context, cfg models.Config) error {
	filteredFiles, commits, err := gatherData(ctx, cfg, true)
	if err != nil {
		return err
	}

	if cfg.Format == "table" {
		format.PrintReportHeader("Knowledge Silos", cfg.Since, len(filteredFiles), len(commits))
	}

	result := processor.ProcessRepository(ctx, filteredFiles, commits, !cfg.NoProgress, 0)
	silos := processor.ProcessSilos(result, cfg.MinLOC)
	format.PrintSilos(silos)
	return nil
}

func runTrends(ctx context.Context, cfg models.Config) error {
	_, commits, err := gatherData(ctx, cfg, false)
	if err != nil {
		return err
	}

	if cfg.Format == "table" {
		format.PrintReportHeader("Repository Trends", cfg.Since, 0, len(commits))
	}

	trends := processor.ProcessTrends(commits)
	format.PrintTrends(trends)
	return nil
}

func runAwards(ctx context.Context, cfg models.Config) error {
	filteredFiles, commits, err := gatherData(ctx, cfg, true)
	if err != nil {
		return err
	}

	showProgress := !cfg.NoProgress
	if cfg.Format == "json" || cfg.Format == "csv" {
		showProgress = false
	}

	if cfg.Format == "table" {
		format.PrintReportHeader("Awards Ceremony", cfg.Since, len(filteredFiles), len(commits))
	}

	result := processor.ProcessRepository(ctx, filteredFiles, commits, showProgress, 0)
	awards := processor.ProcessAwards(result.Result)

	switch cfg.Format {
	case "json":
		return format.PrintAwardsJSON(awards)
	case "csv":
		return format.PrintAwardsCSV(awards)
	default:
		format.PrintAwards(awards)
	}
	return nil
}

func filterFiles(files []string, include, exclude string) ([]string, error) {
	// Common large generated files/extensions to skip by default
	skipExts := map[string]bool{
		".lock":      true,
		".json-lock": true,
		".map":       true,
	}
	skipFiles := map[string]bool{
		"package-lock.json": true,
		"pnpm-lock.yaml":    true,
		"yarn.lock":         true,
		"go.sum":            true,
		"Cargo.lock":        true,
		"composer.lock":     true,
	}

	var filtered []string
	for _, f := range files {
		base := filepath.Base(f)
		ext := filepath.Ext(f)

		if skipFiles[base] || skipExts[ext] {
			continue
		}

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
