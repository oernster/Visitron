package store

import (
	"github.com/oernster/visitron/internal/application"
	"github.com/oernster/visitron/internal/domain"
)

// Unavailable stands in for a data file that could not be opened, ported from
// SymDiary. Every operation refuses with the reason, so the window still opens
// and says why nothing can be shown, while the file itself is never touched
// (NFR-REL-002).
type Unavailable struct {
	Reason error
}

// Websites refuses.
func (u Unavailable) Websites() ([]application.Website, error) { return nil, u.Reason }

// AddWebsite refuses.
func (u Unavailable) AddWebsite(application.Website) (int64, error) { return 0, u.Reason }

// UpdateWebsite refuses.
func (u Unavailable) UpdateWebsite(application.Website) error { return u.Reason }

// DeleteWebsite refuses.
func (u Unavailable) DeleteWebsite(int64) error { return u.Reason }

// SaveFiles refuses.
func (u Unavailable) SaveFiles(domain.Day, domain.Repo, []domain.ReleaseFile) error { return u.Reason }

// LatestFiles refuses.
func (u Unavailable) LatestFiles(domain.Repo) ([]domain.ReleaseFile, error) { return nil, u.Reason }

// Snapshots refuses.
func (u Unavailable) Snapshots(domain.Repo, domain.Day) ([]domain.Snapshot, error) {
	return nil, u.Reason
}

// SavePageLoads refuses.
func (u Unavailable) SavePageLoads(domain.Day, domain.Day, []application.PathDay) error {
	return u.Reason
}

// PageLoads refuses.
func (u Unavailable) PageLoads(domain.Day, domain.Day) ([]application.PathDay, error) {
	return nil, u.Reason
}

// Preferences refuses.
func (u Unavailable) Preferences() (application.Preferences, bool, error) {
	return application.Preferences{}, false, u.Reason
}

// SavePreferences refuses.
func (u Unavailable) SavePreferences(application.Preferences) error { return u.Reason }

// CheckRecord refuses.
func (u Unavailable) CheckRecord() (application.CheckRecord, error) {
	return application.CheckRecord{}, u.Reason
}

// SaveCheckRecord refuses.
func (u Unavailable) SaveCheckRecord(application.CheckRecord) error { return u.Reason }
