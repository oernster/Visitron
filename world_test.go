package main

import (
	"reflect"
	"testing"

	"visitron/internal/domain"
)

func TestWorldVisitorsCarryEveryWebsitesCountries(t *testing.T) {
	r, _ := statisticsRig(t, true)
	v, err := r.app.WorldVisitors()
	if err != nil {
		t.Fatal(err)
	}
	want := []CountryDTO{{"GB", "United Kingdom", 5}, {"FR", "France", 2}}
	if v.Period != int(domain.DefaultPeriod) || !reflect.DeepEqual(v.Countries, want) || v.NoGoatCounter || v.Problem != "" {
		t.Fatalf("world visitors = %+v; want %v over the default period", v, want)
	}
}

func TestWorldVisitorsSayGoatCounterIsNotSetUp(t *testing.T) {
	r, _ := statisticsRig(t, false)
	v, err := r.app.WorldVisitors()
	if err != nil || !v.NoGoatCounter || v.Countries == nil || len(v.Countries) != 0 {
		t.Fatalf("world visitors = %+v, %v", v, err)
	}
}

func TestWorldVisitorsNameAFailedRead(t *testing.T) {
	r, _ := statisticsRig(t, true)
	r.loads.err = errPlanted
	v, err := r.app.WorldVisitors()
	if err != nil || v.Problem == "" || v.NoGoatCounter {
		t.Fatalf("world visitors = %+v, %v", v, err)
	}
}
