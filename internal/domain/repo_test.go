package domain

import (
	"errors"
	"reflect"
	"testing"
)

// symdiaryPages is the text measured on symdiary.com on 2026-10-09 (Appendix
// A, M-3): the home page names the repository twice, site.js once more
// through the API.
const symdiaryPages = `
<a href="https://github.com/someone/SymDiary">Source</a>
<a href="https://github.com/someone/SymDiary/blob/main/LICENSE">GPL</a>
fetch('https://api.github.com/repos/someone/SymDiary/releases/latest')`

func TestFindsBothForms(t *testing.T) {
	t.Parallel()
	got := FindRepos(symdiaryPages)
	want := []Repo{{Owner: "someone", Name: "SymDiary"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindRepos = %v; want %v", got, want)
	}
	api := FindRepos(`https://api.github.com/repos/someone/Stellody/releases`)
	if !reflect.DeepEqual(api, []Repo{{Owner: "someone", Name: "Stellody"}}) {
		t.Errorf("api form = %v", api)
	}
}

func TestIgnoresWhatIsNotARepo(t *testing.T) {
	t.Parallel()
	text := `https://github.com/sponsors/someone https://gist.github.com/a/b
	https://github.com/someone https://api.github.com/users/someone/x
	https://notgithub.com/a/b https://someone.github.io/WhatDay/
	https://www.github.com/someone/Locus.git https://github.com/SOMEONE/locus`
	want := []Repo{{Owner: "someone", Name: "Locus"}}
	if got := FindRepos(text); !reflect.DeepEqual(got, want) {
		t.Errorf("FindRepos = %v; want %v", got, want)
	}
}

func TestParseRepo(t *testing.T) {
	t.Parallel()
	r, err := ParseRepo(" someone/SymDiary.git ")
	if err != nil || r != (Repo{Owner: "someone", Name: "SymDiary"}) {
		t.Fatalf("ParseRepo = %v, %v", r, err)
	}
	for _, bad := range []string{"", "someone", "a/b/c", "a/..", "a b/c", "/x"} {
		if _, err := ParseRepo(bad); !errors.Is(err, ErrRepoForm) {
			t.Errorf("ParseRepo(%q) = %v; want ErrRepoForm", bad, err)
		}
	}
	if !r.Same(Repo{Owner: "SOMEONE", Name: "symdiary"}) || r.Same(Repo{Owner: "x", Name: "y"}) {
		t.Error("Same must ignore case only")
	}
}

func TestPreTick(t *testing.T) {
	t.Parallel()
	snark, _ := Normalise("snarkapi.com")
	whatday, _ := Normalise("ernster.dev/WhatDay/")
	hub, _ := Normalise("ernster.dev")
	many := []Repo{{"someone", "snark-api"}, {"someone", "WhatDay"}, {"someone", "Locus"}}
	cases := []struct {
		site  Address
		found []Repo
		want  []Repo
	}{
		{snark, many, []Repo{{"someone", "snark-api"}}},
		{whatday, many, []Repo{{"someone", "WhatDay"}}},
		{hub, many, nil},
		{hub, many[2:], many[2:]},
		{hub, nil, nil},
	}
	for _, c := range cases {
		if got := PreTicked(c.site, c.found); !reflect.DeepEqual(got, c.want) {
			t.Errorf("PreTicked(%s) = %v; want %v", c.site.Key(), got, c.want)
		}
	}
}
