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
	countries, err := a.services.Countries.Of(a.ctx, id, prefs.Period)
	return StatisticsDTO{
		ID: id, URL: d.Website.Address.URL(),
		ByPlatform: platformCounts(d.Totals.ByPlatform),
		ByRepo:     named(d.Totals.ByRepo), ByRelease: releases(d.Totals.ByRelease),
		Visitors: visitors(prefs.Period, countries, err),
	}, nil
}

// WorldVisitors answers the World visitors dialog: every website's visitors by
// country over the chosen period (FR-092). A read that fails or is not set up
// is said in the countries' place rather than refused (FR-096, FR-097).
func (a *App) WorldVisitors() (out VisitorsDTO, err error) {
	defer guard(&err)
	prefs, err := application.Preferred(a.services.Store)
	if err != nil {
		return VisitorsDTO{}, err
	}
	countries, err := a.services.Countries.All(a.ctx, prefs.Period)
	return visitors(prefs.Period, countries, err), nil
}

// visitors turns a country read into what a dialog shows: the countries, else
// why there are none to show.
func visitors(period domain.Period, countries []domain.CountryCount, err error) VisitorsDTO {
	out := VisitorsDTO{Period: int(period), Countries: []CountryDTO{}}
	switch {
	case errors.Is(err, application.ErrNoGoatCounter):
		out.NoGoatCounter = true
	case err != nil:
		out.Problem = err.Error()
	default:
		for _, c := range countries {
			out.Countries = append(out.Countries, CountryDTO{Code: c.Code, Name: c.Name, Count: c.Count})
		}
	}
	return out
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

// named sorts totals largest first, then by name, for a stable table.
func named(counts map[string]int) []NamedCountDTO {
	out := someDownloads(counts)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// releases lists release totals by repository, newest version first
// (Amendment 24), each named by its version rather than its whole key.
func releases(counts map[string]int) []NamedCountDTO {
	out := someDownloads(counts)
	sort.Slice(out, func(i, j int) bool { return domain.ReleaseBefore(out[i].Name, out[j].Name) })
	keys := make([]string, len(out))
	for i, r := range out {
		keys[i] = r.Name
	}
	for i, label := range domain.ReleaseLabels(keys) {
		out[i].Name = label
	}
	return out
}

// someDownloads lists the totals that are not none, for the same reason as
// platformCounts.
func someDownloads(counts map[string]int) []NamedCountDTO {
	out := make([]NamedCountDTO, 0, len(counts))
	for name, n := range counts {
		if n > 0 {
			out = append(out, NamedCountDTO{Name: name, Count: n})
		}
	}
	return out
}
