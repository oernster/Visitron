package main

import (
	"errors"
	"sort"

	"visitron/internal/application"
	"visitron/internal/domain"
)

// Statistics answers one website's Statistics dialog (FR-045 to FR-049). A
// country read that fails does not fail the dialog: the downloads still
// stand; the reason takes the countries' place (FR-049).
func (a *App) Statistics(id int64) (stats StatisticsDTO, err error) {
	defer guard(&err)
	prefs, err := application.Preferred(a.services.Store)
	if err != nil {
		return StatisticsDTO{}, err
	}
	d, err := a.services.Figures.Detail(id, prefs.Period)
	if err != nil {
		return StatisticsDTO{}, err
	}
	stats = StatisticsDTO{
		ID: id, URL: d.Website.Address.URL(), Period: int(prefs.Period),
		ByPlatform: platformCounts(d.Totals.ByPlatform),
		ByRepo:     named(d.Totals.ByRepo), ByRelease: named(d.Totals.ByRelease),
		Countries: []CountryDTO{},
	}
	countries, err := a.services.Countries.Of(a.ctx, id, prefs.Period)
	switch {
	case errors.Is(err, application.ErrNoGoatCounter):
		stats.NoGoatCounter = true
	case err != nil:
		stats.CountriesProblem = err.Error()
	default:
		for _, c := range countries {
			stats.Countries = append(stats.Countries, CountryDTO{Code: c.Code, Name: c.Name, Count: c.Count})
		}
	}
	return stats, nil
}

// platformCounts lists the platforms in the domain's order. A platform with no
// counted downloads is left out, as a .dmg brought to none by the owner's own
// downloads would otherwise stand as a row of 0 (Amendment 22).
func platformCounts(counts map[domain.Platform]int) []NamedCountDTO {
	out := []NamedCountDTO{}
	for _, p := range domain.Platforms {
		if n := counts[p]; n > 0 {
			out = append(out, NamedCountDTO{Name: string(p), Count: n})
		}
	}
	return out
}

// named sorts totals largest first, then by name, for a stable table; a
// total of none is left out, for the same reason as platformCounts.
func named(counts map[string]int) []NamedCountDTO {
	out := make([]NamedCountDTO, 0, len(counts))
	for name, n := range counts {
		if n > 0 {
			out = append(out, NamedCountDTO{Name: name, Count: n})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	return out
}
