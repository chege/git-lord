package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chege/git-lord/internal/cache"
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

	hotspotFs := flag.NewFlagSet("hotspot", flag.ExitOnError)
	setupGlobalFlags(hotspotFs, &cfg)
	hotspotFs.IntVar(&cfg.Window, "window", 30, "Analysis window in days")

	hygieneFs := flag.NewFlagSet("hygiene", flag.ExitOnError)
	setupGlobalFlags(hygieneFs, &cfg)

	branchFs := flag.NewFlagSet("branches", flag.ExitOnError)
	setupGlobalFlags(branchFs, &cfg)
	cfg.Days = 90
	branchFs.BoolVar(&cfg.Stale, "stale", false, "Filter to stale branches only")
	branchFs.BoolVar(&cfg.Unmerged, "unmerged", false, "Filter to unmerged branches only")
	branchFs.BoolVar(&cfg.Orphaned, "orphaned", false, "Filter to orphaned branches only")
	branchFs.BoolVar(&cfg.IncludeRemote, "include-remote", false, "Include remote branches")
	branchFs.BoolVar(&cfg.Purge, "purge", false, "Delete listed branches after displaying")
	branchFs.BoolVar(&cfg.Force, "force", false, "Skip confirmation when purging branches")

	rootFs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: git-lord [command] [options]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  (default)  Compute all-time ownership leaderboard\n")
		fmt.Fprintf(os.Stderr, "  pulse      Compute recent activity metrics (velocity/churn)\n")
		fmt.Fprintf(os.Stderr, "  awards     Hold an award ceremony for the contributors\n")
		fmt.Fprintf(os.Stderr, "  legacy     Show breakdown of code age by year\n")
		fmt.Fprintf(os.Stderr, "  silos      Show high-risk files with low bus factor\n")
		fmt.Fprintf(os.Stderr, "  trends     Show repository growth trends by month\n")
		fmt.Fprintf(os.Stderr, "  hotspot    Show high-churn concentration risk files\n")
		fmt.Fprintf(os.Stderr, "  hygiene    Analyze commit message quality and hygiene\n")
		fmt.Fprintf(os.Stderr, "  branches   Analyze branch health and status\n")
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

	hygieneFs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: git-lord hygiene [options]\n\n")
		fmt.Fprintf(os.Stderr, "Analyzes commit message quality including:\n")
		fmt.Fprintf(os.Stderr, "  - Message length and clarity\n")
		fmt.Fprintf(os.Stderr, "  - Conventional commit format\n")
		fmt.Fprintf(os.Stderr, "  - Issue reference presence\n")
		fmt.Fprintf(os.Stderr, "  - Commit body descriptions\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		hygieneFs.PrintDefaults()
	}

	branchFs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: git-lord branches [options]\n\n")
		fmt.Fprintf(os.Stderr, "Analyzes branch health and status:\n")
		fmt.Fprintf(os.Stderr, "  - Stale branches (no commits beyond threshold)\n")
		fmt.Fprintf(os.Stderr, "  - Unmerged branches (not yet merged to default)\n")
		fmt.Fprintf(os.Stderr, "  - Orphaned branches (stale AND unmerged)\n\n")
		fmt.Fprintf(os.Stderr, "Use --purge to delete listed branches (requires --force)\n")
		fmt.Fprintf(os.Stderr, "Safety: Never deletes default branch or current HEAD\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		branchFs.PrintDefaults()
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
	case "hotspot":
		_ = hotspotFs.Parse(os.Args[2:])
		if cfg.Version {
			fmt.Printf("git-lord %s\n", version)
			return
		}
		if cfg.Window <= 0 {
			cfg.Window = 30
		}
		cfg.Since = fmt.Sprintf("%d days ago", cfg.Window)
		handleError(runHotspot(ctx, cfg))
	case "hygiene":
		_ = hygieneFs.Parse(os.Args[2:])
		if cfg.Version {
			fmt.Printf("git-lord %s\n", version)
			return
		}
		handleError(runHygiene(ctx, cfg))
	case "branches":
		_ = branchFs.Parse(os.Args[2:])
		if cfg.Version {
			fmt.Printf("git-lord %s\n", version)
			return
		}
		handleError(runBranches(ctx, cfg))
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
	fs.StringVar(&cfg.Sort, "sort", "loc", "Sort metric: leaderboard loc|coms|fils|hrs; pulse commits|additions|deletions|net|churn|files")
	fs.StringVar(&cfg.Since, "since", "", "Filter by date")
	fs.StringVar(&cfg.Include, "include", "", "Include glob pattern")
	fs.StringVar(&cfg.Exclude, "exclude", "", "Exclude glob pattern")
	fs.StringVar(&cfg.Format, "format", "table", "Output format: table, json, csv, markdown")
	fs.BoolVar(&cfg.NoHours, "no-hours", false, "Disable hours calculation")
	fs.BoolVar(&cfg.NoProgress, "no-progress", false, "Disable progress bar")
	fs.BoolVar(&cfg.NoCache, "no-cache", false, "Disable caching of blame data")
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
	// Load cache
	var c *cache.Cache
	if !cfg.NoCache {
		headHash, _ := cache.GetHEAD()
		c, _ = cache.Load(headHash)
	}

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

	result := processor.ProcessRepository(ctx, filteredFiles, commits, showProgress, 0, c)

	// Save cache
	if c != nil {
		_ = c.Save()
	}

	stats := format.GenerateStats(result.Result, cfg)

	switch cfg.Format {
	case "json":
		return format.PrintJSON(stats, result.Global)
	case "csv":
		return format.PrintCSV(stats, cfg)
	case "markdown":
		return format.PrintMarkdown(stats, result.Global, cfg)
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

	stats := processor.ProcessPulse(commits, cfg.Sort, showProgress)

	switch cfg.Format {
	case "json":
		return format.PrintPulseJSON(stats)
	case "csv":
		return format.PrintPulseCSV(stats)
	case "markdown":
		return format.PrintPulseMarkdown(stats)
	default:
		format.PrintPulse(stats)
	}
	return nil
}

func runLegacy(ctx context.Context, cfg models.Config) error {
	// Load cache
	var c *cache.Cache
	if !cfg.NoCache {
		headHash, _ := cache.GetHEAD()
		c, _ = cache.Load(headHash)
	}

	filteredFiles, commits, err := gatherData(ctx, cfg, true)
	if err != nil {
		return err
	}

	if cfg.Format == "table" {
		format.PrintReportHeader("Legacy Report", cfg.Since, len(filteredFiles), len(commits))
	}

	result := processor.ProcessRepository(ctx, filteredFiles, commits, !cfg.NoProgress, 0, c)

	// Save cache
	if c != nil {
		_ = c.Save()
	}

	stats := processor.ProcessLegacy(result.Result)
	format.PrintLegacy(stats)
	return nil
}

func runSilos(ctx context.Context, cfg models.Config) error {
	// Load cache
	var c *cache.Cache
	if !cfg.NoCache {
		headHash, _ := cache.GetHEAD()
		c, _ = cache.Load(headHash)
	}

	filteredFiles, commits, err := gatherData(ctx, cfg, true)
	if err != nil {
		return err
	}

	if cfg.Format == "table" {
		format.PrintReportHeader("Knowledge Silos", cfg.Since, len(filteredFiles), len(commits))
	}

	result := processor.ProcessRepository(ctx, filteredFiles, commits, !cfg.NoProgress, 0, c)

	// Save cache
	if c != nil {
		_ = c.Save()
	}

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
	// Load cache
	var c *cache.Cache
	if !cfg.NoCache {
		headHash, _ := cache.GetHEAD()
		c, _ = cache.Load(headHash)
	}

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

	result := processor.ProcessRepository(ctx, filteredFiles, commits, showProgress, 0, c)

	// Save cache
	if c != nil {
		_ = c.Save()
	}

	awards := processor.ProcessAwards(result.Result, showProgress)

	switch cfg.Format {
	case "json":
		return format.PrintAwardsJSON(awards)
	case "csv":
		return format.PrintAwardsCSV(awards)
	case "markdown":
		return format.PrintAwardsMarkdown(awards)
	default:
		format.PrintAwards(awards)
	}
	return nil
}

func runHotspot(ctx context.Context, cfg models.Config) error {
	// Load cache
	var c *cache.Cache
	if !cfg.NoCache {
		headHash, _ := cache.GetHEAD()
		c, _ = cache.Load(headHash)
	}

	filteredFiles, commits, err := gatherData(ctx, cfg, true)
	if err != nil {
		return err
	}

	showProgress := !cfg.NoProgress
	if cfg.Format == "json" || cfg.Format == "csv" {
		showProgress = false
	}

	if cfg.Format == "table" {
		format.PrintReportHeader("Hotspot Analysis", cfg.Since, len(filteredFiles), len(commits))
	}

	result := processor.ProcessRepository(ctx, filteredFiles, commits, showProgress, 0, c)

	// Save cache
	if c != nil {
		_ = c.Save()
	}

	hotspots := processor.ProcessHotspots(result, commits, cfg.Window, time.Now())

	switch cfg.Format {
	case "json":
		return format.PrintHotspotsJSON(hotspots)
	case "csv":
		return format.PrintHotspotsCSV(hotspots)
	case "markdown":
		return format.PrintHotspotsMarkdown(hotspots)
	default:
		format.PrintHotspots(hotspots)
	}
	return nil
}

func runHygiene(ctx context.Context, cfg models.Config) error {
	_, commits, err := gatherData(ctx, cfg, false)
	if err != nil {
		return err
	}

	if cfg.Format == "table" {
		format.PrintReportHeader("Commit Message Hygiene", cfg.Since, 0, len(commits))
	}

	hygiene := processor.ProcessCommitHygiene(commits)

	switch cfg.Format {
	case "json":
		return format.PrintCommitHygieneJSON(hygiene)
	case "csv":
		return format.PrintCommitHygieneCSV(hygiene)
	case "markdown":
		return format.PrintCommitHygieneMarkdown(hygiene)
	default:
		format.PrintCommitHygiene(hygiene)
	}
	return nil
}

func runBranches(ctx context.Context, cfg models.Config) error {
	if err := gitcmd.IsValidRepo(ctx); err != nil {
		return err
	}

	report := processor.ProcessBranchHealth(ctx, cfg.Days, cfg.IncludeRemote)

	var filtered []models.BranchHealthRecord
	for _, b := range report.Branches {
		if cfg.Stale && !b.IsStale {
			continue
		}
		if cfg.Unmerged && !b.IsUnmerged {
			continue
		}
		if cfg.Orphaned && !b.IsOrphaned {
			continue
		}
		filtered = append(filtered, b)
	}

	filteredReport := models.BranchHealthReport{
		Branches:      filtered,
		DefaultBranch: report.DefaultBranch,
		TotalCount:    len(filtered),
	}

	staleCount := 0
	unmergedCount := 0
	orphanedCount := 0
	for _, b := range filtered {
		if b.IsStale {
			staleCount++
		}
		if b.IsUnmerged {
			unmergedCount++
		}
		if b.IsOrphaned {
			orphanedCount++
		}
	}
	filteredReport.StaleCount = staleCount
	filteredReport.UnmergedCount = unmergedCount
	filteredReport.OrphanedCount = orphanedCount

	switch cfg.Format {
	case "json":
		_ = format.PrintBranchHealthJSON(filteredReport)
	case "csv":
		_ = format.PrintBranchHealthCSV(filteredReport)
	case "markdown":
		_ = format.PrintBranchHealthMarkdown(filteredReport)
	default:
		format.PrintBranchHealth(filteredReport)
	}

	// Handle purge mode
	if cfg.Purge && len(filtered) > 0 {
		return purgeBranches(ctx, filtered, filteredReport.DefaultBranch, cfg.Force)
	}

	return nil
}

// purgeBranches deletes the listed branches with safety checks.
func purgeBranches(ctx context.Context, branches []models.BranchHealthRecord, defaultBranch string, force bool) error {
	// Get current branch to protect it
	currentBranch, err := gitcmd.GetCurrentBranch(ctx)
	if err != nil {
		return fmt.Errorf("cannot determine current branch: %w", err)
	}

	// Filter out protected branches
	var toDelete []models.BranchHealthRecord
	for _, b := range branches {
		if b.Name == defaultBranch {
			fmt.Fprintf(os.Stderr, "Skipping default branch: %s\n", b.Name)
			continue
		}
		if b.Name == currentBranch {
			fmt.Fprintf(os.Stderr, "Skipping current branch: %s\n", b.Name)
			continue
		}
		if b.IsHead {
			fmt.Fprintf(os.Stderr, "Skipping HEAD branch: %s\n", b.Name)
			continue
		}
		toDelete = append(toDelete, b)
	}

	if len(toDelete) == 0 {
		fmt.Fprintln(os.Stderr, "No branches to delete (all were protected)")
		return nil
	}

	// Show preview
	fmt.Fprintf(os.Stderr, "\n⚠️  Will delete %d branch(es):\n", len(toDelete))
	for _, b := range toDelete {
		status := ""
		if b.IsStale {
			status += "stale "
		}
		if b.IsUnmerged {
			status += "unmerged "
		}
		fmt.Fprintf(os.Stderr, "  - %s (%s- %d days old)\n", b.Name, status, b.DaysSinceLastCommit)
	}

	if !force {
		fmt.Fprintln(os.Stderr, "\nUse --force to actually delete these branches")
		return nil
	}

	// Delete branches
	fmt.Fprintln(os.Stderr, "\nDeleting branches...")
	var deleted, failed int
	for _, b := range toDelete {
		if err := gitcmd.DeleteBranch(ctx, b.Name, true); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", b.Name, err)
			failed++
		} else {
			fmt.Fprintf(os.Stderr, "  ✓ %s\n", b.Name)
			deleted++
		}
	}

	fmt.Fprintf(os.Stderr, "\nDeleted: %d, Failed: %d\n", deleted, failed)
	if failed > 0 {
		return fmt.Errorf("some branches could not be deleted")
	}
	return nil
}

func filterFiles(files []string, include, exclude string) ([]string, error) {
	// Common large generated files/extensions to skip by default
	skipExts := map[string]bool{
		".lock":      true,
		".json-lock": true,
		".map":       true,
		".jar":       true,
		".zip":       true,
		".tar":       true,
		".gz":        true,
		".bz2":       true,
		".7z":        true,
		".rar":       true,
		".png":       true,
		".jpg":       true,
		".jpeg":      true,
		".gif":       true,
		".bmp":       true,
		".svg":       true,
		".ico":       true,
		".pdf":       true,
		".doc":       true,
		".docx":      true,
		".xls":       true,
		".xlsx":      true,
		".ppt":       true,
		".pptx":      true,
		".exe":       true,
		".dll":       true,
		".so":        true,
		".dylib":     true,
		".bin":       true,
		".dat":       true,
		".o":         true,
		".a":         true,
		".class":     true,
		".pyc":       true,
		".pyo":       true,
		".whl":       true,
		".woff":      true,
		".woff2":     true,
		".ttf":       true,
		".otf":       true,
		".eot":       true,
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
