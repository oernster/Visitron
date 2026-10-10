package domain

import (
	"errors"
	"testing"
)

func TestNormalise(t *testing.T) {
	t.Parallel()
	cases := []struct{ entry, key, url string }{
		{"SymDiary.com", "symdiary.com/", "https://symdiary.com/"},
		{"  https://SYMDIARY.com  ", "symdiary.com/", "https://symdiary.com/"},
		{"http://symdiary.com/download.html?x=1#get", "symdiary.com/", "https://symdiary.com/"},
		{"https://example.com/App", "example.com/App/", "https://example.com/App/"},
		{"example.com/App/index.html", "example.com/App/", "https://example.com/App/"},
		{"https://example.com:443/a/b/", "example.com/a/b/", "https://example.com/a/b/"},
	}
	for _, c := range cases {
		a, err := Normalise(c.entry)
		if err != nil {
			t.Fatalf("Normalise(%q): %v", c.entry, err)
		}
		if a.Key() != c.key || a.URL() != c.url {
			t.Errorf("Normalise(%q) = %q, %q; want %q, %q", c.entry, a.Key(), a.URL(), c.key, c.url)
		}
	}
}

func TestRefusals(t *testing.T) {
	t.Parallel()
	cases := map[string]error{
		"":                ErrEmptyAddress,
		"   ":             ErrEmptyAddress,
		"ftp://x.com":     ErrScheme,
		"symdiary":        ErrHostNotDNS,
		"https://":        ErrNoHost,
		"https://%zz.com": ErrNoHost,
	}
	for entry, want := range cases {
		if _, err := Normalise(entry); !errors.Is(err, want) {
			t.Errorf("Normalise(%q) = %v; want %v", entry, err, want)
		}
	}
}

func TestSegmentsAndLabels(t *testing.T) {
	t.Parallel()
	root := Address{Host: "example.com", Path: "/"}
	sub := Address{Host: "example.com", Path: "/App/"}
	deep := Address{Host: "example.com", Path: "/a/App/"}
	if root.LastSegment() != "" || sub.LastSegment() != "App" || deep.LastSegment() != "App" {
		t.Errorf("LastSegment: %q %q %q", root.LastSegment(), sub.LastSegment(), deep.LastSegment())
	}
	if sub.FirstLabel() != "example" {
		t.Errorf("FirstLabel = %q", sub.FirstLabel())
	}
	if !root.Contains(sub) || sub.Contains(root) {
		t.Error("Contains: root must contain the sub-site and not the reverse")
	}
	if root.Contains(Address{Host: "symdiary.com", Path: "/"}) {
		t.Error("Contains crossed hosts")
	}
}
