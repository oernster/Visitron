package main

// The update check's surface (FR-075, Amendment 5), ported from Bridge Talk:
// the check, the download and the skip. The page never holds an address: the
// facade keeps the release it last offered and Download and Skip act on it.

import (
	"errors"

	"github.com/oernster/visitron/internal/application"
)

// errNothingOffered refuses a Download or a Skip asked for before any check
// has offered a release.
var errNothingOffered = errors.New("no update has been offered to act on")

// CheckForUpdates runs one update check. An automatic one asks nothing when
// the check is off in Settings and leaves out the release the owner skipped;
// one asked for from About offers it anyway. Wails runs a bound call off the
// window's thread, so the wait of up to five seconds never holds the window.
func (a *App) CheckForUpdates(manual bool) (update UpdateDTO, err error) {
	defer guard(&err)
	var status application.UpdateStatus
	if manual {
		status = a.services.Updates.CheckManually(a.ctx)
	} else if status, err = a.services.Updates.CheckAutomatically(a.ctx); err != nil {
		return UpdateDTO{}, err
	}
	if status.Outcome == application.UpdateAvailable {
		a.mu.Lock()
		a.offered = status
		a.mu.Unlock()
	}
	return UpdateDTO{Outcome: string(status.Outcome), Running: status.Running, Latest: status.Latest}, nil
}

// DownloadUpdate hands the offered release's Windows file to the browser, its
// page where it carries none.
func (a *App) DownloadUpdate() (err error) {
	defer guard(&err)
	offered, err := a.offer()
	if err != nil {
		return err
	}
	if a.opener == nil {
		return errNoOpener
	}
	a.opener.Open(offered.Address)
	return nil
}

// SkipUpdate keeps the offered release's version, so an automatic check does
// not offer it again.
func (a *App) SkipUpdate() (err error) {
	defer guard(&err)
	offered, err := a.offer()
	if err != nil {
		return err
	}
	return a.services.Updates.Skip(offered.Latest)
}

// offer answers the release last offered, refusing where none has been.
func (a *App) offer() (application.UpdateStatus, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.offered.Outcome != application.UpdateAvailable {
		return application.UpdateStatus{}, errNothingOffered
	}
	return a.offered, nil
}
