package application

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/oernster/visitron/internal/domain"
)

var errPlanted = errors.New("planted failure")

// fakeClock is a settable clock in London time.
type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

func london() *time.Location {
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		panic(err)
	}
	return loc
}

func at(y, m, d, h int) time.Time { return time.Date(y, time.Month(m), d, h, 0, 0, 0, london()) }

// fakeFetcher answers pages by URL; a URL it does not hold is an error.
type fakeFetcher struct {
	pages   map[string]string
	fetched []string
}

func (f *fakeFetcher) Fetch(_ context.Context, url string) (string, error) {
	f.fetched = append(f.fetched, url)
	body, ok := f.pages[url]
	if !ok {
		return "", errors.New("404 for " + url)
	}
	return body, nil
}

// fakeReleases answers release files by repo.
type fakeReleases struct {
	files     map[string][]domain.ReleaseFile
	failFor   string
	limitFor  string
	reset     time.Time
	existsErr error
	verifyErr error
	asked     []string
}

func (f *fakeReleases) Files(_ context.Context, repo domain.Repo) ([]domain.ReleaseFile, time.Time, error) {
	f.asked = append(f.asked, repo.String())
	switch repo.String() {
	case f.limitFor:
		return nil, f.reset, ErrRateLimited
	case f.failFor:
		return nil, time.Time{}, errPlanted
	}
	return f.files[repo.String()], time.Time{}, nil
}

func (f *fakeReleases) Exists(_ context.Context, repo domain.Repo) (bool, error) {
	if f.existsErr != nil {
		return false, f.existsErr
	}
	_, ok := f.files[repo.String()]
	return ok, nil
}

func (f *fakeReleases) Verify(context.Context, string) error { return f.verifyErr }

// fakePageLoads answers fixed page loads.
type fakePageLoads struct {
	loads     []PathDay
	err       error
	verifyErr error
	keyUsed   string
	siteUsed  domain.GoatCounterSite
}

func (f *fakePageLoads) Daily(_ context.Context, site domain.GoatCounterSite, key string, _, _ domain.Day) ([]PathDay, error) {
	f.keyUsed, f.siteUsed = key, site
	return f.loads, f.err
}

func (f *fakePageLoads) Verify(_ context.Context, site domain.GoatCounterSite, _ string) error {
	f.siteUsed = site
	return f.verifyErr
}

// fakeSecrets is an in-memory Credential Manager.
type fakeSecrets struct {
	values map[Secret]string
	err    error
}

func (f *fakeSecrets) Get(name Secret) (string, error) { return f.values[name], f.err }

func (f *fakeSecrets) Set(name Secret, value string) error {
	if f.err != nil {
		return f.err
	}
	if f.values == nil {
		f.values = map[Secret]string{}
	}
	f.values[name] = value
	return nil
}

func (f *fakeSecrets) Delete(name Secret) error { delete(f.values, name); return f.err }

// fakeStartup is an in-memory Run entry.
type fakeStartup struct {
	on  bool
	err error
}

func (f *fakeStartup) Enabled() (bool, error) { return f.on, f.err }
func (f *fakeStartup) SetEnabled(on bool) error {
	f.on = on
	return f.err
}

// fakeStore keeps everything in memory with the store's semantics.
type fakeStore struct {
	sites   []Website
	nextID  int64
	seeded  bool
	files   map[string]map[domain.Day][]domain.ReleaseFile
	loads   []PathDay
	prefs   *Preferences
	record  CheckRecord
	failOn  string
	saveErr error
}

func newStore() *fakeStore {
	return &fakeStore{files: map[string]map[domain.Day][]domain.ReleaseFile{}}
}

func (s *fakeStore) fail(op string) error {
	if s.failOn == op {
		return errPlanted
	}
	return nil
}

func (s *fakeStore) Websites() ([]Website, error) {
	return append([]Website{}, s.sites...), s.fail("Websites")
}

func (s *fakeStore) AddWebsite(w Website) (int64, error) {
	if err := s.fail("AddWebsite"); err != nil {
		return 0, err
	}
	s.nextID++
	w.ID = s.nextID
	s.sites = append(s.sites, w)
	return w.ID, nil
}

func (s *fakeStore) UpdateWebsite(w Website) error {
	for i := range s.sites {
		if s.sites[i].ID == w.ID {
			s.sites[i] = w
			return nil
		}
	}
	return ErrNoSuchSite
}

func (s *fakeStore) DeleteWebsite(id int64) error {
	for i := range s.sites {
		if s.sites[i].ID == id {
			s.sites = append(s.sites[:i], s.sites[i+1:]...)
			return nil
		}
	}
	return ErrNoSuchSite
}

func (s *fakeStore) Seeded() (bool, error) { return s.seeded, s.fail("Seeded") }
func (s *fakeStore) MarkSeeded() error {
	s.seeded = true
	return nil
}

func (s *fakeStore) SaveFiles(day domain.Day, repo domain.Repo, files []domain.ReleaseFile) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	key := strings.ToLower(repo.String())
	if s.files[key] == nil {
		s.files[key] = map[domain.Day][]domain.ReleaseFile{}
	}
	s.files[key][day] = files
	return nil
}

func (s *fakeStore) days(repo domain.Repo) []domain.Day {
	var days []domain.Day
	for d := range s.files[strings.ToLower(repo.String())] {
		days = append(days, d)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
	return days
}

func (s *fakeStore) LatestFiles(repo domain.Repo) ([]domain.ReleaseFile, error) {
	if err := s.fail("LatestFiles"); err != nil {
		return nil, err
	}
	days := s.days(repo)
	if len(days) == 0 {
		return nil, nil
	}
	return s.files[strings.ToLower(repo.String())][days[len(days)-1]], nil
}

func (s *fakeStore) Snapshots(repo domain.Repo, first domain.Day) ([]domain.Snapshot, error) {
	if err := s.fail("Snapshots"); err != nil {
		return nil, err
	}
	var snaps []domain.Snapshot
	for _, d := range s.days(repo) {
		if !d.Before(first) {
			snaps = append(snaps, domain.SnapshotOf(d, s.files[strings.ToLower(repo.String())][d]))
		}
	}
	return snaps, nil
}

func (s *fakeStore) SavePageLoads(_, _ domain.Day, loads []PathDay) error {
	s.loads = loads
	return s.fail("SavePageLoads")
}

func (s *fakeStore) PageLoads(first, last domain.Day) ([]PathDay, error) {
	var out []PathDay
	for _, pd := range s.loads {
		if !pd.Day.Before(first) && !last.Before(pd.Day) {
			out = append(out, pd)
		}
	}
	return out, s.fail("PageLoads")
}

func (s *fakeStore) Preferences() (Preferences, bool, error) {
	if s.prefs == nil {
		return Preferences{}, false, s.fail("Preferences")
	}
	return *s.prefs, true, s.fail("Preferences")
}

func (s *fakeStore) SavePreferences(p Preferences) error {
	s.prefs = &p
	return s.fail("SavePreferences")
}

func (s *fakeStore) CheckRecord() (CheckRecord, error) { return s.record, s.fail("CheckRecord") }
func (s *fakeStore) SaveCheckRecord(r CheckRecord) error {
	s.record = r
	return s.fail("SaveCheckRecord")
}

func noProgress(int, int, string) {}
