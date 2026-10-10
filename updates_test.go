package main

import (
	"errors"
	"testing"

	"github.com/oernster/visitron/internal/application"
	"github.com/oernster/visitron/internal/infrastructure/store"
)

func TestAnOfferIsKeptForDownloadAndSkip(t *testing.T) {
	r := newRig(t, nil)
	got, err := r.app.CheckForUpdates(false)
	want := UpdateDTO{Outcome: string(application.UpdateAvailable), Running: "1.1.0", Latest: "1.2.0"}
	if err != nil || got != want {
		t.Fatalf("CheckForUpdates = %+v, %v; want %+v", got, err, want)
	}
	if err := r.app.DownloadUpdate(); err != nil || len(r.window.opened) != 1 || r.window.opened[0] != "https://github.com/setup.exe" {
		t.Fatalf("DownloadUpdate: %v, opened %v", err, r.window.opened)
	}
	if err := r.app.SkipUpdate(); err != nil {
		t.Fatal(err)
	}
	if got, _ := r.app.CheckForUpdates(false); got.Outcome != string(application.UpdateSkipped) {
		t.Errorf("after Skip, an automatic check found %s", got.Outcome)
	}
	if got, _ := r.app.CheckForUpdates(true); got.Outcome != string(application.UpdateAvailable) {
		t.Errorf("a check asked for found %s", got.Outcome)
	}
}

func TestNothingOfferedIsNothingToActOn(t *testing.T) {
	r := newRig(t, nil)
	r.app.services.Updates = application.NewUpdates(&fakePublished{err: errPlanted}, r.store, "1.1.0", "windows")
	if got, err := r.app.CheckForUpdates(true); err != nil || got.Outcome != string(application.UpdateUnreachable) {
		t.Fatalf("CheckForUpdates = %+v, %v", got, err)
	}
	if err := r.app.DownloadUpdate(); !errors.Is(err, errNothingOffered) {
		t.Errorf("DownloadUpdate: %v", err)
	}
	if err := r.app.SkipUpdate(); !errors.Is(err, errNothingOffered) {
		t.Errorf("SkipUpdate: %v", err)
	}
}

func TestAnAutomaticCheckSaysWhyItCouldNotReadTheSettings(t *testing.T) {
	r := newRig(t, nil)
	r.app.services.Updates = application.NewUpdates(published, store.Unavailable{Reason: errPlanted}, "1.1.0", "windows")
	if _, err := r.app.CheckForUpdates(false); !errors.Is(err, errPlanted) {
		t.Errorf("err = %v", err)
	}
}

func TestDownloadWithoutABrowserIsRefused(t *testing.T) {
	r := newRig(t, nil)
	if _, err := r.app.CheckForUpdates(true); err != nil {
		t.Fatal(err)
	}
	r.app.opener = nil
	if err := r.app.DownloadUpdate(); !errors.Is(err, errNoOpener) {
		t.Errorf("err = %v", err)
	}
}
