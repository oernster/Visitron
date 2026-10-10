package application

import (
	"context"
	"errors"
	"fmt"

	"visitron/internal/domain"
)

// CrawlPageLimit is the most pages one crawl fetches (FR-005).
const CrawlPageLimit = 50

// Refusals from the website service.
var (
	ErrDuplicate    = errors.New("that website is already recorded")
	ErrNoSuchSite   = errors.New("that website is no longer recorded")
	ErrRepoNotFound = errors.New("GitHub has no public repository by that name")
)

// Proposal is what the Add and Edit dialogs offer once a crawl has run.
type Proposal struct {
	Address domain.Address
	Found   []domain.Repo
	Ticked  []domain.Repo
	// Problem says why the crawl could not read the site, "" when it could
	// (FR-008). The website can still be saved.
	Problem string
}

// Websites is the website service.
type Websites struct {
	store    Store
	fetcher  Fetcher
	releases Releases
}

// NewWebsites builds the website service.
func NewWebsites(store Store, fetcher Fetcher, releases Releases) *Websites {
	return &Websites{store: store, fetcher: fetcher, releases: releases}
}

// List answers every recorded website.
func (s *Websites) List() ([]Website, error) { return s.store.Websites() }

// Propose reduces entry to a site address and crawls it (FR-001). When
// editing, editID is the website being changed, so its own address is not a
// duplicate and its chosen repos stay ticked where found again (FR-004);
// otherwise it is 0.
func (s *Websites) Propose(ctx context.Context, entry string, editID int64) (Proposal, error) {
	addr, err := domain.Normalise(entry)
	if err != nil {
		return Proposal{}, err
	}
	sites, err := s.store.Websites()
	if err != nil {
		return Proposal{}, err
	}
	var kept []domain.Repo
	for _, w := range sites {
		switch {
		case w.ID == editID:
			kept = w.Repos
		case w.Address == addr:
			return Proposal{}, fmt.Errorf("%w: %s", ErrDuplicate, addr.URL())
		}
	}
	found, problem := s.crawl(ctx, addr)
	ticked := domain.PreTicked(addr, found)
	for _, r := range found {
		if domain.ContainsRepo(kept, r) && !domain.ContainsRepo(ticked, r) {
			ticked = append(ticked, r)
		}
	}
	return Proposal{Address: addr, Found: found, Ticked: ticked, Problem: problem}, nil
}

// crawl fetches the site and the pages it links under its own path (up to
// CrawlPageLimit) then lists the repos they name (FR-005, FR-006). Only a
// failure to read the site address itself is a problem; a broken inner link
// is skipped.
func (s *Websites) crawl(ctx context.Context, addr domain.Address) ([]domain.Repo, string) {
	queue := []string{addr.URL()}
	seen := map[string]bool{addr.URL(): true}
	var text string
	for fetched := 0; len(queue) > 0 && fetched < CrawlPageLimit; fetched++ {
		page := queue[0]
		queue = queue[1:]
		body, err := s.fetcher.Fetch(ctx, page)
		if err != nil {
			if fetched == 0 {
				return nil, err.Error()
			}
			continue
		}
		text += body
		for _, link := range domain.SiteLinks(addr, page, body) {
			if !seen[link] {
				seen[link] = true
				queue = append(queue, link)
			}
		}
	}
	return domain.FindRepos(text), ""
}

// Confirm checks a repository typed by hand with GitHub (FR-009).
func (s *Websites) Confirm(ctx context.Context, text string) (domain.Repo, error) {
	repo, err := domain.ParseRepo(text)
	if err != nil {
		return domain.Repo{}, err
	}
	ok, err := s.releases.Exists(ctx, repo)
	if err != nil {
		return domain.Repo{}, err
	}
	if !ok {
		return domain.Repo{}, fmt.Errorf("%w: %s", ErrRepoNotFound, repo)
	}
	return repo, nil
}

// Save records a website: a new one when id is 0, else the website with that
// id, given its new address and chosen repos.
func (s *Websites) Save(id int64, addr domain.Address, repos []domain.Repo) (int64, error) {
	w := Website{ID: id, Address: addr, Repos: repos}
	if id == 0 {
		return s.store.AddWebsite(w)
	}
	return id, s.store.UpdateWebsite(w)
}

// Delete removes a website with its history (FR-010). The window asks first.
func (s *Websites) Delete(id int64) error { return s.store.DeleteWebsite(id) }
