package processor

import (
	"sort"
	"time"

	"github.com/chege/git-lord/internal/models"
)

// ProcessLegacy breakdown LOC by year.
func ProcessLegacy(res models.Result) []models.LegacyStat {
	yearMap := make(map[int]int)
	total := 0

	for _, a := range res.Authors {
		if a.OldestLineTs == 0 {
			continue
		}
		year := time.Unix(a.OldestLineTs, 0).Year()
		yearMap[year] += a.Loc
		total += a.Loc
	}

	var stats []models.LegacyStat
	for year, loc := range yearMap {
		pct := 0.0
		if total > 0 {
			pct = float64(loc) / float64(total) * 100
		}
		stats = append(stats, models.LegacyStat{
			Year: year,
			Loc:  loc,
			Pct:  pct,
		})
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Year < stats[j].Year
	})

	return stats
}

// ProcessSilos finds high-risk knowledge silos.
func ProcessSilos(res ResultExtended, minLOC int) []models.SiloRecord {
	var silos []models.SiloRecord

	for path, owners := range res.FileOwners {
		total := 0
		var maxOwner string
		maxLoc := 0
		for email, loc := range owners {
			total += loc
			if loc > maxLoc {
				maxLoc = loc
				maxOwner = email
			}
		}

		if total < minLOC {
			continue
		}

		ownership := (float64(maxLoc) / float64(total)) * 100.0
		if ownership >= 80.0 {
			silos = append(silos, models.SiloRecord{
				Path:      path,
				LOC:       total,
				Owner:     maxOwner,
				Ownership: ownership,
			})
		}
	}

	sort.Slice(silos, func(i, j int) bool {
		return silos[i].LOC > silos[j].LOC
	})

	if len(silos) > 15 {
		silos = silos[:15]
	}

	return silos
}
