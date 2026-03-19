# Hotspot Scoring Model

This document defines the scoring model for the planned hotspot report so the
processor, formatter, CLI, and tests all target the same behavior.

## Goal

Rank files that are risky because they change often, are large enough to matter,
and have understanding concentrated in too few hands.

The model should bias toward recent churn so the report surfaces active risk,
not just historically concentrated ownership.

## Data Sources

- Current file size and ownership reuse blame-derived `FileOwners` data from
  repository processing.
- Recent churn aggregates per-file additions, deletions, and touch count from
  `git log --numstat` over a recent window.
- Tracked files only; ignore deleted files and files with no current tracked
  path.
- Rename-aware history should use `git log -M` so renamed files keep their
  recent churn.
- Ignore binary-file `numstat` rows with `-` values.

## Default Analysis Window

- Default recent window: 30 days.
- The CLI can make the window configurable later, but the processor should use
  30 days when no explicit window is provided.

## Eligibility Gate

Only score files that meet all of the following:

- current `loc >= 50`
- `recent_churn > 0` in the analysis window
- file still exists in the tracked file set

This keeps the report focused on meaningful, currently-lived code.

## Ownership Terms

For each eligible file:

- `primary_owner`: author or email with the most current blamed lines
- `owner_lines`: current blamed lines for the primary owner
- `ownership_pct`: `owner_lines / loc * 100`
- `active_owners`: count of authors with at least one current blamed line in the file

Ownership concentration is driven primarily by `ownership_pct`. `active_owners`
is retained as supporting context and for tie-breaking and display.

## Component Scores

All component scores are normalized to `0..100`. Round the final hotspot score
to the nearest whole number.

### 1. Churn Score (50% weight)

Churn dominates the ranking because the report is about active risk.

Inputs:

- `recent_churn = additions + deletions` over the recent window
- `recent_commits =` number of commits that touched the file in the recent window

Subscores:

- `churn_volume_score = min(100, recent_churn / 200 * 100)`
- `touch_frequency_score = min(100, recent_commits / 8 * 100)`

Combine:

- `churn_score = round(0.7 * churn_volume_score + 0.3 * touch_frequency_score)`

Interpretation:

- about 200 changed lines in-window is enough to max the volume portion
- about 8 touches in-window is enough to max the frequency portion

### 2. Ownership Score (30% weight)

Ownership risk stays low until one person clearly dominates the file.

Formula:

- if `ownership_pct <= 50`, `ownership_score = 0`
- else `ownership_score = min(100, (ownership_pct - 50) / 45 * 100)`

Effects:

- 50% ownership => 0 concentration risk
- 80% ownership => about 67 ownership score
- 95%+ ownership => 100 ownership score

This aligns with the repo's existing silo semantics, where 80% is already high
and 95% is critical.

### 3. Size Score (20% weight)

Size matters, but should not outrank churn.

Formula:

- `size_score = min(100, loc / 800 * 100)`

Effects:

- 50 LOC => 6.25 size score
- 400 LOC => 50 size score
- 800+ LOC => 100 size score

## Final Hotspot Score

`hotspot_score = round(0.5 * churn_score + 0.3 * ownership_score + 0.2 * size_score)`

This keeps the score on a familiar 0-100 scale and makes weighting explicit.

## Ranking Rules

Sort descending by:

1. `hotspot_score`
2. `recent_churn`
3. `ownership_pct`
4. `loc`
5. `path` ascending for deterministic output

## Risk Bands

Map final score to severity labels:

- `CRITICAL` => `score >= 80`
- `HIGH` => `score >= 65`
- `MEDIUM` => `score >= 50`
- `WATCH` => `score >= 35`
- below 35 => omit from the human-facing table report

## Output Contract

The processor should return enough data for table, JSON, CSV, and tests without
recomputing.

Required fields per hotspot row:

- `path`
- `score`
- `risk`
- `loc`
- `recent_churn`
- `recent_commits`
- `primary_owner`
- `owner_lines`
- `ownership_pct`
- `active_owners`
- `churn_score`
- `ownership_score`
- `size_score`

## Report Shape

- Human-facing table should show the top 15 rows after filtering out scores
  below `WATCH`.
- JSON and CSV should expose the same raw and computed fields so tests and
  downstream tooling can validate exact scoring.

## Non-Goals For First Implementation

- no age or legacy weighting
- no language-aware thresholds
- no per-directory rollups
- no branch-aware or team-aware heuristics

## Why This Model

- Keeps implementation straightforward with data already close at hand.
- Produces a stable, testable 0-100 score.
- Preserves the repo's existing silo severity intuition around 80% and 95%.
- Prioritizes files that are both active and fragile, not merely large or busy.
