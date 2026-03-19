# git-lord 👑

[![Go Report Card](https://goreportcard.com/badge/github.com/chege/git-lord)](https://goreportcard.com/report/github.com/chege/git-lord)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**High-performance Git contributor intelligence suite.** 

`git-lord` analyzes ownership silos, code retention, and team momentum. It replaces standard counters with deep behavioral insights and a gamified award ceremony.

---

## ⚡️ Fast. Precise. Actionable.

Most Git stats tools get slow as your repo grows. `git-lord` is built in Go and utilizes parallel native processing to deliver results in milliseconds, not minutes.

## 🚀 Subcommands

### 🏆 Ownership Leaderboard (`git lord`)
The "King of the Hill" view. See who owns the code, who is retaining it, and where the knowledge silos are.
- **Progressive Disclosure:** Use `--silos`, `--social`, or `--all` to reveal deeper metrics.
- **Bus Factor:** See at a glance how many people need to "get hit by a bus" before your repo is in trouble.

### ⚡ Activity Pulse (`git lord pulse`)
What's happening *right now*? 
- **Velocity:** Analyze the last 7, 30, or 90 days.
- **Net Impact:** Highlight the "Code Janitors" who are deleting more than they add.

### 🎖️ The Award Ceremony (`git lord awards`)
Behavioral analysis turned into a game. 
- **🧹 The Janitor:** Highest refactor impact.
- **🤠 Indiana Jones:** Owner of the oldest surviving code.
- **📚 The Novelist:** Most descriptive commit logs.
- **🏎️ Speed Demon:** Shortest average time between commits.

### 🏛️ Archaeology (`git lord legacy` & `silos`)
- **Legacy:** Breakdown your surviving lines by the year they were written.
- **Silos:** A "Risk Map" identifying large files owned by only one person.

---

## 📦 Installation

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

# The trophy cabinet
git lord awards
```

### Global Flags

| Flag | Default | Description |
| :--- | :--- | :--- |
| `--all` | `false` | Show every single metric and column. |
| `--sort` | `loc` | Sort by command-specific metrics; `pulse` supports `commits`, `additions`, `deletions`, `net`, `churn`, and `files`. |
| `--format` | `table` | Output: `table`, `json`, `csv`. |
| `--since` | `""` | Filter by date (e.g. "2023-01-01"). |
| `--no-progress` | `false` | Hide the ASCII spinner. |

---

Developed with ❤️ for the Git community. Rule your repository.
