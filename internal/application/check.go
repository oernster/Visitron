package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"visitron/internal/domain"
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

// Log is where a check says what it did (NFR-OBS-001): its start, each
// website's outcome and its end with the time it took.
type Log interface {
	Printf(format string, args ...any)
}

// Check is one pass over every website.
type Check struct {
	store     Store
	releases  Releases
	pageLoads PageLoads
	secrets   Secrets
	clock     Clock
	log       Log
}

// NewCheck builds the check service.
func NewCheck(store Store, releases Releases, pageLoads PageLoads, secrets Secrets, clock Clock, log Log) *Check {
	return &Check{store: store, releases: releases, pageLoads: pageLoads, secrets: secrets, clock: clock, log: log}
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
		c.log.Printf("check could not start: %s", failed.Failures[0])
		return failed, errors.Join(err, c.record(now, failed))
	}
	c.log.Printf("check started: %d websites", len(sites))
	var out Outcome
	read := map[string]bool{}
	for i, w := range sites {
		progress(i, len(sites), w.Address.URL())
		before := len(out.Failures)
		for _, repo := range w.Repos {
			key := strings.ToLower(repo.String())
			if read[key] || !out.RateLimitedUntil.IsZero() {
				continue
			}
			read[key] = true
			c.readRepo(ctx, DayOf(now), repo, &out)
		}
		c.log.Printf("%s: %s", w.Address.URL(), siteOutcome(len(w.Repos), out.Failures[before:], out.RateLimitedUntil))
	}
	progress(len(sites), len(sites), "")
	before := len(out.Failures)
	c.readPageLoads(ctx, now, &out)
	c.log.Printf("page loads: %s", loadsOutcome(out.NoKey, out.Failures[before:]))
	verdict := "succeeded"
	if !out.Succeeded() {
		verdict = "failed"
	}
	c.log.Printf("check %s in %s", verdict, c.clock.Now().Sub(now).Round(time.Millisecond))
	return out, c.record(now, out)
}

// siteOutcome words one website's part of a check for the log.
func siteOutcome(repos int, failures []string, limited time.Time) string {
	switch {
	case len(failures) > 0:
		return strings.Join(failures, "; ")
	case !limited.IsZero():
		return ErrRateLimited.Error()
	case repos == 0:
		return "no repositories chosen"
	default:
		return fmt.Sprintf("%d repositories read", repos)
	}
}

// loadsOutcome words the page-load read for the log.
func loadsOutcome(noKey bool, failures []string) string {
	switch {
	case len(failures) > 0:
		return strings.Join(failures, "; ")
	case noKey:
		return "not read; GoatCounter is not set up"
	default:
		return "read"
	}
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
	site, key, err := goatCounter(c.store, c.secrets)
	if errors.Is(err, ErrNoGoatCounter) {
		out.NoKey = true
		return
	}
	if err != nil {
		out.Failures = append(out.Failures, err.Error())
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
