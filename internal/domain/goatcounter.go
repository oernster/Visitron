package domain

import (
	"errors"
	"strings"
)

// ErrGoatCounterSite refuses an entry that names no goatcounter.com site
// (Amendment 11).
var ErrGoatCounterSite = errors.New(
	"a GoatCounter site is the code before .goatcounter.com in its address, as in yourname")

// goatCounterHost is the domain every hosted GoatCounter site sits under. The
// site is held to it, so the key is only ever sent to GoatCounter
// (NFR-PRIV-001).
const goatCounterHost = ".goatcounter.com"

// GoatCounterSite is one goatcounter.com site, named by its code: the label
// before .goatcounter.com. Each owner names their own in Settings.
type GoatCounterSite struct{ code string }

// ParseGoatCounterSite reads a code typed alone, with .goatcounter.com after
// it or as the whole address, in any case. The code is letters, digits and
// hyphens, starting with a letter or digit.
func ParseGoatCounterSite(text string) (GoatCounterSite, error) {
	code := strings.ToLower(strings.TrimSpace(text))
	if i := strings.Index(code, schemeMark); i >= 0 {
		code = code[i+len(schemeMark):]
	}
	code = strings.TrimSuffix(code, pathSep)
	code = strings.TrimSuffix(code, goatCounterHost)
	if code == "" || !isCodeStart(code[0]) {
		return GoatCounterSite{}, ErrGoatCounterSite
	}
	for i := 0; i < len(code); i++ {
		if !isCodeStart(code[i]) && code[i] != '-' {
			return GoatCounterSite{}, ErrGoatCounterSite
		}
	}
	return GoatCounterSite{code: code}, nil
}

func isCodeStart(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') }

// Code is the site's code; "" when none is set.
func (s GoatCounterSite) Code() string { return s.code }

// IsZero reports whether no site is set.
func (s GoatCounterSite) IsZero() bool { return s.code == "" }

// URL is the site's address, which its API sits under.
func (s GoatCounterSite) URL() string { return defaultScheme + s.code + goatCounterHost }
