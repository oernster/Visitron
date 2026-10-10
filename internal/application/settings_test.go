package application

import (
	"context"
	"errors"
	"testing"

	"github.com/oernster/visitron/internal/domain"
)

func settingsFixture() (*Settings, *fakeStore, *fakeSecrets, *fakeStartup, *fakeReleases, *fakePageLoads) {
	store, sec, start := newStore(), &fakeSecrets{}, &fakeStartup{on: true}
	rel, loads := &fakeReleases{}, &fakePageLoads{}
	return NewSettings(store, sec, start, rel, loads), store, sec, start, rel, loads
}

func TestDefaultsUntilChanged(t *testing.T) {
	t.Parallel()
	s, _, _, _, _, _ := settingsFixture()
	v, err := s.View()
	if err != nil || v.Preferences != DefaultPreferences || !v.StartWithWindows || v.GoatCounterSet || v.GitHubTokenSet {
		t.Errorf("view %+v err %v", v, err)
	}
}

func TestIntervalRange(t *testing.T) {
	t.Parallel()
	s, store, _, _, _, _ := settingsFixture()
	if err := s.SaveInterval(48); err != nil || store.prefs.IntervalHours != 48 {
		t.Errorf("save 48: %v", err)
	}
	if err := s.SaveInterval(0); !errors.Is(err, domain.ErrInterval) {
		t.Errorf("save 0: %v", err)
	}
}

func TestPeriodRemembered(t *testing.T) {
	t.Parallel()
	s, store, _, _, _, _ := settingsFixture()
	if err := s.SavePeriod(90); err != nil || store.prefs.Period != domain.Quarter {
		t.Errorf("save 90: %v", err)
	}
	if err := s.SavePeriod(14); !errors.Is(err, domain.ErrPeriod) {
		t.Errorf("save 14: %v", err)
	}
	if store.prefs.IntervalHours != domain.DefaultIntervalHours {
		t.Error("saving the period lost the interval default")
	}
}

func TestToggles(t *testing.T) {
	t.Parallel()
	s, store, _, start, _, _ := settingsFixture()
	if err := s.SaveUpdateCheck(false); err != nil || store.prefs.UpdateCheck {
		t.Errorf("update check: %v", err)
	}
	if err := s.SaveStartWithWindows(false); err != nil || start.on {
		t.Errorf("start with Windows: %v", err)
	}
}

func TestKeyVerified(t *testing.T) {
	t.Parallel()
	s, _, sec, _, rel, loads := settingsFixture()
	if why, err := s.SaveSecret(context.Background(), GoatCounterKey, "good"); why != "" || err != nil {
		t.Errorf("good key: %q %v", why, err)
	}
	loads.verifyErr = errPlanted
	rel.verifyErr = errPlanted
	for _, name := range []Secret{GoatCounterKey, GitHubToken} {
		why, err := s.SaveSecret(context.Background(), name, "bad")
		if why == "" || err != nil || sec.values[name] != "bad" {
			t.Errorf("%s: %q %v; a key that fails is still stored", name, why, err)
		}
	}
	v, _ := s.View()
	if !v.GoatCounterSet || !v.GitHubTokenSet {
		t.Errorf("view %+v", v)
	}
	if got, err := s.Secret(GoatCounterKey); got != "bad" || err != nil {
		t.Errorf("Secret = %q, %v; the stored key is answered as it is", got, err)
	}
	if err := s.RemoveSecret(GitHubToken); err != nil || sec.values[GitHubToken] != "" {
		t.Errorf("remove: %v", err)
	}
	sec.err = errPlanted
	if _, err := s.SaveSecret(context.Background(), GitHubToken, "x"); !errors.Is(err, errPlanted) {
		t.Errorf("Credential Manager failure: %v", err)
	}
}

func TestSettingsFaults(t *testing.T) {
	t.Parallel()
	s, store, sec, start, _, _ := settingsFixture()
	store.failOn = "Preferences"
	if _, err := s.View(); !errors.Is(err, errPlanted) {
		t.Errorf("view prefs: %v", err)
	}
	if err := s.SaveUpdateCheck(true); !errors.Is(err, errPlanted) {
		t.Errorf("change: %v", err)
	}
	store.failOn = ""
	start.err = errPlanted
	if _, err := s.View(); !errors.Is(err, errPlanted) {
		t.Errorf("view startup: %v", err)
	}
	start.err = nil
	sec.err = errPlanted
	if _, err := s.View(); !errors.Is(err, errPlanted) {
		t.Errorf("view secrets: %v", err)
	}
}

func TestViewTokenFault(t *testing.T) {
	t.Parallel()
	store := newStore()
	sec := &tokenFaultSecrets{}
	s := NewSettings(store, sec, &fakeStartup{}, &fakeReleases{}, &fakePageLoads{})
	if _, err := s.View(); !errors.Is(err, errPlanted) {
		t.Errorf("token read fault: %v", err)
	}
}

// tokenFaultSecrets fails only when the GitHub token is read.
type tokenFaultSecrets struct{ fakeSecrets }

func (f *tokenFaultSecrets) Get(name Secret) (string, error) {
	if name == GitHubToken {
		return "", errPlanted
	}
	return "", nil
}
