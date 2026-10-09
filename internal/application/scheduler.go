package application

import (
	"context"
	"errors"
	"sync"
	"time"
)

// FailureRetry is how long the scheduler waits after a failed check before
// trying again on its own, so an outage does not mean a check every minute.
const FailureRetry = 30 * time.Minute

// ErrBusy refuses a check while one is running (FR-033).
var ErrBusy = errors.New("a check is already running")

// Scheduler decides when a check runs and lets only one run at a time.
type Scheduler struct {
	check *Check
	store Store
	clock Clock

	mu      sync.Mutex
	running bool
	resume  time.Time
}

// NewScheduler builds the scheduler.
func NewScheduler(check *Check, store Store, clock Clock) *Scheduler {
	return &Scheduler{check: check, store: store, clock: clock}
}

// Running reports whether a check is under way.
func (s *Scheduler) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// Due reports whether a check should start now (FR-030, FR-031): never while
// GitHub's rate limit is waiting to reset; otherwise when no check has run,
// when the interval has passed since the last success or when FailureRetry
// has passed since a later failure.
func (s *Scheduler) Due() (bool, error) {
	now := s.clock.Now()
	s.mu.Lock()
	resume := s.resume
	s.mu.Unlock()
	if !resume.IsZero() {
		return !now.Before(resume), nil
	}
	rec, err := s.store.CheckRecord()
	if err != nil {
		return false, err
	}
	prefs, err := Preferred(s.store)
	if err != nil {
		return false, err
	}
	if rec.LastSuccess.IsZero() && rec.LastFailure.IsZero() {
		return true, nil
	}
	interval := time.Duration(prefs.IntervalHours) * time.Hour
	if rec.LastFailure.After(rec.LastSuccess) {
		return now.Sub(rec.LastFailure) >= FailureRetry, nil
	}
	return now.Sub(rec.LastSuccess) >= interval, nil
}

// Tick runs a check when one is due. The infrastructure calls it every
// minute, at start and on resume from sleep, which is what gives FR-031 its
// one minute.
func (s *Scheduler) Tick(ctx context.Context, progress Progress) (Outcome, bool, error) {
	due, err := s.Due()
	if err != nil || !due {
		return Outcome{}, false, err
	}
	out, err := s.Refresh(ctx, progress)
	if errors.Is(err, ErrBusy) {
		return Outcome{}, false, nil
	}
	return out, err == nil, err
}

// Refresh runs a check now unless one is running (FR-032, FR-033).
func (s *Scheduler) Refresh(ctx context.Context, progress Progress) (Outcome, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return Outcome{}, ErrBusy
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()
	out, err := s.check.Run(ctx, progress)
	s.mu.Lock()
	s.resume = out.RateLimitedUntil
	s.mu.Unlock()
	return out, err
}
