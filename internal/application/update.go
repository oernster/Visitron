package application

// The update check (FR-075, Amendment 5), ported from Bridge Talk: Visitron's
// latest published release compared with the running one.

import (
	"context"
	"errors"
	"strings"

	"visitron/internal/domain"
)

// UpdateOutcome says what one update check found. It is named rather than left
// as a pair of flags because the window words each one differently and shows
// only an offer unasked.
type UpdateOutcome string

const (
	// UpdateOff means the automatic check is turned off in Settings, so
	// nothing was asked.
	UpdateOff UpdateOutcome = "off"
	// UpdateNone means no release has been published yet (Amendment 6).
	UpdateNone UpdateOutcome = "none"
	// UpdateUnreachable means the release could not be read.
	UpdateUnreachable UpdateOutcome = "unreachable"
	// UpdateNoSource means this build names no repository to read releases
	// from, as a build outside a git checkout does not (Amendment 15).
	UpdateNoSource UpdateOutcome = "nosource"
	// UpdateUncomparable means the running version is not dotted whole
	// numbers, as a build from source is not.
	UpdateUncomparable UpdateOutcome = "uncomparable"
	// UpdateCurrent means the release is not newer than the running version.
	UpdateCurrent UpdateOutcome = "current"
	// UpdateSkipped means the release is newer and is the one the owner
	// chose to skip.
	UpdateSkipped UpdateOutcome = "skipped"
	// UpdateAvailable means the release is newer and is offered.
	UpdateAvailable UpdateOutcome = "available"
)

// UpdateStatus is the answer to one update check. Latest is the release's
// version, empty where it could not be read; Address is what Download hands
// the browser, empty unless the release was read.
type UpdateStatus struct {
	Outcome UpdateOutcome
	Running string
	Latest  string
	Address string
}

// platformFiles names, for each platform Visitron ships on, the ending of the
// file a release carries for it. Visitron ships on Windows alone; a platform
// not named is sent to the release's page.
var platformFiles = map[string]string{"windows": ".exe"}

// Updates is the update check service.
type Updates struct {
	source  ReleaseSource
	store   Store
	running string
	ending  string
}

// NewUpdates builds the check over its source, the store holding the skipped
// version, the running version and the platform it runs on, named as Go names
// it.
func NewUpdates(source ReleaseSource, store Store, running, platform string) *Updates {
	return &Updates{source: source, store: store, running: running, ending: platformFiles[platform]}
}

// CheckAutomatically is the check the window runs unasked: nothing at all
// when it is turned off in Settings; otherwise the skipped release reported as
// skipped rather than offered.
func (u *Updates) CheckAutomatically(ctx context.Context) (UpdateStatus, error) {
	prefs, err := Preferred(u.store)
	if err != nil {
		return UpdateStatus{}, err
	}
	if !prefs.UpdateCheck {
		return UpdateStatus{Outcome: UpdateOff, Running: u.running}, nil
	}
	return u.check(ctx, prefs.SkippedUpdate), nil
}

// CheckManually is the check the owner asks for: it offers a skipped release
// too and reports every outcome.
func (u *Updates) CheckManually(ctx context.Context) UpdateStatus { return u.check(ctx, "") }

// Skip keeps version as the release not to offer again.
func (u *Updates) Skip(version string) error {
	p, err := Preferred(u.store)
	if err != nil {
		return err
	}
	p.SkippedUpdate = version
	return u.store.SavePreferences(p)
}

func (u *Updates) check(ctx context.Context, skipped string) UpdateStatus {
	status := UpdateStatus{Outcome: UpdateUnreachable, Running: u.running}
	release, err := u.source.Latest(ctx)
	if errors.Is(err, ErrNoRelease) {
		status.Outcome = UpdateNone
		return status
	}
	if errors.Is(err, ErrNoReleaseSource) {
		status.Outcome = UpdateNoSource
		return status
	}
	if err != nil {
		return status
	}
	status.Latest = domain.ReleaseVersion(release.Tag)
	status.Address = u.download(release)
	newer, comparable := domain.Newer(status.Latest, u.running)
	switch {
	case !comparable:
		status.Outcome = UpdateUncomparable
	case !newer:
		status.Outcome = UpdateCurrent
	case status.Latest == skipped:
		status.Outcome = UpdateSkipped
	default:
		status.Outcome = UpdateAvailable
	}
	return status
}

// download answers the release's file for this platform, matched by its
// ending without regard to case; the release's page where it carries none.
func (u *Updates) download(release Release) string {
	if u.ending != "" {
		for _, asset := range release.Assets {
			if strings.HasSuffix(strings.ToLower(asset.Name), u.ending) {
				return asset.Address
			}
		}
	}
	return release.Page
}
