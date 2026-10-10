package domain

import "testing"

func TestSortCountriesPutsTheLargestFirstThenOrdersByName(t *testing.T) {
	t.Parallel()
	got := []CountryCount{
		{Code: "FR", Name: "France", Count: 7},
		{Code: "US", Name: "United States", Count: 12},
		{Code: "CA", Name: "Canada", Count: 7},
	}
	SortCountries(got)
	want := []string{"US", "CA", "FR"}
	for i, code := range want {
		if got[i].Code != code {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}
