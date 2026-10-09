package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/oernster/visitron/internal/application"
	"github.com/oernster/visitron/internal/domain"
)

// dayLayout reads the stored form of a day, which domain.Day.String writes.
const dayLayout = "%04d-%02d-%02d"

// errDamagedDay refuses a stored day that is not one domain.Day.String wrote.
var errDamagedDay = errors.New("a stored day is damaged")

func parseDay(text string) (domain.Day, error) {
	var d domain.Day
	if _, err := fmt.Sscanf(text, dayLayout, &d.Year, &d.Month, &d.Date); err != nil {
		return domain.Day{}, fmt.Errorf("a stored day %q: %w", text, err)
	}
	// Sscanf stops quietly at a stray character, so "2026-10-0x" reads as a
	// day; only a day that prints back as stored is the day that was stored.
	if d.String() != text {
		return domain.Day{}, fmt.Errorf("%w: %q", errDamagedDay, text)
	}
	return d, nil
}

// SaveFiles replaces repo's release files for day.
func (s *Store) SaveFiles(day domain.Day, repo domain.Repo, files []domain.ReleaseFile) error {
	return s.inTransaction(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`DELETE FROM release_files WHERE repo = ? AND day = ?`, repoKey(repo), day.String()); err != nil {
			return err
		}
		for _, f := range files {
			if _, err := tx.Exec(`INSERT INTO release_files (day, repo, owner, name, release, file, raw)
				VALUES (?, ?, ?, ?, ?, ?, ?)`, day.String(), repoKey(repo), f.Repo.Owner, f.Repo.Name,
				f.Release, f.Name, f.Raw); err != nil {
				return err
			}
		}
		return nil
	})
}

// LatestFiles answers repo's release files from its most recent day.
func (s *Store) LatestFiles(repo domain.Repo) ([]domain.ReleaseFile, error) {
	byDay, err := s.filesByDay(`SELECT day, owner, name, release, file, raw FROM release_files
		WHERE repo = ?1 AND day = (SELECT max(day) FROM release_files WHERE repo = ?1)
		ORDER BY release, file`, repoKey(repo))
	if err != nil || len(byDay) == 0 {
		return nil, err
	}
	return byDay[0].files, nil
}

// Snapshots answers repo's snapshot of each day from first onwards.
func (s *Store) Snapshots(repo domain.Repo, first domain.Day) ([]domain.Snapshot, error) {
	byDay, err := s.filesByDay(`SELECT day, owner, name, release, file, raw FROM release_files
		WHERE repo = ? AND day >= ? ORDER BY day, release, file`, repoKey(repo), first.String())
	if err != nil {
		return nil, err
	}
	snaps := make([]domain.Snapshot, len(byDay))
	for i, d := range byDay {
		snaps[i] = domain.SnapshotOf(d.day, d.files)
	}
	return snaps, nil
}

type dayFiles struct {
	day   domain.Day
	files []domain.ReleaseFile
}

// filesByDay runs a query ordered by day and groups its rows by day.
func (s *Store) filesByDay(query string, args ...any) ([]dayFiles, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []dayFiles
	for rows.Next() {
		var dayText string
		var f domain.ReleaseFile
		if err := rows.Scan(&dayText, &f.Repo.Owner, &f.Repo.Name, &f.Release, &f.Name, &f.Raw); err != nil {
			return nil, err
		}
		day, err := parseDay(dayText)
		if err != nil {
			return nil, err
		}
		if n := len(out); n == 0 || out[n-1].day != day {
			out = append(out, dayFiles{day: day})
		}
		out[len(out)-1].files = append(out[len(out)-1].files, f)
	}
	return out, rows.Err()
}

// SavePageLoads replaces the page loads held from first to last.
func (s *Store) SavePageLoads(first, last domain.Day, loads []application.PathDay) error {
	return s.inTransaction(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`DELETE FROM page_loads WHERE day BETWEEN ? AND ?`, first.String(), last.String()); err != nil {
			return err
		}
		for _, pd := range loads {
			if _, err := tx.Exec(`INSERT INTO page_loads (day, path, count) VALUES (?, ?, ?)
				ON CONFLICT (day, path) DO UPDATE SET count = count + excluded.count`,
				pd.Day.String(), pd.Path, pd.Count); err != nil {
				return err
			}
		}
		return nil
	})
}

// PageLoads answers the page loads held from first to last.
func (s *Store) PageLoads(first, last domain.Day) ([]application.PathDay, error) {
	rows, err := s.db.Query(`SELECT day, path, count FROM page_loads WHERE day BETWEEN ? AND ? ORDER BY day, path`,
		first.String(), last.String())
	if err != nil {
		return nil, err
	}
	var out []application.PathDay
	for rows.Next() {
		var dayText string
		var pd application.PathDay
		if err := rows.Scan(&dayText, &pd.Path, &pd.Count); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if pd.Day, err = parseDay(dayText); err != nil {
			_ = rows.Close()
			return nil, err
		}
		out = append(out, pd)
	}
	return out, errors.Join(rows.Err(), rows.Close())
}
