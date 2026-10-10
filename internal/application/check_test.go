package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"visitron/internal/domain"
)

func checkFixture() (*fakeStore, *fakeReleases, *fakePageLoads, *fakeSecrets, *fakeClock) {
	store := newStore()
	sym, _ := domain.Normalise("symdiary.com")
	hub, _ := domain.Normalise("ernster.dev")
	_, _ = store.AddWebsite(Website{Address: sym, Repos: []domain.Repo{symRepo}})
	_, _ = store.AddWebsite(Website{Address: hub, Repos: []domain.Repo{{Owner: "SOMEONE", Name: "symdiary"}}})
	rel := &fakeReleases{files: map[string][]domain.ReleaseFile{
		"someone/SymDiary": {{Repo: symRepo, Release: "v1", Name: "SymDiary.dmg", Raw: 3}},
	}}
	loads := &fakePageLoads{loads: []PathDay{{Path: "symdiary.com/", Day: domain.Day{Year: 2026, Month: 10, Date: 9}, Count: 4}}}
	secrets := &fakeSecrets{values: map[Secret]string{GoatCounterKey: "k"}}
	prefs := DefaultPreferences
	prefs.GoatCounterSite = "someone"
	store.prefs = &prefs
	return store, rel, loads, secrets, &fakeClock{now: at(2026, 10, 9, 12)}
}

func TestCheckReadsEachRepoOnceAndSaves(t *testing.T) {
	t.Parallel()
	store, rel, loads, secrets, clock := checkFixture()
	var steps []string
	out, err := NewCheck(store, rel, loads, secrets, clock).Run(context.Background(), func(done, total int, site string) {
		steps = append(steps, site)
	})
	if err != nil || !out.Succeeded() || out.NoKey {
		t.Fatalf("outcome %+v err %v", out, err)
	}
	if len(rel.asked) != 1 {
		t.Errorf("GitHub asked %v; the same repo on two sites is read once", rel.asked)
	}
	if len(steps) != 3 || steps[2] != "" {
		t.Errorf("progress steps %q", steps)
	}
	if loads.keyUsed != "k" || loads.siteUsed.Code() != "someone" || len(store.loads) != 1 {
		t.Errorf("page loads not read with the key from the site: %q %q %v",
			loads.keyUsed, loads.siteUsed.Code(), store.loads)
	}
	if store.record.LastSuccess != clock.now {
		t.Errorf("record %+v", store.record)
	}
}

func TestFailureKeepsFigures(t *testing.T) {
	t.Parallel()
	store, rel, loads, secrets, clock := checkFixture()
	check := NewCheck(store, rel, loads, secrets, clock)
	_, _ = check.Run(context.Background(), noProgress)
	clock.now = at(2026, 10, 10, 12)
	rel.failFor = "someone/SymDiary"
	loads.err = errPlanted
	out, err := check.Run(context.Background(), noProgress)
	if err != nil || out.Succeeded() || len(out.Failures) != 2 {
		t.Fatalf("outcome %+v err %v", out, err)
	}
	if latest, _ := store.LatestFiles(symRepo); len(latest) != 1 {
		t.Error("the figures already held were lost")
	}
	if store.record.LastFailure != clock.now || store.record.LastSuccess.IsZero() || store.record.Failure == "" {
		t.Errorf("record %+v", store.record)
	}
}

func TestRateLimitWaitsForReset(t *testing.T) {
	t.Parallel()
	store, rel, loads, secrets, clock := checkFixture()
	other := domain.Repo{Owner: "someone", Name: "Other"}
	o, _ := domain.Normalise("other.example.com")
	_, _ = store.AddWebsite(Website{Address: o, Repos: []domain.Repo{other}})
	store.sites[0], store.sites[2] = store.sites[2], store.sites[0]
	rel.limitFor = "someone/Other"
	rel.reset = at(2026, 10, 9, 13)
	out, _ := NewCheck(store, rel, loads, secrets, clock).Run(context.Background(), noProgress)
	if out.RateLimitedUntil != rel.reset || len(rel.asked) != 1 {
		t.Errorf("outcome %+v asked %v; GitHub must not be asked again", out, rel.asked)
	}
	if !strings.Contains(store.record.Failure, "resumes after") {
		t.Errorf("failure text %q", store.record.Failure)
	}
}

func TestNoKeyDownloadsStillWork(t *testing.T) {
	t.Parallel()
	store, rel, loads, _, clock := checkFixture()
	out, err := NewCheck(store, rel, loads, &fakeSecrets{}, clock).Run(context.Background(), noProgress)
	if err != nil || !out.NoKey || !out.Succeeded() || loads.keyUsed != "" {
		t.Errorf("outcome %+v err %v", out, err)
	}
	if latest, _ := store.LatestFiles(symRepo); len(latest) != 1 {
		t.Error("downloads were not read without a key")
	}
}

func TestAKeyWithNoSiteIsNotSetUp(t *testing.T) {
	t.Parallel()
	store, rel, loads, secrets, clock := checkFixture()
	store.prefs = nil
	out, err := NewCheck(store, rel, loads, secrets, clock).Run(context.Background(), noProgress)
	if err != nil || !out.NoKey || !out.Succeeded() || loads.keyUsed != "" {
		t.Errorf("no site: outcome %+v err %v key used %q; want not set up, GoatCounter not asked",
			out, err, loads.keyUsed)
	}
	store.failOn = "Preferences"
	out, _ = NewCheck(store, rel, loads, secrets, clock).Run(context.Background(), noProgress)
	if len(out.Failures) == 0 || !strings.Contains(strings.Join(out.Failures, " "), "GoatCounter account name") {
		t.Errorf("a preferences fault is not reported: %v", out.Failures)
	}
}

func TestCheckSurfacesStoreFaults(t *testing.T) {
	t.Parallel()
	store, rel, loads, secrets, clock := checkFixture()
	store.saveErr = errPlanted
	secrets.err = errPlanted
	out, _ := NewCheck(store, rel, loads, secrets, clock).Run(context.Background(), noProgress)
	if len(out.Failures) != 2 {
		t.Errorf("failures %v; want the save and the key", out.Failures)
	}
	for _, op := range []string{"Websites", "CheckRecord", "SavePageLoads"} {
		s, r, l, k, c := checkFixture()
		s.failOn = op
		out, err := NewCheck(s, r, l, k, c).Run(context.Background(), noProgress)
		if op == "SavePageLoads" {
			if len(out.Failures) != 1 {
				t.Errorf("%s: failures %v", op, out.Failures)
			}
			continue
		}
		if !errors.Is(err, errPlanted) {
			t.Errorf("%s: err %v", op, err)
		}
	}
}
