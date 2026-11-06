package store

import (
	"database/sql"
	"errors"
	"fmt"

	"visitron/internal/domain"
)

// The two readings kept of each repo (Amendment 33): the one taken at the
// last press of Refresh and the one at the press before it.
const (
	laterReading   = "later"
	earlierReading = "earlier"
)

// errDamagedReading refuses a stored reading that is neither of the two.
var errDamagedReading = errors.New("a stored refresh reading is damaged")

// SaveRefreshReading keeps files as repo's reading at this press; the reading
// it replaces becomes the earlier one and the one before that goes.
func (s *Store) SaveRefreshReading(repo domain.Repo, files []domain.ReleaseFile) error {
	key := repoKey(repo)
	return s.inTransaction(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`DELETE FROM refresh_files WHERE repo = ? AND reading = ?`,
			key, earlierReading); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE refresh_files SET reading = ? WHERE repo = ? AND reading = ?`,
			earlierReading, key, laterReading); err != nil {
			return err
		}
		for _, f := range files {
			if _, err := tx.Exec(`INSERT INTO refresh_files (reading, repo, owner, name, release, file, raw)
				VALUES (?, ?, ?, ?, ?, ?, ?)`, laterReading, key, f.Repo.Owner, f.Repo.Name,
				f.Release, f.Name, f.Raw); err != nil {
				return err
			}
		}
		return nil
	})
}

// RefreshReadings answers repo's readings at the press before last and at the
// last, with GitHub's own counts.
func (s *Store) RefreshReadings(repo domain.Repo) (earlier, later []domain.ReleaseFile, err error) {
	rows, err := s.db.Query(`SELECT reading, owner, name, release, file, raw FROM refresh_files
		WHERE repo = ? ORDER BY release, file`, repoKey(repo))
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var reading string
		var f domain.ReleaseFile
		if err := rows.Scan(&reading, &f.Repo.Owner, &f.Repo.Name, &f.Release, &f.Name, &f.Raw); err != nil {
			return nil, nil, err
		}
		switch reading {
		case laterReading:
			later = append(later, f)
		case earlierReading:
			earlier = append(earlier, f)
		default:
			return nil, nil, fmt.Errorf("%w: %q", errDamagedReading, reading)
		}
	}
	return earlier, later, rows.Err()
}
