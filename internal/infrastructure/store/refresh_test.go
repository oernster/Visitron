package store

import (
	"errors"
	"path/filepath"
	"testing"

	"visitron/internal/application"
	"visitron/internal/domain"
)

func TestRefreshReadingsKeepTheLastTwoPresses(t *testing.T) {
	t.Parallel()
	s := open(t)
	if earlier, later, err := s.RefreshReadings(sym); earlier != nil || later != nil || err != nil {
		t.Fatalf("before any press: %v %v %v", earlier, later, err)
	}
	for _, raw := range []int{10, 13, 20} {
		if err := s.SaveRefreshReading(sym, []domain.ReleaseFile{file("WidgetSetup.exe", raw)}); err != nil {
			t.Fatal(err)
		}
	}
	earlier, later, err := s.RefreshReadings(domain.Repo{Owner: "SOMEONE", Name: "WIDGET"})
	if err != nil || len(earlier) != 1 || len(later) != 1 || earlier[0].Raw != 13 || later[0].Raw != 20 {
		t.Fatalf("earlier %+v later %+v err %v; want 13 then 20", earlier, later, err)
	}
	if later[0].Repo != sym || later[0].Name != "WidgetSetup.exe" {
		t.Errorf("reading %+v lost its file", later[0])
	}
}

func TestRefreshReadingsSurviveReopening(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "visitron.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = s.SaveRefreshReading(sym, []domain.ReleaseFile{file("WidgetSetup.exe", 503)})
	_ = s.Close()
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	if _, later, err := s.RefreshReadings(sym); err != nil || len(later) != 1 || later[0].Raw != 503 {
		t.Errorf("after reopening: %+v %v", later, err)
	}
}

func TestARefreshReadingIsAtomic(t *testing.T) {
	t.Parallel()
	s := open(t)
	_ = s.SaveRefreshReading(sym, []domain.ReleaseFile{file("WidgetSetup.exe", 3)})
	twice := []domain.ReleaseFile{file("WidgetSetup.exe", 4), file("WidgetSetup.exe", 5)}
	if err := s.SaveRefreshReading(sym, twice); err == nil {
		t.Fatal("a reading that breaks the key was accepted")
	}
	if earlier, later, _ := s.RefreshReadings(sym); earlier != nil || len(later) != 1 || later[0].Raw != 3 {
		t.Errorf("a failed save changed the readings: %+v %+v", earlier, later)
	}
}

func TestDeleteRemovesRefreshReadings(t *testing.T) {
	t.Parallel()
	s := open(t)
	id, _ := s.AddWebsite(application.Website{Address: addr(t, "a.example.com"), Repos: []domain.Repo{sym}})
	_ = s.SaveRefreshReading(sym, []domain.ReleaseFile{file("WidgetSetup.exe", 3)})
	if err := s.DeleteWebsite(id); err != nil {
		t.Fatal(err)
	}
	if _, later, _ := s.RefreshReadings(sym); later != nil {
		t.Errorf("readings of a repo no website chooses were kept: %+v", later)
	}
}

func TestADamagedRefreshReadingIsRefused(t *testing.T) {
	t.Parallel()
	s := open(t)
	_, _ = s.db.Exec(`INSERT INTO refresh_files VALUES ('sideways', 'someone/widget', 'someone', 'Widget', 'v1', 'f', 1)`)
	if _, _, err := s.RefreshReadings(sym); !errors.Is(err, errDamagedReading) {
		t.Errorf("damaged reading: %v", err)
	}
	_, _ = s.db.Exec(`DELETE FROM refresh_files`)
	_, _ = s.db.Exec(`INSERT INTO refresh_files VALUES ('later', 'someone/widget', 'someone', 'Widget', 'v1', 'f', 'many')`)
	if _, _, err := s.RefreshReadings(sym); err == nil {
		t.Error("a reading whose count is not a number was read")
	}
}
