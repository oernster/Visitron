package application

import (
	"errors"
	"testing"

	"github.com/oernster/visitron/internal/domain"
)

func figuresFixture() (*Figures, *fakeStore, *fakeClock, int64, int64) {
	store := newStore()
	clock := &fakeClock{now: at(2026, 10, 9, 12)}
	hub, _ := domain.Normalise("ernster.dev")
	wd, _ := domain.Normalise("ernster.dev/WhatDay/")
	hubID, _ := store.AddWebsite(Website{Address: hub})
	wdRepo := domain.Repo{Owner: "oernster", Name: "WhatDay"}
	wdID, _ := store.AddWebsite(Website{Address: wd, Repos: []domain.Repo{wdRepo}})
	day := func(d int) domain.Day { return domain.Day{Year: 2026, Month: 10, Date: d} }
	file := func(raw int) []domain.ReleaseFile {
		return []domain.ReleaseFile{{Repo: wdRepo, Release: "v1", Name: "WhatDaySetup.exe", Raw: raw}}
	}
	_ = store.SaveFiles(day(7), wdRepo, file(10))
	_ = store.SaveFiles(day(8), wdRepo, file(12))
	_ = store.SaveFiles(day(9), wdRepo, file(15))
	store.loads = []PathDay{
		{Path: "ernster.dev/index.html", Day: day(8), Count: 3},
		{Path: "ernster.dev/WhatDay/index.html", Day: day(8), Count: 5},
		{Path: "ernster.dev/WhatDay/", Day: day(9), Count: 2},
		{Path: "elsewhere.com/", Day: day(9), Count: 99},
	}
	return NewFigures(store, clock), store, clock, hubID, wdID
}

func TestOverviewSeparatesSubSites(t *testing.T) {
	t.Parallel()
	f, _, _, _, _ := figuresFixture()
	rows, err := f.Overview(domain.Week)
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows %+v err %v", rows, err)
	}
	hub, wd := rows[0], rows[1]
	if hub.PageLoads != 3 || hub.TotalDownloads != 0 {
		t.Errorf("hub %+v", hub)
	}
	if wd.PageLoads != 7 || wd.TotalDownloads != 15 || wd.Downloads != 5 || wd.SinceLastCheck != 3 {
		t.Errorf("WhatDay %+v; want loads 7, total 15, period 5, since last 3", wd)
	}
}

func TestPeriodEdgeHasADayToRiseFrom(t *testing.T) {
	t.Parallel()
	f, _, clock, _, _ := figuresFixture()
	clock.now = at(2026, 10, 10, 12)
	rows, _ := f.Overview(domain.Period(2))
	if rows[1].Downloads != 3 {
		t.Errorf("two-day period downloads = %d; want 3 (the 9th over the 8th)", rows[1].Downloads)
	}
}

func TestCountedSinceNamesAHistoryShorterThanThePeriod(t *testing.T) {
	t.Parallel()
	f, store, clock, _, _ := figuresFixture()
	since, ok, err := f.CountedSince(domain.Week)
	if err != nil || !ok || since != (domain.Day{Year: 2026, Month: 10, Date: 7}) {
		t.Errorf("week: %v %v %v; want the 7th, the first day held", since, ok, err)
	}
	clock.now = at(2026, 10, 10, 12)
	if _, ok, err := f.CountedSince(domain.Period(2)); ok || err != nil {
		t.Errorf("two days from a snapshot on the day before: ok %v err %v; want covered whole", ok, err)
	}
	store.failOn = "Snapshots"
	if _, _, err := f.CountedSince(domain.Week); !errors.Is(err, errPlanted) {
		t.Errorf("snapshot fault: %v", err)
	}
	store.failOn = "Websites"
	if _, _, err := f.CountedSince(domain.Week); !errors.Is(err, errPlanted) {
		t.Errorf("website fault: %v", err)
	}
}

func TestCountedSinceIsSilentBeforeAnyCheck(t *testing.T) {
	t.Parallel()
	f := NewFigures(newStore(), &fakeClock{now: at(2026, 10, 9, 12)})
	if _, ok, err := f.CountedSince(domain.Week); ok || err != nil {
		t.Errorf("no history: ok %v err %v; want nothing named", ok, err)
	}
}

func TestDetailFillsEveryDay(t *testing.T) {
	t.Parallel()
	f, _, _, _, wdID := figuresFixture()
	d, err := f.Detail(wdID, domain.Week)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.DailyPageLoads) != int(domain.Week) || len(d.DailyDownloads) != int(domain.Week) {
		t.Fatalf("days %d and %d; want %d each", len(d.DailyPageLoads), len(d.DailyDownloads), domain.Week)
	}
	last := len(d.DailyDownloads) - 1
	if d.DailyDownloads[last].Count != 3 || d.DailyDownloads[last-1].Count != 2 || d.DailyDownloads[0].Count != 0 {
		t.Errorf("downloads %v", d.DailyDownloads)
	}
	if d.DailyPageLoads[last].Count != 2 || d.Totals.All != 15 || len(d.Files) != 1 {
		t.Errorf("detail %+v", d)
	}
	if _, err := f.Detail(999, domain.Week); !errors.Is(err, ErrNoSuchSite) {
		t.Errorf("unknown id: %v", err)
	}
}

func TestFiguresFaults(t *testing.T) {
	t.Parallel()
	for _, op := range []string{"Websites", "PageLoads", "LatestFiles", "Snapshots"} {
		f, store, _, _, wdID := figuresFixture()
		store.failOn = op
		if _, err := f.Overview(domain.Week); !errors.Is(err, errPlanted) {
			t.Errorf("overview %s: %v", op, err)
		}
		if _, err := f.Detail(wdID, domain.Week); !errors.Is(err, errPlanted) {
			t.Errorf("detail %s: %v", op, err)
		}
	}
}
