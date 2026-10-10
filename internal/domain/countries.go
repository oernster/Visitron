package domain

import "sort"

// CountryCount is one country's visitors to a website over a period, as
// GoatCounter counts them by location (FR-047). Code is GoatCounter's
// location code, such as GB; Name is how it names the country.
type CountryCount struct {
	Code  string
	Name  string
	Count int
}

// SortCountries orders countries largest first, then by name, so the table
// reads the same on every opening.
func SortCountries(countries []CountryCount) {
	sort.SliceStable(countries, func(i, j int) bool {
		if countries[i].Count != countries[j].Count {
			return countries[i].Count > countries[j].Count
		}
		return countries[i].Name < countries[j].Name
	})
}
