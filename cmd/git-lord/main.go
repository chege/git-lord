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
	sortVar    string
	since      string
	include    string
	exclude    string
	formatVar  string
	noHours    bool
	noProgress bool
}

func main() {
	cfg := config{}

	flag.StringVar(&cfg.sortVar, "sort", "loc", "Sort metric: loc, coms, fils, hrs")
	flag.StringVar(&cfg.since, "since", "", "Filter by date (e.g., '2023-01-01', '2 weeks ago')")
	flag.StringVar(&cfg.include, "include", "", "Include glob pattern")
	flag.StringVar(&cfg.exclude, "exclude", "", "Exclude glob pattern")
	flag.StringVar(&cfg.formatVar, "format", "table", "Output format: table, json, csv")
	flag.BoolVar(&cfg.noHours, "no-hours", false, "Disable hours calculation")
	flag.BoolVar(&cfg.noProgress, "no-progress", false, "Disable progress bar")

	flag.Parse()

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(cfg config) error {
	if err := gitcmd.IsValidRepo(); err != nil {
		return err
	}

	files, err := gitcmd.ListTrackedFiles()
	if err != nil {
		return err
	}

	var filteredFiles []string
	for _, f := range files {
		if cfg.include != "" {
			if match, _ := filepath.Match(cfg.include, f); !match {
				continue
			}
		}
		if cfg.exclude != "" {
			if match, _ := filepath.Match(cfg.exclude, f); match {
				continue
			}
		}
		filteredFiles = append(filteredFiles, f)
	}

	commits, err := gitcmd.GetCommitHistory(cfg.since)
	if err != nil {
		return err
	}

	result := processor.ProcessRepository(filteredFiles, commits)
	stats := format.GenerateStats(result, cfg.sortVar)

	switch cfg.formatVar {
	case "json":
		return format.PrintJSON(stats, result.Global)
	case "csv":
		return format.PrintCSV(stats, cfg.noHours)
	case "table":
		fallthrough
	default:
		format.PrintTable(stats, result.Global, cfg.noHours)
	}
	return nil
}
