// Package update reads Visitron's latest published release from GitHub
// (FR-075), ported from Bridge Talk.
//
// It rides the shared web client, so it adds no connection of its own
// (NFR-PRIV-001). It asks unauthenticated: the owner's token is for the
// figures, never for this. The endpoint answers only a release that is
// published, neither a draft nor a pre-release, so that guard is the
// endpoint's own contract rather than a check here.
package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"visitron/internal/application"
	"visitron/internal/domain"
	"visitron/internal/infrastructure/github"
	"visitron/internal/infrastructure/web"
)

// latestPath is the latest-release endpoint for a repository, below the GitHub
// API's base. The repository is the build's own, never named in the source
// (Amendment 15).
func latestPath(repo domain.Repo) string { return "/repos/" + repo.String() + "/releases/latest" }

// releaseHost is where every address in a release's answer must point. An
// address is handed to the browser, so one anywhere else is refused rather
// than opened.
const releaseHost = "https://github.com/"

// acceptHeader asks the API for its own JSON.
const acceptHeader = "application/vnd.github+json"

// requestTimeout is how long the check waits before giving up.
const requestTimeout = 5 * time.Second

// answerCap is the most of an answer read. A release's answer is a few
// kilobytes; a larger one is not a release.
const answerCap = 1 << 20

// errNotARelease refuses an answer with no tag or no page on GitHub.
var errNotARelease = errors.New("the answer names no release tag or no release page on GitHub")

// answer is the part of GitHub's answer the check reads.
type answer struct {
	Tag    string `json:"tag_name"`
	Page   string `json:"html_url"`
	Assets []struct {
		Name     string `json:"name"`
		Download string `json:"browser_download_url"`
	} `json:"assets"`
}

// Source is an application.ReleaseSource over GitHub.
type Source struct {
	client  *web.Client
	address string
}

// New builds the source the application uses, reading the releases of repo:
// "owner/name" as the build states it. An empty or malformed one leaves the
// source with nothing to ask.
func New(client *web.Client, repo string) *Source { return NewAt(client, github.DefaultBase, repo) }

// NewAt builds a source against another API base, for a test.
func NewAt(client *web.Client, base, repo string) *Source {
	named, err := domain.ParseRepo(repo)
	if err != nil {
		return &Source{client: client}
	}
	return &Source{client: client, address: base + latestPath(named)}
}

// Latest reads the latest published release. Every failure is an error; the
// check reads them all alike, as unreachable.
func (s *Source) Latest(ctx context.Context) (application.Release, error) {
	if s.address == "" {
		return application.Release{}, application.ErrNoReleaseSource
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	resp, err := s.client.Get(ctx, s.address, map[string]string{"Accept": acceptHeader}, answerCap)
	if err != nil {
		return application.Release{}, fmt.Errorf("asking for the latest release: %w", err)
	}
	// GitHub answers 404 for a repository with no published release, which
	// is the answer before the first one rather than a failure.
	if resp.Status == http.StatusNotFound {
		return application.Release{}, application.ErrNoRelease
	}
	if !resp.OK() {
		return application.Release{}, resp.StatusError("GitHub")
	}
	return parse(resp.Body)
}

// parse reads a release out of an answer, passing over any file that names
// no file or names an address off GitHub.
func parse(body []byte) (application.Release, error) {
	var read answer
	if err := json.Unmarshal(body, &read); err != nil {
		return application.Release{}, fmt.Errorf("reading the latest release: %w", err)
	}
	if read.Tag == "" || !strings.HasPrefix(read.Page, releaseHost) {
		return application.Release{}, errNotARelease
	}
	release := application.Release{Tag: read.Tag, Page: read.Page}
	for _, each := range read.Assets {
		if each.Name != "" && strings.HasPrefix(each.Download, releaseHost) {
			release.Assets = append(release.Assets, application.Asset{Name: each.Name, Address: each.Download})
		}
	}
	return release, nil
}
