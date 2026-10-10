// Package github reads release download counts from the GitHub REST API,
// anonymously or with the owner's optional token (C-4).
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"visitron/internal/application"
	"visitron/internal/domain"
	"visitron/internal/infrastructure/web"
)

// DefaultBase is the GitHub REST API.
const DefaultBase = "https://api.github.com"

// pageSize is the most releases GitHub returns per page.
const pageSize = 100

// bodyLimit caps one page of releases; a hundred releases with their notes
// sit far below it.
const bodyLimit = 16 << 20

// The headers GitHub states its rate limit in, measured 2026-10-09.
const (
	remainingHeader = "X-RateLimit-Remaining"
	resetHeader     = "X-RateLimit-Reset"
	noneRemaining   = "0"
)

// maxPages caps the pages of one repository's releases, so a Link that never
// ends cannot hold a check. At pageSize a page it is ten thousand releases.
const maxPages = 100

// Refusals of a next page GitHub named (NFR-SEC-001, house robustness rule 5).
var (
	errForeignPage  = errors.New("GitHub named a next page on another host, which was not followed")
	errTooManyPages = errors.New("GitHub named more pages of releases than Visitron reads")
)

// nextLink finds the next page in GitHub's Link header.
var nextLink = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)

// Client reads GitHub.
type Client struct {
	web     *web.Client
	secrets application.Secrets
	base    string
}

// NewClient builds a GitHub reader at base, taking the token from secrets on
// every call so a token saved in Settings applies at once.
func NewClient(w *web.Client, secrets application.Secrets, base string) *Client {
	return &Client{web: w, secrets: secrets, base: base}
}

type release struct {
	Tag    string `json:"tag_name"`
	Assets []struct {
		Name      string `json:"name"`
		Downloads int    `json:"download_count"`
	} `json:"assets"`
}

// Files lists every release file of repo, following GitHub's pages.
func (c *Client) Files(ctx context.Context, repo domain.Repo) ([]domain.ReleaseFile, time.Time, error) {
	token, err := c.secrets.Get(application.GitHubToken)
	if err != nil {
		return nil, time.Time{}, err
	}
	url := fmt.Sprintf("%s/repos/%s/releases?per_page=%d", c.base, repo, pageSize)
	var files []domain.ReleaseFile
	for pages := 0; url != ""; pages++ {
		// The next page comes from GitHub's answer, so it is foreign input:
		// the token goes only under base and the pages are counted.
		if !strings.HasPrefix(url, c.base+"/") {
			return nil, time.Time{}, errForeignPage
		}
		if pages == maxPages {
			return nil, time.Time{}, errTooManyPages
		}
		resp, err := c.web.Get(ctx, url, headers(token), bodyLimit)
		if err != nil {
			return nil, time.Time{}, err
		}
		if reset, limited := rateLimited(resp); limited {
			return nil, reset, application.ErrRateLimited
		}
		if !resp.OK() {
			return nil, time.Time{}, resp.StatusError("GitHub")
		}
		var page []release
		if err := json.Unmarshal(resp.Body, &page); err != nil {
			return nil, time.Time{}, fmt.Errorf("reading GitHub's releases of %s: %w", repo, err)
		}
		for _, r := range page {
			for _, a := range r.Assets {
				files = append(files, domain.ReleaseFile{Repo: repo, Release: r.Tag, Name: a.Name, Raw: a.Downloads})
			}
		}
		url = ""
		if m := nextLink.FindStringSubmatch(resp.Header.Get("Link")); m != nil {
			url = m[1]
		}
	}
	return files, time.Time{}, nil
}

// Exists reports whether repo is a public repository (FR-009).
func (c *Client) Exists(ctx context.Context, repo domain.Repo) (bool, error) {
	token, err := c.secrets.Get(application.GitHubToken)
	if err != nil {
		return false, err
	}
	resp, err := c.web.Get(ctx, fmt.Sprintf("%s/repos/%s", c.base, repo), headers(token), bodyLimit)
	switch {
	case err != nil:
		return false, err
	case resp.Status == http.StatusNotFound:
		return false, nil
	case !resp.OK():
		return false, resp.StatusError("GitHub")
	}
	return true, nil
}

// Verify tries token once against the rate limit endpoint, which answers 401
// for a token GitHub does not accept (FR-061).
func (c *Client) Verify(ctx context.Context, token string) error {
	resp, err := c.web.Get(ctx, c.base+"/rate_limit", headers(token), bodyLimit)
	if err != nil {
		return err
	}
	if !resp.OK() {
		return resp.StatusError("GitHub")
	}
	return nil
}

// headers are GitHub's recommended request headers, plus the token when set.
func headers(token string) map[string]string {
	h := map[string]string{
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": "2022-11-28",
		"User-Agent":           "Visitron",
	}
	if token != "" {
		h["Authorization"] = "Bearer " + token
	}
	return h
}

// rateLimited reports a refusal for the rate limit (GitHub documents it as 403
// or 429 with no requests remaining) along with when it resets.
func rateLimited(resp web.Response) (time.Time, bool) {
	if resp.Status != http.StatusForbidden && resp.Status != http.StatusTooManyRequests {
		return time.Time{}, false
	}
	if resp.Header.Get(remainingHeader) != noneRemaining {
		return time.Time{}, false
	}
	seconds, err := strconv.ParseInt(resp.Header.Get(resetHeader), 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(seconds, 0), true
}
