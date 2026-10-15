package application

import (
	"context"
	"errors"
	"fmt"

	"visitron/internal/domain"
)

// ErrNoGoatCounter is GoatCounter not being set up: no key, else no site to
// use it with. It is an answer rather than a failure; downloads still work.
var ErrNoGoatCounter = errors.New("GoatCounter is not set up")

// goatCounter answers the site and key GoatCounter is read with;
// ErrNoGoatCounter while either is missing. The check and the countries both
// read it this way, so they cannot disagree about what "set up" means.
func goatCounter(store Store, secrets Secrets) (domain.GoatCounterSite, string, error) {
	key, err := secrets.Get(GoatCounterKey)
	if err != nil {
		return domain.GoatCounterSite{}, "", fmt.Errorf("reading the GoatCounter key: %w", err)
	}
	prefs, err := Preferred(store)
	if err != nil {
		return domain.GoatCounterSite{}, "", fmt.Errorf("reading the GoatCounter account name: %w", err)
	}
	site := prefs.Site()
	if key == "" || site.IsZero() {
		return domain.GoatCounterSite{}, "", ErrNoGoatCounter
	}
	return site, key, nil
}

// Countries is the visitors-by-country service (FR-047).
type Countries struct {
	store   Store
	loads   PageLoads
	secrets Secrets
	clock   Clock
}

// NewCountries builds the countries service.
func NewCountries(store Store, loads PageLoads, secrets Secrets, clock Clock) *Countries {
	return &Countries{store: store, loads: loads, secrets: secrets, clock: clock}
}

// Of answers website id's visitors by country over period, largest first.
// Only the paths the website owns are counted, by the rule the page loads
// use (FR-020); a website with no visited path in the period has none.
// ErrNoGoatCounter means GoatCounter is not set up.
func (c *Countries) Of(ctx context.Context, id int64, period domain.Period) ([]domain.CountryCount, error) {
	sites, err := c.store.Websites()
	if err != nil {
		return nil, err
	}
	var own *Website
	for i := range sites {
		if sites[i].ID == id {
			own = &sites[i]
		}
	}
	if own == nil {
		return nil, ErrNoSuchSite
	}
	return c.read(ctx, sites, period, func(owner domain.Address) bool { return owner == own.Address })
}

// All answers every website's visitors by country over period, largest
// first, as one read over every path a recorded website owns (FR-092), so
// GoatCounter does the counting across them. With no visited path there are
// none. ErrNoGoatCounter means GoatCounter is not set up.
func (c *Countries) All(ctx context.Context, period domain.Period) ([]domain.CountryCount, error) {
	sites, err := c.store.Websites()
	if err != nil {
		return nil, err
	}
	return c.read(ctx, sites, period, func(domain.Address) bool { return true })
}

// read answers the visitors by country to the paths owned, by FR-020's rule,
// by a website among sites that counted accepts. A path no website owns is
// never read.
func (c *Countries) read(ctx context.Context, sites []Website, period domain.Period,
	counted func(owner domain.Address) bool) ([]domain.CountryCount, error) {
	addrs := make([]domain.Address, len(sites))
	for i := range sites {
		addrs[i] = sites[i].Address
	}
	site, key, err := goatCounter(c.store, c.secrets)
	if err != nil {
		return nil, err
	}
	now := c.clock.Now()
	first, last := DaysBack(now, int(period)), DayOf(now)
	paths, err := c.loads.Paths(ctx, site, key, first, last)
	if err != nil {
		return nil, fmt.Errorf("GoatCounter: %w", err)
	}
	var ids []int64
	for _, p := range paths {
		if owner, ok := domain.Owner(p.Path, addrs); ok && counted(owner) {
			ids = append(ids, p.ID)
		}
	}
	if len(ids) == 0 {
		return []domain.CountryCount{}, nil
	}
	countries, err := c.loads.Countries(ctx, site, key, first, last, ids)
	if err != nil {
		return nil, fmt.Errorf("GoatCounter: %w", err)
	}
	domain.SortCountries(countries)
	return countries, nil
}
