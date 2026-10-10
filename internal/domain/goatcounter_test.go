package domain

import (
	"errors"
	"testing"
)

func TestGoatCounterSiteReadsEveryWayItIsTyped(t *testing.T) {
	t.Parallel()
	for _, typed := range []string{
		"someone", " Someone ", "someone.goatcounter.com", "https://someone.goatcounter.com",
		"https://SOMEONE.goatcounter.com/", "http://someone.goatcounter.com",
	} {
		site, err := ParseGoatCounterSite(typed)
		if err != nil || site.Code() != "someone" || site.URL() != "https://someone.goatcounter.com" {
			t.Errorf("%q: %q %q %v", typed, site.Code(), site.URL(), err)
		}
	}
	if site, err := ParseGoatCounterSite("my-site2"); err != nil || site.Code() != "my-site2" {
		t.Errorf("hyphen and digit: %q %v", site.Code(), err)
	}
}

func TestGoatCounterSiteRefusesAnythingElse(t *testing.T) {
	t.Parallel()
	for _, typed := range []string{
		"", "  ", "-lead", "has space", "evil.com", "someone.goatcounter.com.evil.com", "https://evil.com/",
		"a_b",
	} {
		if site, err := ParseGoatCounterSite(typed); !errors.Is(err, ErrGoatCounterSite) {
			t.Errorf("%q accepted as %q", typed, site.Code())
		}
	}
	var none GoatCounterSite
	if !none.IsZero() || none.Code() != "" {
		t.Error("the zero site is not empty")
	}
}
