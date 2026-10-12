package main

import "testing"

func TestReleaseRepo(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"https://github.com/someone/Example.git\n": "someone/Example",
		"https://github.com/someone/Example":       "someone/Example",
		"https://github.com/someone/Example/":      "someone/Example",
		"git@github.com:someone/my.repo.git":       "someone/my.repo",
		"https://gitlab.com/someone/Example.git":   "",
		"https://github.com/someone":               "",
		"":                                         "",
	}
	for origin, want := range cases {
		if got := releaseRepo(origin); got != want {
			t.Errorf("releaseRepo(%q) = %q, want %q", origin, got, want)
		}
	}
}
