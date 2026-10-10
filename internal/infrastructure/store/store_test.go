package store

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"visitron/internal/application"
	"visitron/internal/domain"
)

var (
	_ application.Store = (*Store)(nil)
	_ application.Store = Unavailable{}
)

var (
	sym   = domain.Repo{Owner: "someone", Name: "SymDiary"}
	oct8  = domain.Day{Year: 2026, Month: 10, Date: 8}
	oct9  = domain.Day{Year: 2026, Month: 10, Date: 9}
	oct10 = domain.Day{Year: 2026, Month: 10, Date: 10}
)

func open(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "nested", "visitron.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func addr(t *testing.T, entry string) domain.Address {
	t.Helper()
	a, err := domain.Normalise(entry)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func file(name string, raw int) domain.ReleaseFile {
	return domain.ReleaseFile{Repo: sym, Release: "v1", Name: name, Raw: raw}
}

func TestWebsitesRoundTrip(t *testing.T) {
	t.Parallel()
	s := open(t)
	other := domain.Repo{Owner: "someone", Name: "Other"}
	id, err := s.AddWebsite(application.Website{Address: addr(t, "symdiary.com"), Repos: []domain.Repo{sym, other}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddWebsite(application.Website{Address: addr(t, "symdiary.com")}); err == nil {
		t.Error("a second website at the same address was accepted")
	}
	got, err := s.Websites()
	if err != nil || len(got) != 1 || got[0].ID != id || !reflect.DeepEqual(got[0].Repos, []domain.Repo{sym, other}) {
		t.Fatalf("websites %+v err %v", got, err)
	}
	moved := application.Website{ID: id, Address: addr(t, "example.com/App/"), Repos: []domain.Repo{other}}
	if err := s.UpdateWebsite(moved); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Websites()
	if !reflect.DeepEqual(got, []application.Website{moved}) {
		t.Errorf("after update %+v", got)
	}
	for _, err := range []error{s.UpdateWebsite(application.Website{ID: 99}), s.DeleteWebsite(99)} {
		if !errors.Is(err, application.ErrNoSuchSite) {
			t.Errorf("unknown id: %v", err)
		}
	}
}

func TestDeleteRemovesHistory(t *testing.T) {
	t.Parallel()
	s := open(t)
	shared := domain.Repo{Owner: "SOMEONE", Name: "symdiary"}
	a, _ := s.AddWebsite(application.Website{Address: addr(t, "a.example.com"), Repos: []domain.Repo{sym}})
	b, _ := s.AddWebsite(application.Website{Address: addr(t, "b.example.com"), Repos: []domain.Repo{shared}})
	_ = s.SaveFiles(oct9, sym, []domain.ReleaseFile{file("SymDiary.dmg", 3)})
	if err := s.DeleteWebsite(a); err != nil {
		t.Fatal(err)
	}
	if latest, _ := s.LatestFiles(sym); len(latest) != 1 {
		t.Error("history still chosen by another website was removed")
	}
	_ = s.DeleteWebsite(b)
	if latest, _ := s.LatestFiles(sym); len(latest) != 0 {
		t.Error("history of a repo no website chooses was kept")
	}
	if got, _ := s.Websites(); len(got) != 0 {
		t.Errorf("websites left %+v", got)
	}
}

func TestFilesBySnapshot(t *testing.T) {
	t.Parallel()
	s := open(t)
	_ = s.SaveFiles(oct8, sym, []domain.ReleaseFile{file("SymDiary.dmg", 2), file("SymDiarySetup.exe", 5)})
	_ = s.SaveFiles(oct9, sym, []domain.ReleaseFile{file("SymDiary.dmg", 9)})
	_ = s.SaveFiles(oct9, sym, []domain.ReleaseFile{file("SymDiary.dmg", 3), file("SymDiarySetup.exe", 7)})
	latest, err := s.LatestFiles(domain.Repo{Owner: "SOMEONE", Name: "SYMDIARY"})
	if err != nil || len(latest) != 2 || latest[0].Raw != 3 || latest[0].Repo != sym {
		t.Fatalf("latest %+v err %v; a later save on a day replaces it", latest, err)
	}
	// The history keeps GitHub's own counts; nothing is taken off a disk image
	// until it is counted, since how much is a setting (Amendment 19).
	held, err := s.History(sym, oct8)
	if err != nil || len(held) != 2 || held[0].Day != oct8 || held[1].Files[0].Name != "SymDiary.dmg" || held[1].Files[0].Raw != 3 {
		t.Fatalf("history %+v err %v", held, err)
	}
	if held, _ := s.History(sym, oct9); len(held) != 1 {
		t.Errorf("from the 9th: %d days", len(held))
	}
	if none, err := s.LatestFiles(domain.Repo{Owner: "x", Name: "y"}); none != nil || err != nil {
		t.Errorf("unknown repo: %v %v", none, err)
	}
}

func TestSnapshotIsAtomic(t *testing.T) {
	t.Parallel()
	s := open(t)
	_ = s.SaveFiles(oct9, sym, []domain.ReleaseFile{file("SymDiary.dmg", 3)})
	twice := []domain.ReleaseFile{file("SymDiary.dmg", 4), file("SymDiary.dmg", 5)}
	if err := s.SaveFiles(oct9, sym, twice); err == nil {
		t.Fatal("a save that breaks the key was accepted")
	}
	if latest, _ := s.LatestFiles(sym); len(latest) != 1 || latest[0].Raw != 3 {
		t.Errorf("a failed save changed the data: %+v", latest)
	}
}

func TestPageLoads(t *testing.T) {
	t.Parallel()
	s := open(t)
	loads := []application.PathDay{
		{Path: "symdiary.com/", Day: oct8, Count: 1},
		{Path: "symdiary.com/", Day: oct9, Count: 2},
		{Path: "symdiary.com/", Day: oct9, Count: 3},
	}
	if err := s.SavePageLoads(oct8, oct9, loads); err != nil {
		t.Fatal(err)
	}
	_ = s.SavePageLoads(oct9, oct10, []application.PathDay{{Path: "symdiary.com/", Day: oct10, Count: 7}})
	got, err := s.PageLoads(oct8, oct10)
	want := []application.PathDay{{Path: "symdiary.com/", Day: oct8, Count: 1}, {Path: "symdiary.com/", Day: oct10, Count: 7}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("page loads %+v err %v; want %+v", got, err, want)
	}
}

func TestMeta(t *testing.T) {
	t.Parallel()
	s := open(t)
	if _, found, err := s.Preferences(); found || err != nil {
		t.Errorf("preferences found %v %v", found, err)
	}
	p := application.Preferences{IntervalHours: 6, Period: domain.Year, UpdateCheck: false}
	_ = s.SavePreferences(p)
	if got, found, _ := s.Preferences(); !found || got != p {
		t.Errorf("preferences %+v", got)
	}
	r := application.CheckRecord{LastSuccess: time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC), Failure: "x"}
	_ = s.SaveCheckRecord(r)
	if got, _ := s.CheckRecord(); !got.LastSuccess.Equal(r.LastSuccess) || got.Failure != "x" {
		t.Errorf("record %+v", got)
	}
}

func TestReopenKeepsData(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "visitron.db")
	s, _ := Open(path)
	_, _ = s.AddWebsite(application.Website{Address: addr(t, "symdiary.com")})
	_ = s.Close()
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	if got, _ := s.Websites(); len(got) != 1 {
		t.Errorf("after reopening: %+v", got)
	}
}

func TestNewerSchemaRefused(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "visitron.db")
	s, _ := Open(path)
	_, _ = s.db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, len(migrations)+1))
	_ = s.Close()
	if _, err := Open(path); !errors.Is(err, ErrNewerSchema) {
		t.Errorf("newer file: %v", err)
	}
}

func TestUnreadableFolder(t *testing.T) {
	t.Parallel()
	blocker := filepath.Join(t.TempDir(), "file")
	s, _ := Open(blocker)
	_ = s.Close()
	if _, err := Open(filepath.Join(blocker, "visitron.db")); err == nil {
		t.Error("a file standing where the folder goes was not reported")
	}
}

func TestDefaultPath(t *testing.T) {
	t.Parallel()
	p, err := DefaultPath()
	if err != nil || filepath.Base(p) != "visitron.db" || filepath.Base(filepath.Dir(p)) != "Visitron" {
		t.Errorf("DefaultPath = %q %v", p, err)
	}
}

func TestStoredDayMustParse(t *testing.T) {
	t.Parallel()
	s := open(t)
	// Days compare as text, so the damaged day must sort inside the range.
	_, _ = s.db.Exec(`INSERT INTO page_loads (day, path, count) VALUES ('2026-10-0x', 'x', 1)`)
	if _, err := s.PageLoads(oct8, oct10); err == nil {
		t.Error("a damaged day was read")
	}
	_, _ = s.db.Exec(`INSERT INTO release_files VALUES ('bad', 'someone/symdiary', 'someone', 'SymDiary', 'v1', 'f', 1)`)
	if _, err := s.LatestFiles(sym); err == nil {
		t.Error("a damaged day was read from release files")
	}
}

func TestClosedFileRefusesEverything(t *testing.T) {
	t.Parallel()
	s := open(t)
	w := application.Website{ID: 1, Address: addr(t, "a.example.com"), Repos: []domain.Repo{sym}}
	_ = s.Close()
	_, e1 := s.Websites()
	_, e2 := s.AddWebsite(w)
	_, e4 := s.LatestFiles(sym)
	_, e5 := s.History(sym, oct9)
	_, e6 := s.PageLoads(oct9, oct9)
	_, _, e7 := s.Preferences()
	_, e8 := s.CheckRecord()
	for i, err := range []error{e1, e2, s.UpdateWebsite(w), s.DeleteWebsite(1),
		s.SaveFiles(oct9, sym, []domain.ReleaseFile{file("a", 1)}), e4, e5,
		s.SavePageLoads(oct9, oct9, nil), e6, e7, s.SavePreferences(application.Preferences{}), e8,
		s.SaveCheckRecord(application.CheckRecord{}), s.migrate()} {
		if err == nil {
			t.Errorf("operation %d succeeded on a closed file", i)
		}
	}
}

func TestUnavailableRefusesEverything(t *testing.T) {
	t.Parallel()
	why := errors.New("the file is locked")
	u := Unavailable{Reason: why}
	_, e1 := u.Websites()
	_, e2 := u.AddWebsite(application.Website{})
	_, e4 := u.LatestFiles(sym)
	_, e5 := u.History(sym, oct9)
	_, e6 := u.PageLoads(oct9, oct9)
	_, _, e7 := u.Preferences()
	_, e8 := u.CheckRecord()
	for i, err := range []error{e1, e2, u.UpdateWebsite(application.Website{}), u.DeleteWebsite(1),
		u.SaveFiles(oct9, sym, nil), e4, e5, u.SavePageLoads(oct9, oct9, nil), e6, e7,
		u.SavePreferences(application.Preferences{}), e8, u.SaveCheckRecord(application.CheckRecord{})} {
		if !errors.Is(err, why) {
			t.Errorf("operation %d: %v", i, err)
		}
	}
}
