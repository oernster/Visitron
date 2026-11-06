package application

import (
	"context"
	"strings"
	"testing"

	"visitron/internal/domain"
)

// pressRig is one website with one repo, driven the way the app drives it:
// presses of Refresh, automatic checks and a fresh launch over the same data,
// which is what a reboot amounts to.
type pressRig struct {
	store *fakeStore
	rel   *fakeReleases
	clock *fakeClock
	repo  domain.Repo
}

func newPressRig() *pressRig {
	store := newStore()
	repo := domain.Repo{Owner: "someone", Name: "App"}
	site, _ := domain.Normalise("example.com")
	_, _ = store.AddWebsite(Website{Address: site, Repos: []domain.Repo{repo}})
	prefs := DefaultPreferences
	store.prefs = &prefs
	return &pressRig{
		store: store,
		rel:   &fakeReleases{files: map[string][]domain.ReleaseFile{}},
		clock: &fakeClock{now: at(2026, 11, 6, 10)},
		repo:  repo,
	}
}

func (r *pressRig) setup(raw int) []domain.ReleaseFile {
	return []domain.ReleaseFile{{Repo: r.repo, Release: "v1", Name: "AppSetup.exe", Raw: raw}}
}

// github sets what GitHub answers for the repo from now on.
func (r *pressRig) github(raw int) { r.rel.files[r.repo.String()] = r.setup(raw) }

// launch starts Visitron: a new check and scheduler over the same store.
func (r *pressRig) launch() *Scheduler {
	check := NewCheck(r.store, r.rel, &fakePageLoads{}, &fakeSecrets{}, r.clock, &fakeLog{})
	return NewScheduler(check, r.store, r.clock)
}

func (r *pressRig) press(t *testing.T, s *Scheduler) Outcome {
	t.Helper()
	out, err := s.Refresh(context.Background(), noProgress)
	if err != nil {
		t.Fatalf("press: %v", err)
	}
	return out
}

func (r *pressRig) since(t *testing.T) int {
	t.Helper()
	rows, err := NewFigures(r.store, r.clock).Overview(domain.Week)
	if err != nil || len(rows) != 1 {
		t.Fatalf("overview %+v err %v", rows, err)
	}
	return rows[0].SinceLastCheck
}

// The defect this pins: after a reboot the column showed every download
// since the last day Visitron ran (510 over the 5th's 377, so +133) rather
// than since the press before.
func TestSinceLastCheckIsTheRiseBetweenTwoPresses(t *testing.T) {
	t.Parallel()
	r := newPressRig()
	_ = r.store.SaveFiles(domain.Day{Year: 2026, Month: 11, Date: 5}, r.repo, r.setup(377))
	s := r.launch()
	r.github(500)
	r.press(t, s)
	if got := r.since(t); got != 0 {
		t.Errorf("after the first press: %d; want 0, nothing to rise from", got)
	}
	r.clock.now = at(2026, 11, 6, 11)
	r.github(503)
	r.press(t, s)
	if got := r.since(t); got != 3 {
		t.Errorf("after the second press: %d; want 3", got)
	}
	r.clock.now = at(2026, 11, 6, 12)
	r.github(508)
	r.store.record = CheckRecord{}
	if _, ran, err := s.Tick(context.Background(), noProgress); !ran || err != nil {
		t.Fatalf("automatic check: ran %v err %v", ran, err)
	}
	if got := r.since(t); got != 3 {
		t.Errorf("after an automatic check: %d; want 3, only a press moves it", got)
	}
	s = r.launch()
	r.clock.now = at(2026, 11, 6, 15)
	r.github(510)
	r.press(t, s)
	if got := r.since(t); got != 7 {
		t.Errorf("after a reboot and a press: %d; want 7, 510 over the 503 pressed before", got)
	}
}

func TestAPressThatCannotReadARepoKeepsItsChange(t *testing.T) {
	t.Parallel()
	r := newPressRig()
	s := r.launch()
	r.github(500)
	r.press(t, s)
	r.github(503)
	r.press(t, s)
	r.github(520)
	r.rel.failFor = r.repo.String()
	if out := r.press(t, s); out.Succeeded() {
		t.Fatal("a press GitHub refused reported success")
	}
	if got := r.since(t); got != 3 {
		t.Errorf("after a failed press: %d; want 3, the repo was not read", got)
	}
}

func TestAPressSaysWhenItsReadingCannotBeKept(t *testing.T) {
	t.Parallel()
	r := newPressRig()
	r.github(500)
	r.store.failOn = "SaveRefreshReading"
	out := r.press(t, r.launch())
	if len(out.Failures) != 1 || !strings.Contains(out.Failures[0], "refresh reading") {
		t.Errorf("failures %q; want the refresh reading named", out.Failures)
	}
}

func TestAPressWhoseSnapshotFailsTakesNoReading(t *testing.T) {
	t.Parallel()
	r := newPressRig()
	r.github(500)
	r.store.saveErr = errPlanted
	r.press(t, r.launch())
	if _, later, _ := r.store.RefreshReadings(r.repo); later != nil {
		t.Errorf("reading %v kept although the snapshot was not", later)
	}
}
