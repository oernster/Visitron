// Package store keeps Visitron's data in one SQLite file (C-3), ported from
// SymDiary's store.
//
// Every write is one transaction. The file runs in WAL mode with synchronous
// FULL, so a check reported as saved survives a power loss and an interrupted
// one leaves the last good state (NFR-REL-001). The store never logs.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/oernster/visitron/internal/product"

	// The pure-Go SQLite driver, registered as "sqlite".
	_ "modernc.org/sqlite"
)

// driverName is the name modernc.org/sqlite registers itself under.
const driverName = "sqlite"

// busyTimeoutMillis is how long a statement waits for a lock held by another
// connection; Visitron holds one, so the wait covers an outside tool only.
const busyTimeoutMillis = 5000

// dsnLayout opens the file with the durability pragmas on the connection.
const dsnLayout = "file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(FULL)" +
	"&_pragma=foreign_keys(1)&_pragma=busy_timeout(%d)"

// directoryMode is the permission a missing data folder is made with.
const directoryMode = 0o700

// ErrNewerSchema refuses a file written by a later Visitron.
var ErrNewerSchema = errors.New("the data was written by a newer Visitron")

// Store is the SQLite file.
type Store struct {
	db *sql.DB
}

// DefaultPath is where the data lives: %LOCALAPPDATA%\Visitron\visitron.db.
func DefaultPath() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("finding the folder for Visitron's data: %w", err)
	}
	return filepath.Join(base, product.Name, product.RecordFileName), nil
}

// Open opens the file at path, creating it and its folder when absent and
// upgrading an older schema. A file that cannot be read is reported and left
// as it was.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), directoryMode); err != nil {
		return nil, fmt.Errorf("creating the folder for %s: %w", path, err)
	}
	db, err := sql.Open(driverName, fmt.Sprintf(dsnLayout, filepath.ToSlash(path), busyTimeoutMillis))
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	// One connection: Visitron is the single writer; a pragma set on one
	// connection is not set on another.
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return s, nil
}

// Close closes the file.
func (s *Store) Close() error { return s.db.Close() }

// migrations holds each schema version's statements, oldest first. Version N
// is migrations[N-1]; user_version says how many are applied. A released
// migration is never edited; a change is a new entry.
var migrations = []string{
	`CREATE TABLE websites (
		id   INTEGER PRIMARY KEY,
		host TEXT NOT NULL,
		path TEXT NOT NULL,
		UNIQUE (host, path)
	);
	CREATE TABLE website_repos (
		website_id INTEGER NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
		position   INTEGER NOT NULL,
		owner      TEXT NOT NULL,
		name       TEXT NOT NULL,
		PRIMARY KEY (website_id, position)
	);
	CREATE TABLE release_files (
		day     TEXT NOT NULL,
		repo    TEXT NOT NULL,
		owner   TEXT NOT NULL,
		name    TEXT NOT NULL,
		release TEXT NOT NULL,
		file    TEXT NOT NULL,
		raw     INTEGER NOT NULL,
		PRIMARY KEY (repo, day, release, file)
	);
	CREATE TABLE page_loads (
		day   TEXT NOT NULL,
		path  TEXT NOT NULL,
		count INTEGER NOT NULL,
		PRIMARY KEY (day, path)
	);
	CREATE TABLE meta (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`,
}

// migrate applies every migration the file has not had, each in one
// transaction with the version it reaches.
func (s *Store) migrate() error {
	var version int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version > len(migrations) {
		return fmt.Errorf("%w (schema %d, this one knows %d)", ErrNewerSchema, version, len(migrations))
	}
	for next := version; next < len(migrations); next++ {
		if err := s.inTransaction(func(tx *sql.Tx) error {
			if _, err := tx.Exec(migrations[next]); err != nil {
				return err
			}
			_, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, next+1))
			return err
		}); err != nil {
			return fmt.Errorf("upgrading to schema %d: %w", next+1, err)
		}
	}
	return nil
}

// inTransaction runs work in one transaction, committing only on success.
func (s *Store) inTransaction(work func(tx *sql.Tx) error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if err := work(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
