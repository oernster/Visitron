package application

import (
	"context"
	"errors"
	"testing"

	"visitron/internal/domain"
)

// countriesFixture is the check's fixture with GoatCounter holding paths on
// both websites: example.org (id 1) owns two, example.com one.
func countriesFixture() (*fakeStore, *fakePageLoads, *fakeSecrets, *fakeClock) {
	store, _, loads, secrets, clock := checkFixture()
	loads.paths = []SitePath{
		{Path: "example.org/", ID: 11},
		{Path: "example.com/", ID: 31},
		{Path: "example.org/docs/", ID: 12},
	}
	loads.countries = []domain.CountryCount{
		{Code: "FR", Name: "France", Count: 2},
		{Code: "GB", Name: "United Kingdom", Count: 5},
	}
	return store, loads, secrets, clock
}

func TestCountriesReadOnlyTheWebsitesOwnPathsLargestFirst(t *testing.T) {
	t.Parallel()
	store, loads, secrets, clock := countriesFixture()
	got, err := NewCountries(store, loads, secrets, clock).Of(context.Background(), 1, domain.DefaultPeriod)
	if err != nil {
		t.Fatal(err)
	}
	if len(loads.idsAsked) != 2 || loads.idsAsked[0] != 11 || loads.idsAsked[1] != 12 {
		t.Errorf("asked about %v; want example.org's paths 11 and 12 alone", loads.idsAsked)
	}
	if len(got) != 2 || got[0].Code != "GB" || got[1].Code != "FR" {
		t.Errorf("countries = %v; want GB then FR", got)
	}
}

func TestCountriesAreNoneWhereTheWebsiteHadNoVisitedPath(t *testing.T) {
	t.Parallel()
	store, loads, secrets, clock := countriesFixture()
	loads.paths = []SitePath{{Path: "example.com/", ID: 31}}
	got, err := NewCountries(store, loads, secrets, clock).Of(context.Background(), 1, domain.DefaultPeriod)
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("countries = %v, %v; want an empty list", got, err)
	}
	if loads.idsAsked != nil {
		t.Error("GoatCounter was asked about countries with no path to ask about")
	}
}

func TestCountriesSayWhenGoatCounterIsNotSetUp(t *testing.T) {
	t.Parallel()
	store, loads, secrets, clock := countriesFixture()
	delete(secrets.values, GoatCounterKey)
	_, err := NewCountries(store, loads, secrets, clock).Of(context.Background(), 1, domain.DefaultPeriod)
	if !errors.Is(err, ErrNoGoatCounter) {
		t.Fatalf("err = %v; want ErrNoGoatCounter", err)
	}
}

func TestCountriesPassOnEveryFailure(t *testing.T) {
	t.Parallel()
	cases := map[string]func(*fakeStore, *fakePageLoads, *fakeSecrets){
		"websites":  func(s *fakeStore, _ *fakePageLoads, _ *fakeSecrets) { s.failOn = "Websites" },
		"key":       func(_ *fakeStore, _ *fakePageLoads, k *fakeSecrets) { k.err = errPlanted },
		"paths":     func(_ *fakeStore, l *fakePageLoads, _ *fakeSecrets) { l.pathsErr = errPlanted },
		"countries": func(_ *fakeStore, l *fakePageLoads, _ *fakeSecrets) { l.countriesErr = errPlanted },
	}
	for name, plant := range cases {
		store, loads, secrets, clock := countriesFixture()
		plant(store, loads, secrets)
		_, err := NewCountries(store, loads, secrets, clock).Of(context.Background(), 1, domain.DefaultPeriod)
		if !errors.Is(err, errPlanted) {
			t.Errorf("%s: err = %v; want the planted failure", name, err)
		}
	}
}

func TestCountriesRefuseAWebsiteNoLongerRecorded(t *testing.T) {
	t.Parallel()
	store, loads, secrets, clock := countriesFixture()
	_, err := NewCountries(store, loads, secrets, clock).Of(context.Background(), 99, domain.DefaultPeriod)
	if !errors.Is(err, ErrNoSuchSite) {
		t.Fatalf("err = %v; want ErrNoSuchSite", err)
	}
}
