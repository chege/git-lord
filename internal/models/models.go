package models

import "github.com/christopher/git-lord/internal/gitcmd"

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
	FirstCommit       int64
	LastCommit        int64
	LifetimeAdditions int
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
	LocDist   float64 `json:"loc_percent"`
	ComsDist  float64 `json:"commits_percent"`
	FilesDist float64 `json:"files_percent"`
	Retention float64 `json:"retention_percent"`
}

// PulseStat holds metrics for recent activity.
type PulseStat struct {
	Name      string `json:"author"`
	Commits   int    `json:"commits"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Churn     int    `json:"churn"`
	Files     int    `json:"files"`
}

// CommitData is an alias to avoid circular dependencies if needed,
// but for now we'll just import gitcmd.
type CommitData = gitcmd.CommitData
