package application

import (
	"github.com/oernster/visitron/internal/domain"
)

// SiteFigures is one row of the website list (FR-041).
type SiteFigures struct {
	Website Website
	// PageLoads and Downloads are over the chosen period. Downloads come
	// from the history, which starts at the first check.
	PageLoads int
	Downloads int
	// TotalDownloads is every counted download to date.
	TotalDownloads int
	// SinceLastCheck is the rise between the last two kept snapshots.
	SinceLastCheck int
}

// Detail is the selected website's figures (FR-042).
type Detail struct {
	Website        Website
	Totals         domain.Totals
	Files          []domain.ReleaseFile
	DailyPageLoads []domain.DayCount
	DailyDownloads []domain.DayCount
}

// Figures is the figures service.
type Figures struct {
	store Store
	clock Clock
}

// NewFigures builds the figures service.
func NewFigures(store Store, clock Clock) *Figures { return &Figures{store: store, clock: clock} }

// Overview answers every website's row over period.
func (f *Figures) Overview(period domain.Period) ([]SiteFigures, error) {
	sites, err := f.store.Websites()
	if err != nil {
		return nil, err
	}
	loads, err := f.dailyLoads(sites, period)
	if err != nil {
		return nil, err
	}
	rows := make([]SiteFigures, 0, len(sites))
	for _, w := range sites {
		files, rises, err := f.downloads(w, period)
		if err != nil {
			return nil, err
		}
		row := SiteFigures{Website: w, TotalDownloads: domain.Total(files).All}
		for _, d := range loads[w.Address] {
			row.PageLoads += d.Count
		}
		for _, d := range rises {
			row.Downloads += d.Count
		}
		if n := len(rises); n > 0 {
			row.SinceLastCheck = rises[n-1].Count
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// Detail answers one website's figures over period, with a value for every
// day so a chart has no gaps.
func (f *Figures) Detail(id int64, period domain.Period) (Detail, error) {
	sites, err := f.store.Websites()
	if err != nil {
		return Detail{}, err
	}
	for _, w := range sites {
		if w.ID != id {
			continue
		}
		loads, err := f.dailyLoads(sites, period)
		if err != nil {
			return Detail{}, err
		}
		files, rises, err := f.downloads(w, period)
		if err != nil {
			return Detail{}, err
		}
		days := DaysFrom(DaysBack(f.clock.Now(), int(period)), f.clock.Now())
		return Detail{
			Website:        w,
			Totals:         domain.Total(files),
			Files:          files,
			DailyPageLoads: everyDay(days, loads[w.Address]),
			DailyDownloads: everyDay(days, rises),
		}, nil
	}
	return Detail{}, ErrNoSuchSite
}

// CountedSince answers the day the downloads over period are counted from
// when Visitron's own history starts after the period does (Amendment 9).
// GitHub keeps only running totals, so a day's rise needs a snapshot on the
// day before; the period is covered whole only when one is held on the day
// before its first. ok is false when it is; also when nothing is held yet.
func (f *Figures) CountedSince(period domain.Period) (since domain.Day, ok bool, err error) {
	sites, err := f.store.Websites()
	if err != nil {
		return domain.Day{}, false, err
	}
	before := DaysBack(f.clock.Now(), int(period)+1)
	for _, w := range sites {
		for _, repo := range w.Repos {
			snaps, err := f.store.Snapshots(repo, before)
			if err != nil {
				return domain.Day{}, false, err
			}
			if len(snaps) > 0 && (!ok || snaps[0].Day.Before(since)) {
				since, ok = snaps[0].Day, true
			}
		}
	}
	if !ok || since == before {
		return domain.Day{}, false, nil
	}
	return since, true, nil
}

// dailyLoads sums the page loads held for period by owning website (FR-020).
func (f *Figures) dailyLoads(sites []Website, period domain.Period) (map[domain.Address][]domain.DayCount, error) {
	now := f.clock.Now()
	held, err := f.store.PageLoads(DaysBack(now, int(period)), DayOf(now))
	if err != nil {
		return nil, err
	}
	addrs := make([]domain.Address, len(sites))
	for i, w := range sites {
		addrs[i] = w.Address
	}
	byDay := map[domain.Address]map[domain.Day]int{}
	for _, pd := range held {
		owner, ok := domain.Owner(pd.Path, addrs)
		if !ok {
			continue
		}
		if byDay[owner] == nil {
			byDay[owner] = map[domain.Day]int{}
		}
		byDay[owner][pd.Day] += pd.Count
	}
	out := map[domain.Address][]domain.DayCount{}
	for addr, days := range byDay {
		for day, n := range days {
			out[addr] = append(out[addr], domain.DayCount{Day: day, Count: n})
		}
	}
	return out, nil
}

// downloads answers a website's latest release files plus its daily rises in
// period, summed over its chosen repos, in day order. One more day than the
// period is read so the first day of the period has a day to rise from.
func (f *Figures) downloads(w Website, period domain.Period) ([]domain.ReleaseFile, []domain.DayCount, error) {
	now := f.clock.Now()
	first := DaysBack(now, int(period))
	before := DaysBack(now, int(period)+1)
	var files []domain.ReleaseFile
	byDay := map[domain.Day]int{}
	for _, repo := range w.Repos {
		latest, err := f.store.LatestFiles(repo)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, latest...)
		snaps, err := f.store.Snapshots(repo, before)
		if err != nil {
			return nil, nil, err
		}
		for _, d := range domain.DailyRises(snaps) {
			if !d.Day.Before(first) {
				byDay[d.Day] += d.Count
			}
		}
	}
	var rises []domain.DayCount
	for _, day := range DaysFrom(first, now) {
		if n, ok := byDay[day]; ok {
			rises = append(rises, domain.DayCount{Day: day, Count: n})
		}
	}
	return files, rises, nil
}

// everyDay lays counts out over days, zero where a day has none.
func everyDay(days []domain.Day, counts []domain.DayCount) []domain.DayCount {
	have := map[domain.Day]int{}
	for _, c := range counts {
		have[c.Day] += c.Count
	}
	out := make([]domain.DayCount, len(days))
	for i, d := range days {
		out[i] = domain.DayCount{Day: d, Count: have[d]}
	}
	return out
}
