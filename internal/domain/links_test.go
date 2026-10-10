package domain

import (
	"reflect"
	"testing"
)

func TestSiteLinksStayOnSite(t *testing.T) {
	t.Parallel()
	site, _ := Normalise("symdiary.com")
	body := `<link href="site.css?v=1"><script src='site.js?v=52470ecff2'></script>
	<a href="download.html#get">Get</a> <a href="download.html">Again</a>
	<a href="https://github.com/someone/SymDiary">Source</a>
	<a HREF="https://SYMDIARY.com/why.html">Why</a> <a href="mailto:x@y.z">Mail</a>
	<a href="//other.com/x.html">Other</a> <a href="#top">Top</a>`
	want := []string{
		"https://symdiary.com/site.css",
		"https://symdiary.com/site.js",
		"https://symdiary.com/download.html",
		"https://SYMDIARY.com/why.html",
	}
	if got := SiteLinks(site, "https://symdiary.com/", body); !reflect.DeepEqual(got, want) {
		t.Errorf("SiteLinks = %v; want %v", got, want)
	}
}

func TestSiteLinksStayUnderPath(t *testing.T) {
	t.Parallel()
	site, _ := Normalise("ernster.dev/WhatDay/")
	body := `<a href="features.html">F</a> <a href="/">Hub</a> <a href="/Locus/">L</a>`
	want := []string{"https://ernster.dev/WhatDay/features.html"}
	if got := SiteLinks(site, "https://ernster.dev/WhatDay/", body); !reflect.DeepEqual(got, want) {
		t.Errorf("SiteLinks = %v; want %v", got, want)
	}
	if got := SiteLinks(site, "%zz", body); got != nil {
		t.Errorf("an unreadable page URL gave %v", got)
	}
	root := `<a href="https://ernster.dev">Root</a>`
	hub, _ := Normalise("ernster.dev")
	if got := SiteLinks(hub, "https://ernster.dev/x.html", root); !reflect.DeepEqual(got, []string{"https://ernster.dev"}) {
		t.Errorf("root link = %v", got)
	}
}
