package application

import (
	"context"
	"errors"
	"testing"
)

// fakeSource answers a fixed release or fails.
type fakeSource struct {
	release Release
	err     error
	asked   int
}

func (f *fakeSource) Latest(context.Context) (Release, error) {
	f.asked++
	return f.release, f.err
}

var published = Release{
	Tag:  "v1.2.0",
	Page: "https://github.com/someone/Visitron/releases/tag/v1.2.0",
	Assets: []Asset{
		{Name: "Visitron.dmg", Address: "https://github.com/dmg"},
		{Name: "VisitronSetup.EXE", Address: "https://github.com/exe"},
	},
}

func updatesFixture(running, platform string) (*Updates, *fakeSource, *fakeStore) {
	source := &fakeSource{release: published}
	store := newStore()
	return NewUpdates(source, store, running, platform), source, store
}

func TestAnUpdateIsOfferedWithTheWindowsFile(t *testing.T) {
	t.Parallel()
	u, _, _ := updatesFixture("1.1.0", "windows")
	status, err := u.CheckAutomatically(context.Background())
	want := UpdateStatus{Outcome: UpdateAvailable, Running: "1.1.0", Latest: "1.2.0", Address: "https://github.com/exe"}
	if err != nil || status != want {
		t.Fatalf("status = %+v, %v; want %+v", status, err, want)
	}
}

func TestAPlatformWithNoFileIsSentToThePage(t *testing.T) {
	t.Parallel()
	u, source, _ := updatesFixture("1.1.0", "linux")
	if got := u.CheckManually(context.Background()).Address; got != published.Page {
		t.Errorf("address = %q, want the page", got)
	}
	source.release.Assets = nil
	u.ending = platformFiles["windows"]
	if got := u.CheckManually(context.Background()).Address; got != published.Page {
		t.Errorf("a release with no Windows file: address = %q, want the page", got)
	}
}

func TestTheSameOrAnOlderReleaseIsCurrent(t *testing.T) {
	t.Parallel()
	for _, running := range []string{"1.2.0", "1.3.0"} {
		u, _, _ := updatesFixture(running, "windows")
		if got := u.CheckManually(context.Background()).Outcome; got != UpdateCurrent {
			t.Errorf("running %s: outcome %s", running, got)
		}
	}
}

func TestABuildFromSourceIsUncomparable(t *testing.T) {
	t.Parallel()
	u, _, _ := updatesFixture("0.0.0-dev", "windows")
	if got := u.CheckManually(context.Background()).Outcome; got != UpdateUncomparable {
		t.Errorf("outcome %s", got)
	}
}

func TestAnUnreadableReleaseIsUnreachable(t *testing.T) {
	t.Parallel()
	u, source, _ := updatesFixture("1.1.0", "windows")
	source.err = errPlanted
	if got := u.CheckManually(context.Background()); got.Outcome != UpdateUnreachable || got.Address != "" {
		t.Errorf("status %+v", got)
	}
}

func TestNoPublishedReleaseIsItsOwnAnswer(t *testing.T) {
	t.Parallel()
	u, source, _ := updatesFixture("1.1.0", "windows")
	source.err = ErrNoRelease
	if got := u.CheckManually(context.Background()); got.Outcome != UpdateNone || got.Address != "" {
		t.Errorf("status %+v", got)
	}
}

func TestABuildWithNoReleaseSourceSaysSo(t *testing.T) {
	t.Parallel()
	u, source, _ := updatesFixture("1.1.0", "windows")
	source.err = ErrNoReleaseSource
	if got := u.CheckManually(context.Background()); got.Outcome != UpdateNoSource || got.Address != "" {
		t.Errorf("status %+v; want no source named, nothing offered", got)
	}
}

func TestASkippedReleaseIsNotOfferedUnasked(t *testing.T) {
	t.Parallel()
	u, _, store := updatesFixture("1.1.0", "windows")
	if err := u.Skip("1.2.0"); err != nil {
		t.Fatal(err)
	}
	if store.prefs.SkippedUpdate != "1.2.0" || !store.prefs.UpdateCheck {
		t.Fatalf("prefs %+v: the skip lost the other settings", store.prefs)
	}
	if got, _ := u.CheckAutomatically(context.Background()); got.Outcome != UpdateSkipped {
		t.Errorf("automatic: %s", got.Outcome)
	}
	if got := u.CheckManually(context.Background()); got.Outcome != UpdateAvailable {
		t.Errorf("manual ignores the skip: %s", got.Outcome)
	}
	if err := u.Skip("1.1.5"); err != nil {
		t.Fatal(err)
	}
	if got, _ := u.CheckAutomatically(context.Background()); got.Outcome != UpdateAvailable {
		t.Errorf("a different skipped version still offers: %s", got.Outcome)
	}
}

func TestTheSwitchTurnsTheAutomaticCheckOff(t *testing.T) {
	t.Parallel()
	u, source, store := updatesFixture("1.1.0", "windows")
	off := DefaultPreferences
	off.UpdateCheck = false
	store.prefs = &off
	got, err := u.CheckAutomatically(context.Background())
	if err != nil || got.Outcome != UpdateOff || source.asked != 0 {
		t.Fatalf("status %+v, %v after %d requests", got, err, source.asked)
	}
	if u.CheckManually(context.Background()).Outcome != UpdateAvailable {
		t.Error("a check asked for must still run with the switch off")
	}
}

func TestUpdateStoreFaults(t *testing.T) {
	t.Parallel()
	u, _, store := updatesFixture("1.1.0", "windows")
	store.failOn = "Preferences"
	if _, err := u.CheckAutomatically(context.Background()); !errors.Is(err, errPlanted) {
		t.Errorf("automatic: %v", err)
	}
	if err := u.Skip("1.2.0"); !errors.Is(err, errPlanted) {
		t.Errorf("skip: %v", err)
	}
}
