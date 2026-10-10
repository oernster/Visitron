package domain

import "strings"

// Owner reports which site address owns a GoatCounter path: the longest
// address whose key is a prefix of it (FR-020). GoatCounter stores a page as
// host plus path, so "example.com/app/index.html" belongs to
// example.com/app/ rather than to example.com/. The host is compared
// without case; the path with it.
func Owner(path string, sites []Address) (Address, bool) {
	host, rest, _ := strings.Cut(path, pathSep)
	target := Address{Host: strings.ToLower(host), Path: pathSep + rest}
	var best Address
	found := false
	for _, s := range sites {
		if s.Contains(target) && (!found || len(s.Path) > len(best.Path)) {
			best, found = s, true
		}
	}
	return best, found
}
