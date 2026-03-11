# git-lord 👑

`git-lord` is a high-performance Git contributor intelligence suite. It analyzes ownership silos, code retention, and activity velocity with gamified behavioral awards.

Designed for speed, `git-lord` utilizes parallel native Git processing to deliver instant insights even on large repositories.

## 🚀 Key Features

### 🏆 The Leaderboard (Default)
The classic ownership view, refined.
- **LOC & Distribution**: Who owns the most surviving code.
- **Retention %**: How much of an author's lifetime work still exists.
- **Exclusivity**: Identify knowledge silos (files owned 100% by one person).
- **Bus Factor**: The ultimate repository health metric.
- **Max Gap**: Identify the most consistent (and the ghosting) contributors.

### ⚡ Activity Pulse (`git lord pulse`)
Instant activity metrics without the heavy blame pass.
- **Velocity**: Commits, additions, and deletions in the last N days.
- **Net Impact**: See who is cleaning up technical debt (negative net lines).
- **Churn**: Total volume of code change.

### 🧛 Vampire Stats (`git lord night-owl`)
Understand the team's working rhythm with emoji distributions.
- **Time Windows**: Morning ☀️, Afternoon 🌤️, Evening 🌙, and Night 🧛.
- **Flow State**: Shortest average time between commits.

### 🎖️ The Awards Ceremony (`git lord awards`)
Unlock behavioral trophies based on Git history.
- **🧹 The Janitor**: Most technical debt removed.
- **🤠 Indiana Jones**: Author of the oldest surviving line of code.
- **📚 The Novelist**: Most descriptive commit messages.
- **🎲 Friday Roulette**: Most pushes after 4 PM on a Friday.
- **🧘 Deep Thinker**: Longest streak of strategic meditation (inactivity).

## 📦 Installation

### Quick Install (One-liner)

Install the latest version directly to your `$GOBIN`:

```bash
go install github.com/chege/git-lord/cmd/git-lord@latest
```

_(Make sure your `$(go env GOPATH)/bin` is in your system `$PATH`)_

## 🛠 Usage

```bash
# Standard leaderboard
git lord

# See all columns (Tenure, Retention, Badges, etc.)
git lord --all

# Check the last 7 days of activity
git lord pulse --days 7

# Who are the night owls?
git lord night-owl

# Hold the awards ceremony
git lord awards
```

### Global Options

| Flag         | Default | Description                                                            |
| :----------- | :------ | :--------------------------------------------------------------------- |
| `--all`      | `false` | Reveal all available metrics and columns.                              |
| `--silos`    | `false` | Show Exclusivity, Retention, and Bus Factor.                           |
| `--social`   | `false` | Show Hours, Max Gap, and Badges.                                       |
| `--sort`     | `loc`   | Sort by metric: `loc`, `coms`, `fils`, `hrs`.                          |
| `--since`    | `""`    | Filter history by date (e.g., `"2023-01-01"`, `"2 weeks ago"`).        |
| `--include`  | `""`    | Only include files matching glob pattern (e.g., `"*.go"`).             |
| `--format`   | `table` | Render format: `table`, `json`, `csv`.                                 |

## 🧪 Development

```bash
make test   # Run full E2E and unit suite (caching disabled)
make lint   # Run golangci-lint
```

---
Rule your repository with `git-lord`.
