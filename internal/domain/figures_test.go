package domain

import (
	"errors"
	"reflect"
	"testing"
)

func TestLongestPrefixOwns(t *testing.T) {
	t.Parallel()
	hub, _ := Normalise("example.com")
	app, _ := Normalise("example.com/App/")
	sym, _ := Normalise("example.org")
	sites := []Address{hub, app, sym}
	cases := map[string]Address{
		"example.com/index.html":        hub,
		"example.com/":                  hub,
		"example.com":                   hub,
		"example.com/applications.html": hub,
		"example.com/?ref=x":            hub,
		"example.com/App/index.html":    app,
		"example.com/App/":              app,
		"EXAMPLE.COM/App/":              app,
		"example.com/app/":              hub,
		"example.org/download.html":     sym,
	}
	for path, want := range cases {
		if got, ok := Owner(path, sites); !ok || got != want {
			t.Errorf("Owner(%q) = %v, %v; want %v", path, got.Key(), ok, want.Key())
		}
	}
	if _, ok := Owner("elsewhere.com/", sites); ok {
		t.Error("a path on no recorded site must have no owner")
	}
}

func TestSelfDownloadAllowance(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name            string
		raw, self, want int
	}{
		{"Widget.dmg", 5, DefaultSelfDownloads, 5},
		{"Widget.dmg", 1, 1, 0},
		{"Widget.dmg", 0, 1, 0},
		{"Widget.DMG", 5, 1, 4},
		{"Widget.dmg", 5, 3, 2},
		{"Widget.dmg", 5, -2, 5},
		{"WidgetSetup.exe", 4, 1, 4},
		{"widget.flatpak", 0, 1, 0},
	}
	for _, c := range cases {
		if got := Counted(c.name, c.raw, c.self); got != c.want {
			t.Errorf("Counted(%q, %d, %d) = %d; want %d", c.name, c.raw, c.self, got, c.want)
		}
	}
}

func TestSelfDownloadsRange(t *testing.T) {
	t.Parallel()
	for _, ok := range []int{MinSelfDownloads, DefaultSelfDownloads, MaxSelfDownloads} {
		if err := ValidSelfDownloads(ok); err != nil {
			t.Errorf("ValidSelfDownloads(%d) = %v", ok, err)
		}
	}
	for _, bad := range []int{MinSelfDownloads - 1, MaxSelfDownloads + 1} {
		if err := ValidSelfDownloads(bad); !errors.Is(err, ErrSelfDownloads) {
			t.Errorf("ValidSelfDownloads(%d) = %v; want ErrSelfDownloads", bad, err)
		}
	}
}

func TestPlatforms(t *testing.T) {
	t.Parallel()
	cases := map[string]Platform{
		"MixerSetup.exe": Windows, "Mailer.dmg": MacOS,
		"ledger.flatpak": Linux, "notes.txt": Other, "README": Other,
	}
	for name, want := range cases {
		if got := PlatformOf(name); got != want {
			t.Errorf("PlatformOf(%q) = %s; want %s", name, got, want)
		}
	}
}

func TestTotals(t *testing.T) {
	t.Parallel()
	sym := Repo{"someone", "Widget"}
	tr := Repo{"someone", "Gadget"}
	files := []ReleaseFile{
		{sym, "v1.3.0", "Widget.dmg", 3},
		{sym, "v1.3.0", "WidgetSetup.exe", 5},
		{sym, "v1.2.0", "widget.flatpak", 2},
		{tr, "v1.3.0", "GadgetSetup.exe", 1},
	}
	got := Total(files, 1)
	want := Totals{
		All:        10,
		ByRepo:     map[string]int{"someone/Widget": 9, "someone/Gadget": 1},
		ByRelease:  map[string]int{"someone/Widget/v1.3.0": 7, "someone/Widget/v1.2.0": 2, "someone/Gadget/v1.3.0": 1},
		ByPlatform: map[Platform]int{Windows: 6, MacOS: 2, Linux: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Total = %+v; want %+v", got, want)
	}
}

func TestDailyRise(t *testing.T) {
	t.Parallel()
	sym := Repo{"someone", "Widget"}
	file := func(raw int) []ReleaseFile { return []ReleaseFile{{sym, "v1", "WidgetSetup.exe", raw}} }
	oct1, oct3, oct5 := Day{2026, 10, 1}, Day{2026, 10, 3}, Day{2026, 10, 5}
	snaps := []Snapshot{
		SnapshotOf(oct3, file(14), DefaultSelfDownloads),
		SnapshotOf(oct1, file(8), DefaultSelfDownloads),
		SnapshotOf(oct1, file(10), DefaultSelfDownloads),
		SnapshotOf(oct5, file(20), DefaultSelfDownloads),
	}
	want := []DayCount{{oct3, 4}, {oct5, 6}}
	if got := DailyRises(snaps); !reflect.DeepEqual(got, want) {
		t.Errorf("DailyRises = %v; want %v", got, want)
	}
	if got := DailyRises(nil); len(got) != 0 {
		t.Errorf("no snapshots gave %v", got)
	}
}

func TestFallIsNotNegative(t *testing.T) {
	t.Parallel()
	before := Snapshot{Counts: map[string]int{"a": 10, "b": 5, "gone": 7}}
	after := Snapshot{Counts: map[string]int{"a": 4, "b": 8, "new": 3}}
	if got := Rise(before, after); got != 3 {
		t.Errorf("Rise = %d; want 3 (only b rose)", got)
	}
}

func TestDayOrderAndText(t *testing.T) {
	t.Parallel()
	days := []Day{{2025, 12, 31}, {2026, 1, 1}, {2026, 2, 1}, {2026, 2, 2}}
	for i := 1; i < len(days); i++ {
		if !days[i-1].Before(days[i]) || days[i].Before(days[i-1]) {
			t.Errorf("order wrong between %s and %s", days[i-1], days[i])
		}
	}
	if s := (Day{2026, 10, 9}).String(); s != "2026-10-09" {
		t.Errorf("String = %q", s)
	}
}

func TestIntervalRange(t *testing.T) {
	t.Parallel()
	for _, ok := range []int{MinIntervalHours, DefaultIntervalHours, MaxIntervalHours} {
		if err := ValidInterval(ok); err != nil {
			t.Errorf("ValidInterval(%d) = %v", ok, err)
		}
	}
	for _, bad := range []int{MinIntervalHours - 1, MaxIntervalHours + 1} {
		if !errors.Is(ValidInterval(bad), ErrInterval) {
			t.Errorf("ValidInterval(%d) accepted", bad)
		}
	}
}

func TestPeriodsAndTheme(t *testing.T) {
	t.Parallel()
	if p, err := ValidPeriod(int(Quarter)); err != nil || p != Quarter {
		t.Errorf("ValidPeriod(90) = %v, %v", p, err)
	}
	if _, err := ValidPeriod(14); !errors.Is(err, ErrPeriod) {
		t.Errorf("ValidPeriod(14) = %v", err)
	}
	if DefaultPeriod != Month || DefaultTheme != Dark {
		t.Error("defaults changed: the spec says 30 days and dark")
	}
	if Dark.Next() != Light || Light.Next() != Dark {
		t.Error("theme must alternate")
	}
}
