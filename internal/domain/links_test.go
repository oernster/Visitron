package domain

import (
	"reflect"
	"testing"
)

func TestSiteLinksStayOnSite(t *testing.T) {
	t.Parallel()
	site, _ := Normalise("example.org")
	body := `<link href="site.css?v=1"><script src='site.js?v=52470ecff2'></script>
	<a href="download.html#get">Get</a> <a href="download.html">Again</a>
	<a href="https://github.com/someone/Widget">Source</a>
	<a HREF="https://EXAMPLE.org/why.html">Why</a> <a href="mailto:x@y.z">Mail</a>
	<a href="//other.com/x.html">Other</a> <a href="#top">Top</a>`
	want := []string{
		"https://example.org/site.css",
		"https://example.org/site.js",
		"https://example.org/download.html",
		"https://EXAMPLE.org/why.html",
	}
	if got := SiteLinks(site, "https://example.org/", body); !reflect.DeepEqual(got, want) {
		t.Errorf("SiteLinks = %v; want %v", got, want)
	}
}

func TestSiteLinksStayUnderPath(t *testing.T) {
	t.Parallel()
	site, _ := Normalise("example.com/App/")
	body := `<a href="features.html">F</a> <a href="/">Hub</a> <a href="/Tracker/">L</a>`
	want := []string{"https://example.com/App/features.html"}
	if got := SiteLinks(site, "https://example.com/App/", body); !reflect.DeepEqual(got, want) {
		t.Errorf("SiteLinks = %v; want %v", got, want)
	}
	if got := SiteLinks(site, "%zz", body); got != nil {
		t.Errorf("an unreadable page URL gave %v", got)
	}
	root := `<a href="https://example.com">Root</a>`
	hub, _ := Normalise("example.com")
	if got := SiteLinks(hub, "https://example.com/x.html", root); !reflect.DeepEqual(got, []string{"https://example.com"}) {
		t.Errorf("root link = %v", got)
	}
}
