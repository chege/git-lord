# Hotspot Scoring Model

This document defines the scoring model for the planned hotspot report so the
processor, formatter, CLI, and tests all target the same behavior.

## Goal

The hotspot report should rank files by near-term change risk, not by a single
raw metric. A file becomes risky when it changes often, is large enough to hurt
when it breaks, and is concentrated in too few hands.

## Inputs

Each candidate file combines three signals:

- `recent_churn`: additions plus deletions touching the file inside the chosen
  reporting window.
- `loc`: current blamed lines of code for the file.
- `ownership_pct`: share of the file owned by the primary owner,
  `primary_owner_loc / loc`, expressed as `0.0` to `1.0`.

Supporting fields:

- `primary_owner`: email or identity already used by blame-backed reports.
- `owner_count`: number of authors with blamed lines in the file.

## Eligibility

Ignore files that are too small or inactive to be meaningful hotspots:

- exclude files with fewer than `20` LOC
- exclude files with `0` recent churn in the selected window

This keeps tiny utility files and dormant files out of the ranking.

## Normalization

Normalize each candidate against the largest value in the eligible set:

- `churn_score = recent_churn / max_recent_churn`
- `size_score = loc / max_loc`
- `concentration_score = clamp((ownership_pct - 0.50) / 0.50, 0, 1)`

Notes:

- `concentration_score` stays `0` until a file is more than 50% owned by one
  person.
- a fully concentrated file scores `1.0`
- if the candidate set is empty, the report is empty

## Composite Score

Compute the final score as:

```text
hotspot_score = churn_score*0.50 + concentration_score*0.35 + size_score*0.15
```

Weighting rationale:

- churn is the strongest predictor of immediate coordination risk
- ownership concentration is the main risk multiplier
- size increases blast radius, but should not dominate the ranking

Keep this formula stable unless a later bead explicitly changes it.

## Risk Bands

Ownership concentration bands:

- `ownership_pct >= 0.90`: `critical`
- `ownership_pct >= 0.75`: `high`
- `ownership_pct >= 0.60`: `elevated`
- otherwise: `normal`

Hotspot score bands:

- `hotspot_score >= 0.75`: `critical`
- `hotspot_score >= 0.50`: `high`
- `hotspot_score >= 0.30`: `medium`
- below `0.30`: omit from the default ranked output

## Ranking

Sort descending by:

1. `hotspot_score`
2. `recent_churn`
3. `ownership_pct`
4. `loc`
5. `path` ascending for deterministic output

Default output should keep the top `15` hotspots after filtering.

## Output Fields

The processor should emit, at minimum:

- `path`
- `score`
- `risk_level`
- `churn`
- `loc`
- `primary_owner`
- `ownership_pct`
- `owner_count`
- `churn_pct`
- `size_pct`
- `concentration_pct`

The normalized percentages make formatter output and test fixtures easier to
explain and debug.

## Implementation Notes

- recent churn needs per-file aggregation from the commit log, not just per
  author totals
- ownership should reuse the blame-derived file owner map already used by the
  silo report
- the hotspot time window should follow the same recent-window behavior used by
  `pulse`
- tests should lock exact ordering so later work does not silently drift from
  this model
