package domain

import (
	"errors"
	"reflect"
	"testing"
)

// widgetPages is the text measured on example.org on 2026-10-09 (Appendix
// A, M-3): the home page names the repository twice, site.js once more
// through the API.
const widgetPages = `
<a href="https://github.com/someone/Widget">Source</a>
<a href="https://github.com/someone/Widget/blob/main/LICENSE">GPL</a>
fetch('https://api.github.com/repos/someone/Widget/releases/latest')`

func TestFindsBothForms(t *testing.T) {
	t.Parallel()
	got := FindRepos(widgetPages)
	want := []Repo{{Owner: "someone", Name: "Widget"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindRepos = %v; want %v", got, want)
	}
	api := FindRepos(`https://api.github.com/repos/someone/Player/releases`)
	if !reflect.DeepEqual(api, []Repo{{Owner: "someone", Name: "Player"}}) {
		t.Errorf("api form = %v", api)
	}
}

func TestIgnoresWhatIsNotARepo(t *testing.T) {
	t.Parallel()
	text := `https://github.com/sponsors/someone https://gist.github.com/a/b
	https://github.com/someone https://api.github.com/users/someone/x
	https://notgithub.com/a/b https://someone.github.io/App/
	https://www.github.com/someone/Tracker.git https://github.com/SOMEONE/tracker`
	want := []Repo{{Owner: "someone", Name: "Tracker"}}
	if got := FindRepos(text); !reflect.DeepEqual(got, want) {
		t.Errorf("FindRepos = %v; want %v", got, want)
	}
}

func TestParseRepo(t *testing.T) {
	t.Parallel()
	r, err := ParseRepo(" someone/Widget.git ")
	if err != nil || r != (Repo{Owner: "someone", Name: "Widget"}) {
		t.Fatalf("ParseRepo = %v, %v", r, err)
	}
	for _, bad := range []string{"", "someone", "a/b/c", "a/..", "a b/c", "/x"} {
		if _, err := ParseRepo(bad); !errors.Is(err, ErrRepoForm) {
			t.Errorf("ParseRepo(%q) = %v; want ErrRepoForm", bad, err)
		}
	}
	if !r.Same(Repo{Owner: "SOMEONE", Name: "widget"}) || r.Same(Repo{Owner: "x", Name: "y"}) {
		t.Error("Same must ignore case only")
	}
}

func TestPreTick(t *testing.T) {
	t.Parallel()
	gizmo, _ := Normalise("gizmohub.com")
	app, _ := Normalise("example.com/App/")
	hub, _ := Normalise("example.com")
	many := []Repo{{"someone", "gizmo-hub"}, {"someone", "App"}, {"someone", "Tracker"}}
	cases := []struct {
		site  Address
		found []Repo
		want  []Repo
	}{
		{gizmo, many, []Repo{{"someone", "gizmo-hub"}}},
		{app, many, []Repo{{"someone", "App"}}},
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
