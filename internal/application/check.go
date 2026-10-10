package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/oernster/visitron/internal/domain"
)

// Progress is told, before each website, how far a check has got (FR-033).
type Progress func(done, total int, site string)

// Outcome is what one check found out.
type Outcome struct {
	Failures []string
	// RateLimitedUntil is GitHub's reset time when its rate limit stopped the
	// check (FR-035); zero otherwise.
	RateLimitedUntil time.Time
	// NoKey is true when GoatCounter is not set up: no key, no site to use it
	// with or neither (FR-036, Amendment 11).
	NoKey bool
}

// Succeeded reports whether the check read everything it asked for.
func (o Outcome) Succeeded() bool { return len(o.Failures) == 0 && o.RateLimitedUntil.IsZero() }

// Check is one pass over every website.
type Check struct {
	store     Store
	releases  Releases
	pageLoads PageLoads
	secrets   Secrets
	clock     Clock
}

// NewCheck builds the check service.
func NewCheck(store Store, releases Releases, pageLoads PageLoads, secrets Secrets, clock Clock) *Check {
	return &Check{store: store, releases: releases, pageLoads: pageLoads, secrets: secrets, clock: clock}
}

// Run reads every chosen repo's release files from GitHub, then a year of
// page loads from GoatCounter, saving each as it comes. Whatever fails keeps
// the figures already held (FR-034); the outcome is recorded either way.
func (c *Check) Run(ctx context.Context, progress Progress) (Outcome, error) {
	now := c.clock.Now()
	sites, err := c.store.Websites()
	if err != nil {
		// A check that could not start is still a failed check, so the window
		// warns of it like any other (Amendment 3).
		failed := Outcome{Failures: []string{fmt.Sprintf("reading the websites: %v", err)}}
		return failed, errors.Join(err, c.record(now, failed))
	}
	var out Outcome
	read := map[string]bool{}
	for i, w := range sites {
		progress(i, len(sites), w.Address.URL())
		for _, repo := range w.Repos {
			key := strings.ToLower(repo.String())
			if read[key] || !out.RateLimitedUntil.IsZero() {
				continue
			}
			read[key] = true
			c.readRepo(ctx, DayOf(now), repo, &out)
		}
	}
	progress(len(sites), len(sites), "")
	c.readPageLoads(ctx, now, &out)
	return out, c.record(now, out)
}

func (c *Check) readRepo(ctx context.Context, day domain.Day, repo domain.Repo, out *Outcome) {
	files, reset, err := c.releases.Files(ctx, repo)
	switch {
	case errors.Is(err, ErrRateLimited):
		out.RateLimitedUntil = reset
	case err != nil:
		out.Failures = append(out.Failures, fmt.Sprintf("GitHub, %s: %v", repo, err))
	default:
		if err := c.store.SaveFiles(day, repo, files); err != nil {
			out.Failures = append(out.Failures, fmt.Sprintf("saving %s: %v", repo, err))
		}
	}
}

func (c *Check) readPageLoads(ctx context.Context, now time.Time, out *Outcome) {
	key, err := c.secrets.Get(GoatCounterKey)
	if err != nil {
		out.Failures = append(out.Failures, fmt.Sprintf("reading the GoatCounter key: %v", err))
		return
	}
	prefs, err := Preferred(c.store)
	if err != nil {
		out.Failures = append(out.Failures, fmt.Sprintf("reading the GoatCounter account name: %v", err))
		return
	}
	site := prefs.Site()
	if key == "" || site.IsZero() {
		out.NoKey = true
		return
	}
	first, last := DaysBack(now, int(domain.Year)), DayOf(now)
	loads, err := c.pageLoads.Daily(ctx, site, key, first, last)
	if err == nil {
		err = c.store.SavePageLoads(first, last, loads)
	}
	if err != nil {
		out.Failures = append(out.Failures, fmt.Sprintf("GoatCounter: %v", err))
	}
}

// record keeps the time of this check as a success or a failure (FR-044).
// A missing key is not a failure: downloads still worked (FR-036).
func (c *Check) record(now time.Time, out Outcome) error {
	rec, err := c.store.CheckRecord()
	if err != nil {
		return err
	}
	if out.Succeeded() {
		rec.LastSuccess = now
	} else {
		rec.LastFailure = now
		rec.Failure = describe(out)
	}
	return c.store.SaveCheckRecord(rec)
}

func describe(out Outcome) string {
	parts := append([]string{}, out.Failures...)
	if !out.RateLimitedUntil.IsZero() {
		parts = append(parts, fmt.Sprintf("%v; the check resumes after %s",
			ErrRateLimited, out.RateLimitedUntil.Format(time.Kitchen)))
	}
	return strings.Join(parts, "\n")
}
