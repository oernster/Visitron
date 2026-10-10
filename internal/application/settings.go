package application

import (
	"context"

	"visitron/internal/domain"
)

// DefaultPreferences are the settings until the owner changes one.
var DefaultPreferences = Preferences{
	IntervalHours: domain.DefaultIntervalHours,
	Period:        domain.DefaultPeriod,
	UpdateCheck:   true,
}

// Preferred answers the saved preferences, else the defaults.
func Preferred(store Store) (Preferences, error) {
	p, found, err := store.Preferences()
	if err != nil || !found {
		return DefaultPreferences, err
	}
	return p, nil
}

// SettingsView is what the Settings dialog shows (FR-060). The key and token
// appear only as set or not (FR-062).
type SettingsView struct {
	Preferences
	StartWithWindows bool
	GoatCounterSet   bool
	GitHubTokenSet   bool
}

// Settings is the settings service.
type Settings struct {
	store     Store
	secrets   Secrets
	startup   Startup
	releases  Releases
	pageLoads PageLoads
}

// NewSettings builds the settings service.
func NewSettings(store Store, secrets Secrets, startup Startup, releases Releases, pageLoads PageLoads) *Settings {
	return &Settings{store: store, secrets: secrets, startup: startup, releases: releases, pageLoads: pageLoads}
}

// View answers the current settings.
func (s *Settings) View() (SettingsView, error) {
	prefs, err := Preferred(s.store)
	if err != nil {
		return SettingsView{}, err
	}
	on, err := s.startup.Enabled()
	if err != nil {
		return SettingsView{}, err
	}
	key, err := s.secrets.Get(GoatCounterKey)
	if err != nil {
		return SettingsView{}, err
	}
	token, err := s.secrets.Get(GitHubToken)
	if err != nil {
		return SettingsView{}, err
	}
	return SettingsView{Preferences: prefs, StartWithWindows: on, GoatCounterSet: key != "", GitHubTokenSet: token != ""}, nil
}

// SaveInterval keeps a check interval in hours (FR-063).
func (s *Settings) SaveInterval(hours int) error {
	if err := domain.ValidInterval(hours); err != nil {
		return err
	}
	return s.change(func(p *Preferences) { p.IntervalHours = hours })
}

// SavePeriod keeps the period the window reports over (FR-043).
func (s *Settings) SavePeriod(days int) error {
	period, err := domain.ValidPeriod(days)
	if err != nil {
		return err
	}
	return s.change(func(p *Preferences) { p.Period = period })
}

// SaveUpdateCheck turns the update check on or off (FR-075).
func (s *Settings) SaveUpdateCheck(on bool) error {
	return s.change(func(p *Preferences) { p.UpdateCheck = on })
}

// SaveStartWithWindows turns the sign-in start on or off (FR-052).
func (s *Settings) SaveStartWithWindows(on bool) error { return s.startup.SetEnabled(on) }

func (s *Settings) change(edit func(*Preferences)) error {
	p, err := Preferred(s.store)
	if err != nil {
		return err
	}
	edit(&p)
	return s.store.SavePreferences(p)
}

// SaveSecret stores a key or token, then tries it once (FR-061). It is stored
// whether or not it works; the answer is the reason it did not, "" when it
// did.
func (s *Settings) SaveSecret(ctx context.Context, name Secret, value string) (string, error) {
	if err := s.secrets.Set(name, value); err != nil {
		return "", err
	}
	var tried error
	if name == GoatCounterKey {
		prefs, err := Preferred(s.store)
		if err != nil {
			return "", err
		}
		if prefs.Site().IsZero() {
			return NoSiteToTry, nil
		}
		tried = s.pageLoads.Verify(ctx, prefs.Site(), value)
	} else {
		tried = s.releases.Verify(ctx, value)
	}
	if tried != nil {
		return tried.Error(), nil
	}
	return "", nil
}

// NoSiteToTry is why a GoatCounter key saved before any site was set could
// not be tried (Amendment 11).
const NoSiteToTry = "there is no GoatCounter account name to try it on yet; enter it above"

// SaveGoatCounterSite keeps the owner's GoatCounter site, read from a code or
// an address (Amendment 11). A stored key is then tried against it once; the
// answer is why it did not work, "" when it did or when no key is stored.
func (s *Settings) SaveGoatCounterSite(ctx context.Context, text string) (string, error) {
	site, err := domain.ParseGoatCounterSite(text)
	if err != nil {
		return "", err
	}
	if err := s.change(func(p *Preferences) { p.GoatCounterSite = site.Code() }); err != nil {
		return "", err
	}
	key, err := s.secrets.Get(GoatCounterKey)
	if err != nil || key == "" {
		return "", err
	}
	if tried := s.pageLoads.Verify(ctx, site, key); tried != nil {
		return tried.Error(), nil
	}
	return "", nil
}

// Secret answers a stored key or token, "" when it is not set, for the
// Settings dialog to show behind its eye (Amendment 8).
func (s *Settings) Secret(name Secret) (string, error) { return s.secrets.Get(name) }

// RemoveSecret forgets a key or token (Amendment 8).
func (s *Settings) RemoveSecret(name Secret) error { return s.secrets.Delete(name) }
