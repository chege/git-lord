# git-lord

`git-lord` is a high-performance Go-based alternative to the popular `git-fame` command. It computes distribution and contributor metrics for a Git repository but focuses entirely on execution speed for medium-to-large repositories, utilizing parallel blame processing and minimalistic memory footprints.

It perfectly recreates the native metrics you know and love:

- **LOC (Lines of Code)**: Current surviving lines attributed to the author.
- **Commits**: Total commits by the author.
- **Files**: Number of active files where the author owns ≥ 1 line of code.
- **Hours**: Estimated work hours calculated from session-based timestamps (60-minute windows).
- **Months**: Unique active calendar months.
- **Distribution (%)**: Percentages relative to the total repository stats.

## Installation

### From Source

Ensure you have [Go](https://go.dev/doc/install) installed (1.20+ recommended).

1. Clone the repository:

   ```bash
   git clone https://github.com/christopher/git-lord.git
   cd git-lord
   ```

2. Build the binary using the provided `Makefile`:

   ```bash
   make build
   ```

3. Move the binary into your PATH (so Git can find it natively as `git lord`):
   ```bash
   sudo mv bin/git-lord /usr/local/bin/git-lord
   ```

### Quick Install (go install)

Alternatively, you can install it directly via the `go` command:

```bash
go install github.com/christopher/git-lord/cmd/git-lord@latest
```

_(Make sure your `$(go env GOPATH)/bin` is in your system `$PATH`)_

## Usage

Once installed to your `$PATH`, you can invoke it from inside any Git repository simply as:

```bash
git lord
```

### Options / Flags

You can customize the output using the following flags:

| Flag         | Default | Description                                                            |
| :----------- | :------ | :--------------------------------------------------------------------- |
| `--sort`     | `loc`   | Sort output by metric: `loc`, `coms`, `fils`, `hrs`.                   |
| `--since`    | `""`    | Filter commit history by date (e.g., `"2023-01-01"`, `"2 weeks ago"`). |
| `--include`  | `""`    | Only include files matching this glob pattern (e.g., `"*.go"`).        |
| `--exclude`  | `""`    | Exclude files matching this glob pattern.                              |
| `--format`   | `table` | Render format: `table`, `json`, `csv`.                                 |
| `--no-hours` | `false` | Disable hours/months calculation entirely (speeds up processing).      |

## Development & Testing

We provide a robust test suite covering metrics and End-to-End full mock repository validation. Run the test suite using standard Go commands or the Makefile:

```bash
make test
# OR
go test -v ./...
```

To run linting checks (requires `golangci-lint`):

```bash
make lint
```
