# AGENTS.md

Guidance for coding agents and contributors working in this repository.

## Commits

- Use Angular-style Conventional Commits.
- Prefer scoped subjects when they help, for example:
  - `feat(pulse): add churn sort option`
  - `fix(gitcmd): preserve rename paths in numstat parsing`
  - `test(e2e): stabilize pulse fixture dates`
  - `docs(agents): add repository guidance`
- Avoid vague subjects such as `fix`, `update`, or `wip`.
- Add a brief body when behavior, tradeoffs, or follow-up context matters.

## Go workflow

- Format edited Go files with `gofmt -w`.
- Validate changes with:
  - `make build`
  - `make lint`
  - `go vet ./...`
  - `go test ./...`
- Keep changes small and consistent with the existing structure under `cmd/` and `internal/`.

## Static analysis and CI

- CI checks formatting with `gofmt -l .`.
- Linting runs through `golangci-lint` using the repository's `.golangci.yml`.
- Enabled lint checks include `errcheck`, `govet` (with `shadow`), `ineffassign`, `staticcheck`, and `unused`.
- CI also runs `go vet`, `go test -race`, and coverage generation.
- Commit messages are checked by the repository's `git-lord hygiene` job.
- Dependabot is configured for Go modules and GitHub Actions updates.

## Tests

- End-to-end tests build the CLI and run it against temporary Git repositories.
- For time-windowed tests such as `pulse --days ...`, use commit dates relative to the current time instead of hard-coded calendar dates.
