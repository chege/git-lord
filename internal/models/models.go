package models

import (
	"time"

	"github.com/chege/git-lord/internal/gitcmd"
)

// Award Thresholds
const (
	JanitorDeletionThreshold  = 50
	NovelistCommitThreshold   = 5
	StealthActiveDayThreshold = 10
)

// Config holds global and command-specific configuration.
type Config struct {
	Sort       string
	Since      string
	Include    string
	Exclude    string
	Format     string
	NoHours    bool
	NoProgress bool
	NoCache    bool
	Days       int
	ShowAll    bool
	ShowSilos  bool
	ShowSocial bool
	MinLOC     int
	Window     int // Analysis window in days for hotspot command
	Version    bool
	// Branch-specific flags
	Stale         bool
	Unmerged      bool
	Orphaned      bool
	IncludeRemote bool
	Purge         bool // Delete branches after listing
	Force         bool // Skip confirmation prompts
}

// AuthorMetrics holds all statistics for a single author.
type AuthorMetrics struct {
	Name              string
	Email             string
	Loc               int
	Commits           int
	Files             int
	ExclusiveFiles    int
	Hours             int
	Months            int
	MaxGap            int
	FirstCommit       int64
	LastCommit        int64
	LifetimeAdditions int
	LifetimeDeletions int

	// Award specific metrics
	MessageWords    int
	ActiveDays      map[string]bool
	FridayAfterFour int
	OldestLineTs    int64
	LintCommits     int
	MergeCommits    int
	FileExtensions  map[string]bool
	CommitIntervals []int64
}

// GlobalMetrics holds the sum of all metrics.
type GlobalMetrics struct {
	TotalLoc     int
	TotalCommits int
	TotalFiles   int
	TotalHours   int
	TotalMonths  int
	BusFactor    int
}

// Result holds the final processed data.
type Result struct {
	Authors map[string]*AuthorMetrics
	Global  GlobalMetrics
}

// AuthorStat extends AuthorMetrics with calculated distribution percentages.
type AuthorStat struct {
	AuthorMetrics
	LocDist   float64  `json:"loc_percent"`
	ComsDist  float64  `json:"commits_percent"`
	FilesDist float64  `json:"files_percent"`
	Retention float64  `json:"retention_percent"`
	Badges    []string `json:"badges"`
}

// PulseStat holds metrics for recent activity.
type PulseStat struct {
	Name      string `json:"author"`
	Commits   int    `json:"commits"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Net       int    `json:"net"`
	Churn     int    `json:"churn"`
	Files     int    `json:"files"`
}

// Award represents a high-level achievement.
type Award struct {
	ID          string
	Title       string
	Emoji       string
	Winner      string
	Vibe        string
	Description string
	Value       string
}

// LegacyStat holds LOC count per year.
type LegacyStat struct {
	Year int
	Loc  int
	Pct  float64
}

// SiloRecord tracks files with dangerously low ownership diversity.
type SiloRecord struct {
	Path      string
	LOC       int
	Owner     string
	Ownership float64 // % owned by the primary owner
}

// TrendStat holds project growth metrics over time.
type TrendStat struct {
	Period    string // e.g. "2023-01"
	Additions int
	Deletions int
	Net       int
}

// HotspotRecord captures the ranked hotspot metrics for one file.
type HotspotRecord struct {
	Path           string  `json:"path"`
	Score          int     `json:"score"`
	Risk           string  `json:"risk"`
	LOC            int     `json:"loc"`
	RecentChurn    int     `json:"recent_churn"`
	RecentCommits  int     `json:"recent_commits"`
	PrimaryOwner   string  `json:"primary_owner"`
	OwnerLines     int     `json:"owner_lines"`
	OwnershipPct   float64 `json:"ownership_pct"`
	ActiveOwners   int     `json:"active_owners"`
	ChurnScore     int     `json:"churn_score"`
	OwnershipScore int     `json:"ownership_score"`
	SizeScore      int     `json:"size_score"`
}

// HotspotReport is the processor output for ranked hotspot analysis.
type HotspotReport struct {
	WindowDays int             `json:"window_days"`
	Hotspots   []HotspotRecord `json:"hotspots"`
}

type CommitHygieneRecord struct {
	Author              string  `json:"author"`
	Email               string  `json:"email"`
	TotalCommits        int     `json:"total_commits"`
	TooShort            int     `json:"too_short"`
	TooShortPct         float64 `json:"too_short_pct"`
	VagueMessages       int     `json:"vague_messages"`
	VaguePct            float64 `json:"vague_pct"`
	MissingConventional int     `json:"missing_conventional"`
	ConventionalPct     float64 `json:"conventional_pct"`
	MissingIssueRef     int     `json:"missing_issue_ref"`
	IssueRefPct         float64 `json:"issue_ref_pct"`
	HasBody             int     `json:"has_body"`
	BodyPct             float64 `json:"body_pct"`
	AvgMessageLength    float64 `json:"avg_message_length"`
	HygieneScore        float64 `json:"hygiene_score"`
}

type CommitHygieneReport struct {
	Authors []CommitHygieneRecord `json:"authors"`
}

// BranchHealthRecord holds health metrics for a single branch.
type BranchHealthRecord struct {
	Name                string    `json:"name"`
	IsRemote            bool      `json:"is_remote"`
	IsHead              bool      `json:"is_head"`
	LastCommit          time.Time `json:"last_commit"`
	LastAuthor          string    `json:"last_author"`
	LastSubject         string    `json:"last_subject"`
	CommitCount         int       `json:"commit_count"`
	IsMerged            bool      `json:"is_merged"`
	Behind              int       `json:"behind"`
	Ahead               int       `json:"ahead"`
	DaysSinceLastCommit int       `json:"days_since_last_commit"`
	StaleDays           int       `json:"stale_days"`
	IsStale             bool      `json:"is_stale"`
	IsUnmerged          bool      `json:"is_unmerged"`
	IsOrphaned          bool      `json:"is_orphaned"`
}

// BranchHealthReport is the aggregate report for branch health analysis.
type BranchHealthReport struct {
	Branches      []BranchHealthRecord `json:"branches"`
	DefaultBranch string               `json:"default_branch"`
	TotalCount    int                  `json:"total_count"`
	StaleCount    int                  `json:"stale_count"`
	UnmergedCount int                  `json:"unmerged_count"`
	OrphanedCount int                  `json:"orphaned_count"`
}

// CommitData is an alias to avoid circular dependencies if needed,
// but for now we'll just import gitcmd.
type CommitData = gitcmd.CommitData
