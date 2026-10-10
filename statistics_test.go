package main

import (
	"errors"
	"reflect"
	"testing"

	"visitron/internal/application"
	"visitron/internal/domain"
)

// statisticsRig is a rig holding example.org after one check, with
// GoatCounter set up when setUp is true.
func statisticsRig(t *testing.T, setUp bool) (*rig, int64) {
	t.Helper()
	r := newRig(t, nil)
	id, err := r.app.SaveWebsite(0, "example.org", []string{widgetRepo.String()})
	if err != nil {
		t.Fatal(err)
	}
	r.app.tickOnce()
	if setUp {
		if problem, err := r.app.SaveGoatCounterSite("someone"); problem != "" || err != nil {
			t.Fatalf("SaveGoatCounterSite = %q, %v", problem, err)
		}
		r.secrets.values[application.GoatCounterKey] = "k"
	}
	r.loads.paths = []application.SitePath{{Path: "example.org/", ID: 7}}
	r.loads.countries = []domain.CountryCount{
		{Code: "FR", Name: "France", Count: 2},
		{Code: "GB", Name: "United Kingdom", Count: 5},
	}
	return r, id
}

func TestStatisticsCarryTheDownloadTablesAndTheCountries(t *testing.T) {
	r, id := statisticsRig(t, true)
	s, err := r.app.Statistics(id)
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != id || s.URL != "https://example.org/" || s.Period != int(domain.DefaultPeriod) {
		t.Fatalf("statistics = %+v", s)
	}
	if len(s.ByPlatform) != 2 || s.ByPlatform[0].Name != string(domain.Windows) || s.ByPlatform[1].Name != string(domain.MacOS) {
		t.Fatalf("platforms are not in the domain's order: %+v", s.ByPlatform)
	}
	if len(s.ByRepo) != 1 || len(s.ByRelease) != 1 {
		t.Fatalf("repositories %v, releases %v", s.ByRepo, s.ByRelease)
	}
	want := []CountryDTO{{"GB", "United Kingdom", 5}, {"FR", "France", 2}}
	if !reflect.DeepEqual(s.Countries, want) || s.NoGoatCounter || s.CountriesProblem != "" {
		t.Fatalf("countries %v, no GoatCounter %v, problem %q; want %v", s.Countries, s.NoGoatCounter,
			s.CountriesProblem, want)
	}
}

func TestStatisticsSayGoatCounterIsNotSetUpAndKeepTheDownloads(t *testing.T) {
	r, id := statisticsRig(t, false)
	s, err := r.app.Statistics(id)
	if err != nil || !s.NoGoatCounter || s.Countries == nil || len(s.Countries) != 0 || len(s.ByPlatform) == 0 {
		t.Fatalf("statistics = %+v, %v", s, err)
	}
}

func TestStatisticsNameAFailedCountryReadAndKeepTheDownloads(t *testing.T) {
	r, id := statisticsRig(t, true)
	r.loads.err = errPlanted
	s, err := r.app.Statistics(id)
	if err != nil || s.CountriesProblem == "" || s.NoGoatCounter || len(s.ByPlatform) == 0 {
		t.Fatalf("statistics = %+v, %v", s, err)
	}
}

func TestStatisticsRefuseAWebsiteNoLongerRecorded(t *testing.T) {
	r, _ := statisticsRig(t, true)
	if _, err := r.app.Statistics(99); !errors.Is(err, application.ErrNoSuchSite) {
		t.Fatalf("err = %v; want ErrNoSuchSite", err)
	}
}

func TestNamedSortsLargestFirstThenByName(t *testing.T) {
	got := named(map[string]int{"b": 2, "a": 2, "c": 5})
	want := []NamedCountDTO{{"c", 5}, {"a", 2}, {"b", 2}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("named = %v, want %v", got, want)
	}
}

func TestNamedLeavesOutWhatHasNoDownloads(t *testing.T) {
	got := named(map[string]int{"v1.0.0": 0, "v1.1.0": 3})
	if len(got) != 1 || got[0] != (NamedCountDTO{"v1.1.0", 3}) {
		t.Fatalf("named = %v, want only v1.1.0", got)
	}
}

func TestReleasesListTheNewestVersionFirstAndLeaveOutNone(t *testing.T) {
	got := releases(map[string]int{
		"someone/Widget/v1.1.1": 9, "someone/Widget/v1.10.1": 1, "someone/Widget/v1.2.0": 4, "someone/Widget/v0.9.0": 0,
	})
	want := []NamedCountDTO{{"someone/Widget/v1.10.1", 1}, {"someone/Widget/v1.2.0", 4}, {"someone/Widget/v1.1.1", 9}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("releases = %v, want %v", got, want)
	}
}

func TestPlatformCountsKeepTheDomainOrderAndLeaveOutNone(t *testing.T) {
	got := platformCounts(map[domain.Platform]int{
		domain.Other: 2, domain.MacOS: 0, domain.Linux: 1, domain.Windows: 4,
	})
	want := []NamedCountDTO{{string(domain.Windows), 4}, {string(domain.Linux), 1}, {string(domain.Other), 2}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("platformCounts = %v, want %v", got, want)
	}
}
