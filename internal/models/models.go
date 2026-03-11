package models

import "github.com/chege/git-lord/internal/gitcmd"

// Award Thresholds
const (
	JanitorDeletionThreshold = 50
	NovelistCommitThreshold  = 5
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
	Days       int
	ShowAll    bool
	ShowSilos  bool
	ShowSocial bool
	MinLOC     int
	Version    bool
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
	MessageWords      int
	ActiveDays        map[string]bool
	FridayAfterFour   int
	OldestLineTs      int64
	LintCommits       int
	MergeCommits      int
	FileExtensions    map[string]bool
	CommitIntervals   []int64
}

// GlobalMetrics holds the sum of all metrics.
type GlobalMetrics struct {
	TotalLoc       int
	TotalCommits   int
	TotalFiles     int
	TotalHours     int
	TotalMonths    int
	BusFactor      int
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

// CommitData is an alias to avoid circular dependencies if needed,
// but for now we'll just import gitcmd.
type CommitData = gitcmd.CommitData
