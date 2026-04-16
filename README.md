# git-lord 👑

[![Go Report Card](https://goreportcard.com/badge/github.com/chege/git-lord)](https://goreportcard.com/report/github.com/chege/git-lord)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**High-performance Git contributor intelligence suite.** 

`git-lord` analyzes ownership silos, code retention, and team momentum. It replaces standard counters with deep behavioral insights and a gamified award ceremony.

---

## ⚡️ Fast. Precise. Actionable.

Most Git stats tools get slow as your repo grows. `git-lord` is built in Go and utilizes parallel native processing to deliver results in milliseconds, not minutes.

`git-lord` now keeps a repo-local cache under `.git/git-lord-cache` and reuses it until `HEAD` or tracked file contents change.

## 🚀 Subcommands

### 🏆 Ownership Leaderboard (`git lord`)
The "King of the Hill" view. See who owns the code, who is retaining it, and where the knowledge silos are.
- **Progressive Disclosure:** Use `--silos`, `--social`, or `--all` to reveal deeper metrics.
- **Bus Factor:** See at a glance how many people need to "get hit by a bus" before your repo is in trouble.

### ⚡ Activity Pulse (`git lord pulse`)
What's happening *right now*? 
- **Velocity:** Analyze the last 7, 30, or 90 days.
- **Net Impact:** Highlight the "Code Janitors" who are deleting more than they add.
- **Code Churn:** Surface who is rewriting, cleaning up, or touching the most code in a given window.

### 🎖️ The Award Ceremony (`git lord awards`)
Behavioral analysis turned into a game. 
- **🧹 The Janitor:** Highest refactor impact.
- **🤠 Indiana Jones:** Owner of the oldest surviving code.
- **📚 The Novelist:** Most descriptive commit logs.
- **🏎️ Speed Demon:** Shortest average time between commits.
- **🏰 The Landlord:** Most exclusively owned files.
- **🦜 The Polyglot:** Widest range of file types touched.
- **🌲 The Evergreen:** Highest surviving-line retention ratio.
- **🏃 The Marathoner:** Most active months across repo history.

### 🏛️ Archaeology (`git lord legacy` & `silos`)
- **Legacy:** Breakdown your surviving lines by the year they were written.
- **Silos:** A "Risk Map" identifying large files owned by only one person.

### 🧼 Commit Hygiene (`git lord hygiene`)
Analyzes commit message quality across the team:
- **Too Short:** Subject lines under 15 characters or fewer than 3 words
- **Vague Messages:** Generic terms like "fix", "update", "wip"
- **Conventional Format:** Compliance with conventional commits (feat:, fix:, etc.)
- **Issue References:** Presence of ticket/bug references
- **Commit Body:** Commits with descriptive body text

### 🌿 Branch Health (`git lord branches`)
Analyzes repository branches to identify cleanup opportunities:
- **Stale branches**: No commits in configurable days (default 90)
- **Unmerged branches**: Work not yet merged to default branch
- **Orphaned branches**: Both stale AND unmerged (abandoned work)

**Usage:**
```bash
# Show all branches with health status
git lord branches

# Filter to stale branches only
git lord branches --stale --days 90

# Find unmerged feature branches
git lord branches --unmerged

# Identify abandoned work
git lord branches --orphaned

# Include remote branches in analysis
git lord branches --include-remote

# Export report as markdown for documentation
git lord branches --format markdown > BRANCH_HEALTH.md
```

**Flags:**
| Flag | Default | Description |
|------|---------|-------------|
| `--stale` | false | Filter to stale branches only |
| `--unmerged` | false | Filter to unmerged branches only |
| `--orphaned` | false | Filter to orphaned branches only |
| `--days` | 90 | Days threshold for stale detection |
| `--include-remote` | false | Include remote branches in analysis |
| `--format` | table | Output: table, json, csv, markdown |

### 🕵️ Bug Triage (`git lord suspects`)
Ranks likely bug-introducing commits for one tracked text file.
- **Single File Focus:** Investigate one current-path file at a time.
- **Transparent Scoring:** See recency, file churn, ownership familiarity, and message-risk components.
- **Read-Only Workflow:** Produces a shortlist for manual regression testing; it does not run `git bisect`.

---

## 📦 Installation

### Using Homebrew (Recommended)

```bash
brew tap chege/tap
brew install --cask git-lord
```

Or install directly:

```bash
brew install --cask chege/tap/git-lord
```

### Using Go

Install the latest version directly to your `$GOBIN`:

```bash
go install github.com/chege/git-lord/cmd/git-lord@latest
```

---

## 🛠 Usage

```bash
# Basic leaderboard (git-fame style)
git lord

# High-risk file analysis
git lord silos --min-loc 100

# Recent team momentum
git lord pulse --days 14

# Export a churn window for scripts or spreadsheets
git lord pulse --days 30 --format csv

# The trophy cabinet
git lord awards

# Generate markdown reports for documentation or CI/CD
git lord --format markdown > CONTRIBUTORS.md
git lord pulse --days 14 --format markdown > PULSE_REPORT.md

# Analyze commit message hygiene
git lord hygiene
git lord hygiene --format markdown > HYGIENE_REPORT.md

# Rank likely suspect commits for a file before manual triage
git lord suspects target.go

# Export suspect commits as JSON for scripts or notes
git lord suspects target.go --format json --limit 5
```

### Bug triage suspect workflows

Use `git lord suspects` when you already know the file that is misbehaving and want a shortlist of commits to test manually.

```bash
# Start with the default ranked shortlist for one tracked file
git lord suspects internal/api/handler.go

# Narrow the search to a known regression window
git lord suspects internal/api/handler.go --since 2026-03-01

# Sort by raw target-file churn instead of the blended score
git lord suspects internal/api/handler.go --sort churn

# Sort by newest-first when recency matters more than churn
git lord suspects internal/api/handler.go --sort date

# Save a machine-readable shortlist for incident notes or scripts
git lord suspects internal/api/handler.go --format json --limit 10
```

- `git lord suspects` is intentionally **single-file only** in v1.
- The command uses the file’s **current path name only** and does **not** follow renames in v1.
- The output is a **read-only shortlist** for manual triage; it does **not** modify the repo or run `git bisect`.

### Churn-focused pulse workflows

Use `git lord pulse` when you want a fast read on recent code churn instead of long-term ownership. The pulse report exposes `commits`, `additions`, `deletions`, `net`, `churn`, and `files`, so it is a good fit for refactors, cleanup pushes, and migration windows.

```bash
# Catch intense cleanup or rewrite bursts from the last week
git lord pulse --days 7

# Review monthly churn to spot sustained refactors and janitorial work
git lord pulse --days 30

# Find broad file-touching work during a release cycle
git lord pulse --days 90

# Combine with a precise date when investigating a migration window
git lord pulse --since 2026-02-01

# Rank contributors by raw churn using JSON output
git lord pulse --days 30 --format json | jq 'sort_by(-.churn)[] | {author, churn, net, commits, files}'

# Find the most net-negative cleanup work in the same window
git lord pulse --days 30 --format json | jq 'sort_by(.net)[] | {author, net, deletions, additions}'

# Sort a CSV export by churn in a spreadsheet or shell pipeline
git lord pulse --days 30 --format csv
```

- `--days 7` is useful for incident response, hotfixes, and short refactor spikes.
- `--days 30` works well for sprint or monthly review.
- `--days 90` helps surface larger migrations that fan out across many files.
- `--format json` is the easiest way to build churn-first rankings by `churn`, `net`, or `files` in shell tooling.
- `--format csv` is useful when you want to sort and compare windows in a spreadsheet.

### Global Flags

| Flag | Default | Description |
| :--- | :--- | :--- |
| `--all` | `false` | Show every single metric and column. |
| `--sort` | `loc` | Sort by command-specific metrics; `pulse` supports `commits`, `additions`, `deletions`, `net`, `churn`, and `files`, while `suspects` supports `score`, `date`, and `churn`. |
| `--format` | `table` | Output: `table`, `json`, `csv`, `markdown`. |
| `--since` | `""` | Filter by date (e.g. "2023-01-01"). |
| `--no-progress` | `false` | Hide the ASCII spinner. |

---

Developed with ❤️ for the Git community. Rule your repository.
