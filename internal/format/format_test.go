package format

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/chege/git-lord/internal/models"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestPrintReportHeader(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		since   string
		files   int
		commits int
	}{
		{
			name:    "all-time report",
			title:   "Test Report",
			since:   "",
			files:   10,
			commits: 50,
		},
		{
			name:    "date range report",
			title:   "Pulse Report",
			since:   "30 days ago",
			files:   5,
			commits: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			PrintReportHeader(tt.title, tt.since, tt.files, tt.commits)
		})
	}
}

func TestPrintTable(t *testing.T) {
	stats := []models.AuthorStat{
		{
			AuthorMetrics: models.AuthorMetrics{
				Name:    "Alice",
				Loc:     100,
				Commits: 10,
				Files:   5,
			},
			LocDist:   50.0,
			ComsDist:  50.0,
			FilesDist: 50.0,
		},
	}

	global := models.GlobalMetrics{
		TotalLoc:     200,
		TotalCommits: 20,
		TotalFiles:   10,
	}

	cfg := models.Config{}

	output := captureOutput(func() {
		PrintTable(stats, global, cfg)
	})

	if !strings.Contains(output, "Alice") {
		t.Error("output missing author name")
	}
}

func TestPrintPulse(t *testing.T) {
	stats := []models.PulseStat{
		{
			Name:      "Alice",
			Commits:   10,
			Additions: 100,
			Deletions: 50,
			Net:       50,
			Churn:     150,
			Files:     5,
		},
	}

	output := captureOutput(func() {
		PrintPulse(stats)
	})

	if !strings.Contains(output, "Alice") {
		t.Error("output missing author name")
	}
}

func TestPrintSilos(t *testing.T) {
	tests := []struct {
		name  string
		silos []models.SiloRecord
	}{
		{
			name:  "empty silos",
			silos: []models.SiloRecord{},
		},
		{
			name: "with silos",
			silos: []models.SiloRecord{
				{
					Path:      "internal/app.go",
					LOC:       500,
					Owner:     "alice@example.com",
					Ownership: 95.0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				PrintSilos(tt.silos)
			})

			if len(tt.silos) == 0 {
				if !strings.Contains(output, "No high-risk silos") {
					t.Error("empty silos should show success message")
				}
			} else {
				if !strings.Contains(output, tt.silos[0].Path) {
					t.Error("output missing file path")
				}
			}
		})
	}
}

func TestPrintTrends(t *testing.T) {
	trends := []models.TrendStat{
		{
			Period:    "2024-01",
			Additions: 100,
			Deletions: 50,
			Net:       50,
		},
	}

	output := captureOutput(func() {
		PrintTrends(trends)
	})

	if !strings.Contains(output, "2024-01") {
		t.Error("output missing period")
	}
}

func TestPrintLegacy(t *testing.T) {
	stats := []models.LegacyStat{
		{
			Year: 2024,
			Loc:  100,
			Pct:  50.0,
		},
	}

	output := captureOutput(func() {
		PrintLegacy(stats)
	})

	if !strings.Contains(output, "2024") {
		t.Error("output missing year")
	}
}

func TestPrintAwards(t *testing.T) {
	awards := []models.Award{
		{
			ID:          "test",
			Title:       "Test Award",
			Emoji:       "🏆",
			Winner:      "Alice",
			Vibe:        "Testing vibe",
			Description: "Test description",
			Value:       "100",
		},
	}

	PrintAwards(awards)
}

func TestPrintAwardsJSON(t *testing.T) {
	awards := []models.Award{
		{
			ID:     "test",
			Title:  "Test Award",
			Winner: "Alice",
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintAwardsJSON(awards)
	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("PrintAwardsJSON() error = %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	var result []models.Award
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if len(result) != 1 || result[0].Title != "Test Award" {
		t.Error("JSON output does not match input")
	}
}

func TestPrintAwardsCSV(t *testing.T) {
	awards := []models.Award{
		{
			ID:          "test",
			Title:       "Test Award",
			Winner:      "Alice",
			Description: "Test desc",
			Vibe:        "Test vibe",
			Value:       "100",
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintAwardsCSV(awards)
	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("PrintAwardsCSV() error = %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	reader := csv.NewReader(strings.NewReader(buf.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Errorf("output is not valid CSV: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("expected 2 CSV records (header + data), got %d", len(records))
	}
}

func TestPrintJSON(t *testing.T) {
	stats := []models.AuthorStat{
		{
			AuthorMetrics: models.AuthorMetrics{Name: "Alice", Loc: 100},
			LocDist:       50.0,
		},
	}

	global := models.GlobalMetrics{TotalLoc: 200}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintJSON(stats, global)
	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("PrintJSON() error = %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	var result struct {
		Authors []models.AuthorStat `json:"authors"`
		Global  models.GlobalMetrics
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if len(result.Authors) != 1 {
		t.Error("JSON output missing authors")
	}
}

func TestPrintPulseJSON(t *testing.T) {
	stats := []models.PulseStat{
		{Name: "Alice", Commits: 10},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintPulseJSON(stats)
	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("PrintPulseJSON() error = %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	var result []models.PulseStat
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if len(result) != 1 {
		t.Error("JSON output missing stats")
	}
}

func TestPrintPulseCSV(t *testing.T) {
	stats := []models.PulseStat{
		{
			Name:      "Alice",
			Commits:   10,
			Additions: 100,
			Deletions: 50,
			Net:       50,
			Churn:     150,
			Files:     5,
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintPulseCSV(stats)
	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("PrintPulseCSV() error = %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	reader := csv.NewReader(strings.NewReader(buf.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Errorf("output is not valid CSV: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("expected 2 CSV records, got %d", len(records))
	}
}

func TestPrintCSV(t *testing.T) {
	stats := []models.AuthorStat{
		{
			AuthorMetrics: models.AuthorMetrics{
				Name:    "Alice",
				Loc:     100,
				Commits: 10,
				Files:   5,
			},
			LocDist: 50.0,
		},
	}

	cfg := models.Config{}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintCSV(stats, cfg)
	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("PrintCSV() error = %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	reader := csv.NewReader(strings.NewReader(buf.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Errorf("output is not valid CSV: %v", err)
	}

	if len(records) < 2 {
		t.Errorf("expected at least 2 CSV records, got %d", len(records))
	}
}

func TestPrintHotspots(t *testing.T) {
	tests := []struct {
		name   string
		report models.HotspotReport
	}{
		{
			name: "empty hotspots",
			report: models.HotspotReport{
				WindowDays: 30,
				Hotspots:   []models.HotspotRecord{},
			},
		},
		{
			name: "with hotspots",
			report: models.HotspotReport{
				WindowDays: 30,
				Hotspots: []models.HotspotRecord{
					{
						Path:         "internal/app.go",
						Score:        80,
						Risk:         "CRITICAL",
						LOC:          500,
						RecentChurn:  200,
						PrimaryOwner: "alice@example.com",
						OwnershipPct: 90.0,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				PrintHotspots(tt.report)
			})

			if len(tt.report.Hotspots) == 0 {
				if !strings.Contains(output, "No high-risk hotspots") {
					t.Error("empty hotspots should show success message")
				}
			}
		})
	}
}

func TestPrintHotspotsJSON(t *testing.T) {
	report := models.HotspotReport{
		WindowDays: 30,
		Hotspots: []models.HotspotRecord{
			{Path: "test.go", Score: 80},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintHotspotsJSON(report)
	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("PrintHotspotsJSON() error = %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	var result models.HotspotReport
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
}

func TestPrintHotspotsCSV(t *testing.T) {
	report := models.HotspotReport{
		WindowDays: 30,
		Hotspots: []models.HotspotRecord{
			{
				Path:           "test.go",
				Score:          80,
				Risk:           "CRITICAL",
				LOC:            100,
				RecentChurn:    50,
				RecentCommits:  5,
				PrimaryOwner:   "alice@example.com",
				OwnerLines:     90,
				OwnershipPct:   90.0,
				ActiveOwners:   1,
				ChurnScore:     50,
				OwnershipScore: 50,
				SizeScore:      50,
			},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintHotspotsCSV(report)
	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("PrintHotspotsCSV() error = %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	reader := csv.NewReader(strings.NewReader(buf.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Errorf("output is not valid CSV: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("expected 2 CSV records, got %d", len(records))
	}
}

func TestPrintMarkdown(t *testing.T) {
	stats := []models.AuthorStat{
		{
			AuthorMetrics: models.AuthorMetrics{
				Name:    "Alice",
				Loc:     100,
				Commits: 10,
				Files:   5,
			},
			LocDist: 50.0,
		},
	}

	global := models.GlobalMetrics{TotalLoc: 200, TotalCommits: 20, TotalFiles: 10}
	cfg := models.Config{}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintMarkdown(stats, global, cfg)
	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("PrintMarkdown() error = %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "##") {
		t.Error("markdown output missing header")
	}
	if !strings.Contains(output, "|") {
		t.Error("markdown output missing table")
	}
}

func TestPrintPulseMarkdown(t *testing.T) {
	stats := []models.PulseStat{
		{
			Name:      "Alice",
			Commits:   10,
			Additions: 100,
			Deletions: 50,
			Net:       50,
			Churn:     150,
			Files:     5,
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintPulseMarkdown(stats)
	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("PrintPulseMarkdown() error = %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "|") {
		t.Error("markdown output missing table")
	}
}

func TestPrintAwardsMarkdown(t *testing.T) {
	awards := []models.Award{
		{
			Title:       "Test Award",
			Emoji:       "🏆",
			Winner:      "Alice",
			Description: "Test description",
			Vibe:        "Test vibe",
			Value:       "100",
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintAwardsMarkdown(awards)
	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("PrintAwardsMarkdown() error = %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "###") {
		t.Error("markdown output missing subheader")
	}
}

func TestPrintHotspotsMarkdown(t *testing.T) {
	tests := []struct {
		name   string
		report models.HotspotReport
	}{
		{
			name: "empty hotspots",
			report: models.HotspotReport{
				WindowDays: 30,
				Hotspots:   []models.HotspotRecord{},
			},
		},
		{
			name: "with hotspots",
			report: models.HotspotReport{
				WindowDays: 30,
				Hotspots: []models.HotspotRecord{
					{
						Path:         "test.go",
						Score:        80,
						Risk:         "CRITICAL",
						LOC:          100,
						RecentChurn:  50,
						PrimaryOwner: "alice@example.com",
						OwnershipPct: 90.0,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := PrintHotspotsMarkdown(tt.report)
			_ = w.Close()
			os.Stdout = old

			if err != nil {
				t.Errorf("PrintHotspotsMarkdown() error = %v", err)
			}

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			output := buf.String()

			if len(tt.report.Hotspots) == 0 {
				if !strings.Contains(output, "No high-risk hotspots") {
					t.Error("empty hotspots should show success message")
				}
			} else {
				if !strings.Contains(output, "|") {
					t.Error("markdown output missing table")
				}
			}
		})
	}
}

func TestPrintCommitHygiene(t *testing.T) {
	report := models.CommitHygieneReport{
		Authors: []models.CommitHygieneRecord{
			{
				Author:          "Alice",
				TotalCommits:    10,
				HygieneScore:    85.0,
				TooShort:        1,
				TooShortPct:     10.0,
				VagueMessages:   0,
				VaguePct:        0.0,
				ConventionalPct: 80.0,
				IssueRefPct:     70.0,
				BodyPct:         30.0,
			},
		},
	}

	output := captureOutput(func() {
		PrintCommitHygiene(report)
	})

	if !strings.Contains(output, "Alice") {
		t.Error("output missing author name")
	}
}

func TestPrintCommitHygieneJSON(t *testing.T) {
	report := models.CommitHygieneReport{
		Authors: []models.CommitHygieneRecord{
			{Author: "Alice", HygieneScore: 85.0},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintCommitHygieneJSON(report)
	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("PrintCommitHygieneJSON() error = %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	var result models.CommitHygieneReport
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
}

func TestPrintCommitHygieneCSV(t *testing.T) {
	report := models.CommitHygieneReport{
		Authors: []models.CommitHygieneRecord{
			{
				Author:           "Alice",
				Email:            "alice@example.com",
				TotalCommits:     10,
				HygieneScore:     85.0,
				TooShort:         1,
				TooShortPct:      10.0,
				VagueMessages:    0,
				VaguePct:         0.0,
				ConventionalPct:  80.0,
				IssueRefPct:      70.0,
				BodyPct:          30.0,
				AvgMessageLength: 25.0,
			},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintCommitHygieneCSV(report)
	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("PrintCommitHygieneCSV() error = %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	reader := csv.NewReader(strings.NewReader(buf.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Errorf("output is not valid CSV: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("expected 2 CSV records, got %d", len(records))
	}
}

func TestPrintCommitHygieneMarkdown(t *testing.T) {
	report := models.CommitHygieneReport{
		Authors: []models.CommitHygieneRecord{
			{
				Author:          "Alice",
				TotalCommits:    10,
				HygieneScore:    85.0,
				TooShort:        1,
				TooShortPct:     10.0,
				VagueMessages:   0,
				VaguePct:        0.0,
				ConventionalPct: 80.0,
				IssueRefPct:     70.0,
				BodyPct:         30.0,
			},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintCommitHygieneMarkdown(report)
	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("PrintCommitHygieneMarkdown() error = %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "|") {
		t.Error("markdown output missing table")
	}
}
