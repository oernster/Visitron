package store

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/oernster/visitron/internal/application"
)

// The keys of the meta table.
const (
	preferencesKey = "preferences"
	checkRecordKey = "check-record"
)

func (s *Store) getMeta(key string) (string, bool, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM meta WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return value, err == nil, err
}

func (s *Store) setMeta(key, value string) error {
	_, err := s.db.Exec(`INSERT INTO meta (key, value) VALUES (?, ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// getJSON reads key into v, answering whether it was there.
func (s *Store) getJSON(key string, v any) (bool, error) {
	text, found, err := s.getMeta(key)
	if err != nil || !found {
		return false, err
	}
	return true, json.Unmarshal([]byte(text), v)
}

func (s *Store) setJSON(key string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.setMeta(key, string(data))
}

// Preferences answers the saved preferences, found false when none were.
func (s *Store) Preferences() (application.Preferences, bool, error) {
	var p application.Preferences
	found, err := s.getJSON(preferencesKey, &p)
	return p, found, err
}

// SavePreferences keeps the preferences.
func (s *Store) SavePreferences(p application.Preferences) error { return s.setJSON(preferencesKey, p) }

// CheckRecord answers what is known of the latest checks.
func (s *Store) CheckRecord() (application.CheckRecord, error) {
	var r application.CheckRecord
	_, err := s.getJSON(checkRecordKey, &r)
	return r, err
}

// SaveCheckRecord keeps the check record.
func (s *Store) SaveCheckRecord(r application.CheckRecord) error { return s.setJSON(checkRecordKey, r) }
