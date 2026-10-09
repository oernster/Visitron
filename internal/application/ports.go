// Package application holds Visitron's use cases: one service per thing the
// owner can do, each driven through the interfaces declared here, so every
// action runs headlessly from a test with hand-written fakes.
package application

import (
	"context"
	"errors"
	"time"

	"github.com/oernster/visitron/internal/domain"
)

// Clock tells the time in the zone Windows is set to.
type Clock interface {
	Now() time.Time
}

// Fetcher reads one page of a website for the crawl. It enforces the size
// and time caps of FR-005 itself.
type Fetcher interface {
	Fetch(ctx context.Context, url string) (string, error)
}

// ErrRateLimited is GitHub refusing for its rate limit (FR-035). A call that
// answers it also answers the reset time GitHub gave.
var ErrRateLimited = errors.New("GitHub's rate limit was reached")

// Releases reads GitHub.
type Releases interface {
	// Files lists every release file of repo with its raw download count.
	// On ErrRateLimited, reset is when GitHub will answer again.
	Files(ctx context.Context, repo domain.Repo) (files []domain.ReleaseFile, reset time.Time, err error)
	// Exists reports whether repo is a public repository.
	Exists(ctx context.Context, repo domain.Repo) (bool, error)
	// Verify tries token once (FR-061).
	Verify(ctx context.Context, token string) error
}

// PathDay is GoatCounter's page loads for one path on one day.
type PathDay struct {
	Path  string
	Day   domain.Day
	Count int
}

// PageLoads reads GoatCounter.
type PageLoads interface {
	// Daily lists page loads per path per day from first to last inclusive.
	Daily(ctx context.Context, key string, first, last domain.Day) ([]PathDay, error)
	// Verify tries key once (FR-061).
	Verify(ctx context.Context, key string) error
}

// Secret names the two secrets Visitron keeps (C-4).
type Secret string

// The secrets, each kept in Windows Credential Manager.
const (
	GoatCounterKey Secret = "goatcounter-key"
	GitHubToken    Secret = "github-token"
)

// Secrets keeps the key and token out of every file (NFR-SEC-001). Get
// answers "" when the secret is not set.
type Secrets interface {
	Get(name Secret) (string, error)
	Set(name Secret, value string) error
	Delete(name Secret) error
}

// Startup is the start-with-Windows entry (FR-052).
type Startup interface {
	Enabled() (bool, error)
	SetEnabled(on bool) error
}

// Website is one recorded website.
type Website struct {
	ID      int64
	Address domain.Address
	Repos   []domain.Repo
}

// Preferences are the settings kept in the store rather than in Windows.
type Preferences struct {
	IntervalHours int
	Period        domain.Period
	UpdateCheck   bool
}

// CheckRecord is what is known of the latest checks (FR-044).
type CheckRecord struct {
	LastSuccess time.Time
	LastFailure time.Time
	Failure     string
}

// Store is Visitron's local data (C-3). Each method is one transaction, so an
// interruption leaves the last good state (NFR-REL-001).
type Store interface {
	Websites() ([]Website, error)
	AddWebsite(w Website) (int64, error)
	UpdateWebsite(w Website) error
	DeleteWebsite(id int64) error
	Seeded() (bool, error)
	MarkSeeded() error

	// SaveFiles records the release files of repo as read at a check on day,
	// replacing that repo's snapshot for the day.
	SaveFiles(day domain.Day, repo domain.Repo, files []domain.ReleaseFile) error
	// LatestFiles lists the most recent release files of repo.
	LatestFiles(repo domain.Repo) ([]domain.ReleaseFile, error)
	// Snapshots lists repo's kept snapshots from first onwards, in day order.
	Snapshots(repo domain.Repo, first domain.Day) ([]domain.Snapshot, error)

	// SavePageLoads replaces the page loads held for the days covered.
	SavePageLoads(first, last domain.Day, loads []PathDay) error
	PageLoads(first, last domain.Day) ([]PathDay, error)

	// Preferences answers found false when none were ever saved.
	Preferences() (p Preferences, found bool, err error)
	SavePreferences(p Preferences) error
	CheckRecord() (CheckRecord, error)
	SaveCheckRecord(r CheckRecord) error
}
