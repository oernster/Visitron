package main

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"visitron/internal/application"
	"visitron/internal/domain"
	"visitron/internal/infrastructure/store"
)

// The facade's fakes, ported from internal/application/fakes_test.go. The
// store is the real one in a temporary directory; only what reaches the
// network or Windows is faked.

var errPlanted = errors.New("planted failure")

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type fakeFetcher struct{ pages map[string]string }

func (f fakeFetcher) Fetch(_ context.Context, url string) (string, error) {
	body, ok := f.pages[url]
	if !ok {
		return "", errors.New("404 for " + url)
	}
	return body, nil
}

type fakeReleases struct {
	files map[string][]domain.ReleaseFile
}

func (f fakeReleases) Files(_ context.Context, repo domain.Repo) ([]domain.ReleaseFile, time.Time, error) {
	return f.files[repo.String()], time.Time{}, nil
}

func (f fakeReleases) Exists(_ context.Context, repo domain.Repo) (bool, error) {
	_, ok := f.files[repo.String()]
	return ok, nil
}

func (fakeReleases) Verify(context.Context, string) error { return nil }

type fakePageLoads struct{}

func (fakePageLoads) Daily(context.Context, domain.GoatCounterSite, string, domain.Day, domain.Day) ([]application.PathDay, error) {
	return nil, nil
}

func (fakePageLoads) Verify(context.Context, domain.GoatCounterSite, string) error { return nil }

type fakeSecrets struct{ values map[application.Secret]string }

func (f *fakeSecrets) Get(name application.Secret) (string, error) { return f.values[name], nil }

func (f *fakeSecrets) Set(name application.Secret, value string) error {
	f.values[name] = value
	return nil
}

func (f *fakeSecrets) Delete(name application.Secret) error {
	delete(f.values, name)
	return nil
}

// fakePublished answers a fixed release of Visitron or fails.
type fakePublished struct {
	release application.Release
	err     error
}

func (f *fakePublished) Latest(context.Context) (application.Release, error) { return f.release, f.err }

// published is the release every rig sees: newer than the rig's 1.1.0.
var published = &fakePublished{release: application.Release{
	Tag:    "v1.2.0",
	Page:   "https://github.com/someone/Visitron/releases/tag/v1.2.0",
	Assets: []application.Asset{{Name: "VisitronSetup.exe", Address: "https://github.com/setup.exe"}},
}}

type fakeStartup struct{ on bool }

func (f *fakeStartup) Enabled() (bool, error)   { return f.on, nil }
func (f *fakeStartup) SetEnabled(on bool) error { f.on = on; return nil }

// recorder keeps everything the facade hands the window: progress events and
// the names of every event, addresses opened, focus, reveals, hides and quits.
type recorder struct {
	mu       sync.Mutex
	events   []ProgressDTO
	names    []string
	opened   []string
	focused  int
	revealed int
	hidden   int
	quits    int
}

func (r *recorder) Emit(name string, data any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.names = append(r.names, name)
	if progress, ok := data.(ProgressDTO); ok {
		r.events = append(r.events, progress)
	}
}

func (r *recorder) Open(address string) { r.opened = append(r.opened, address) }
func (r *recorder) Focus()              { r.focused++ }
func (r *recorder) Reveal()             { r.revealed++ }
func (r *recorder) Hide()               { r.hidden++ }
func (r *recorder) Quit()               { r.quits++ }

func (r *recorder) seen() []ProgressDTO {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ProgressDTO{}, r.events...)
}

// symdiaryRepo is the one repository the fakes know.
var symdiaryRepo = domain.Repo{Owner: "someone", Name: "SymDiary"}

// rig is a facade over the real store with every port faked.
type rig struct {
	app     *App
	store   *store.Store
	window  *recorder
	secrets *fakeSecrets
}

func newRig(t *testing.T, data application.Store) *rig {
	t.Helper()
	var opened *store.Store
	if data == nil {
		var err error
		opened, err = store.Open(filepath.Join(t.TempDir(), "visitron.db"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = opened.Close() })
		data = opened
	}
	clock := fixedClock{now: time.Date(2026, time.October, 9, 20, 0, 0, 0, time.Local)}
	releases := fakeReleases{files: map[string][]domain.ReleaseFile{symdiaryRepo.String(): {
		{Repo: symdiaryRepo, Release: "v1.0.0", Name: "SymDiary-Setup.exe", Raw: 5},
		{Repo: symdiaryRepo, Release: "v1.0.0", Name: "SymDiary.dmg", Raw: 3},
	}}}
	vault := &fakeSecrets{values: map[application.Secret]string{}}
	check := application.NewCheck(data, releases, fakePageLoads{}, vault, clock)
	services := Services{
		Websites:  application.NewWebsites(data, fakeFetcher{}, releases),
		Figures:   application.NewFigures(data, clock),
		Scheduler: application.NewScheduler(check, data, clock),
		Settings:  application.NewSettings(data, vault, &fakeStartup{}, releases, fakePageLoads{}),
		Updates:   application.NewUpdates(published, data, "1.1.0", "windows"),
		Store:     data,
	}
	app := newApp(services, "1.2.3", "", func() error { return nil })
	window := &recorder{}
	app.emitter, app.opener, app.focuser, app.control = window, window, window, window
	app.ctx, app.cancel = context.WithCancel(context.Background())
	t.Cleanup(app.cancel)
	return &rig{app: app, store: opened, window: window, secrets: vault}
}
