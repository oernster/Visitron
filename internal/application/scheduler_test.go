package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func schedulerFixture() (*Scheduler, *fakeStore, *fakeReleases, *fakeClock) {
	store, rel, loads, secrets, clock := checkFixture()
	return NewScheduler(NewCheck(store, rel, loads, secrets, clock), store, clock), store, rel, clock
}

func TestDueAfterInterval(t *testing.T) {
	t.Parallel()
	s, store, _, clock := schedulerFixture()
	if due, _ := s.Due(); !due {
		t.Error("a first check is due at once")
	}
	store.record = CheckRecord{LastSuccess: clock.now}
	clock.now = clock.now.Add(23 * time.Hour)
	if due, _ := s.Due(); due {
		t.Error("due before the interval")
	}
	clock.now = clock.now.Add(time.Hour)
	if due, _ := s.Due(); !due {
		t.Error("not due once the interval passed")
	}
	store.prefs = &Preferences{IntervalHours: 48}
	if due, _ := s.Due(); due {
		t.Error("a saved interval was ignored")
	}
}

func TestFailureRetriesAfterWait(t *testing.T) {
	t.Parallel()
	s, store, _, clock := schedulerFixture()
	store.record = CheckRecord{LastSuccess: clock.now.Add(-time.Hour), LastFailure: clock.now}
	if due, _ := s.Due(); due {
		t.Error("retried at once after a failure")
	}
	clock.now = clock.now.Add(FailureRetry)
	if due, _ := s.Due(); !due {
		t.Error("no retry after FailureRetry")
	}
}

func TestOverdueOnStart(t *testing.T) {
	t.Parallel()
	s, store, _, clock := schedulerFixture()
	store.record = CheckRecord{LastSuccess: clock.now.Add(-25 * time.Hour)}
	out, ran, err := s.Tick(context.Background(), noProgress)
	if !ran || err != nil || !out.Succeeded() {
		t.Errorf("tick at start: ran %v err %v", ran, err)
	}
	if _, ran, _ := s.Tick(context.Background(), noProgress); ran {
		t.Error("ran again straight after a success")
	}
}

func TestOverdueOnResume(t *testing.T) {
	t.Parallel()
	s, store, _, clock := schedulerFixture()
	store.record = CheckRecord{LastSuccess: clock.now}
	clock.now = clock.now.Add(30 * time.Hour)
	if _, ran, _ := s.Tick(context.Background(), noProgress); !ran {
		t.Error("an overdue check did not run on the resume tick")
	}
}

func TestRateLimitResumesAtReset(t *testing.T) {
	t.Parallel()
	s, _, rel, clock := schedulerFixture()
	rel.limitFor = "oernster/SymDiary"
	rel.reset = clock.now.Add(time.Hour)
	_, _ = s.Refresh(context.Background(), noProgress)
	if due, _ := s.Due(); due {
		t.Error("due before GitHub's reset")
	}
	clock.now = rel.reset
	rel.limitFor = ""
	if _, ran, _ := s.Tick(context.Background(), noProgress); !ran {
		t.Error("did not resume at the reset time")
	}
}

func TestRefreshStartsCheck(t *testing.T) {
	t.Parallel()
	s, store, _, clock := schedulerFixture()
	store.record = CheckRecord{LastSuccess: clock.now}
	if _, err := s.Refresh(context.Background(), noProgress); err != nil || s.Running() {
		t.Errorf("refresh: %v running %v", err, s.Running())
	}
	if len(store.record.LastSuccess.String()) == 0 {
		t.Error("no record")
	}
}

func TestNoOverlap(t *testing.T) {
	t.Parallel()
	s, _, _, _ := schedulerFixture()
	inside := make(chan struct{})
	release := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		first := true
		_, _ = s.Refresh(context.Background(), func(int, int, string) {
			if first {
				first = false
				close(inside)
				<-release
			}
		})
	}()
	<-inside
	if !s.Running() {
		t.Error("not marked running")
	}
	if _, err := s.Refresh(context.Background(), noProgress); !errors.Is(err, ErrBusy) {
		t.Errorf("second refresh = %v; want ErrBusy", err)
	}
	if _, ran, err := s.Tick(context.Background(), noProgress); ran || err != nil {
		t.Errorf("tick during a check: ran %v err %v", ran, err)
	}
	close(release)
	wg.Wait()
}

func TestDueSurfacesStoreFaults(t *testing.T) {
	t.Parallel()
	for _, op := range []string{"CheckRecord", "Preferences"} {
		s, store, _, _ := schedulerFixture()
		store.failOn = op
		if _, err := s.Due(); !errors.Is(err, errPlanted) {
			t.Errorf("%s: %v", op, err)
		}
		if _, _, err := s.Tick(context.Background(), noProgress); !errors.Is(err, errPlanted) {
			t.Errorf("%s tick: %v", op, err)
		}
	}
}
