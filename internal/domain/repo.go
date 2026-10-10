package domain

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

// ErrRepoForm refuses a repository typed by hand that is not owner/name.
var ErrRepoForm = errors.New("a repository is written owner/name")

// Repo is a GitHub repository named owner/name. Its case is kept as found;
// two repos are the same when they match ignoring case, as on GitHub.
type Repo struct {
	Owner string
	Name  string
}

// String is the owner/name form.
func (r Repo) String() string { return r.Owner + pathSep + r.Name }

// Same reports whether two repos name the same repository.
func (r Repo) Same(other Repo) bool { return strings.EqualFold(r.String(), other.String()) }

// ParseRepo reads a repository typed by hand as owner/name (FR-009).
func ParseRepo(text string) (Repo, error) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(text), pathSep), pathSep)
	if len(parts) != 2 || !validSegment(parts[0]) || !validSegment(parts[1]) {
		return Repo{}, ErrRepoForm
	}
	return Repo{Owner: parts[0], Name: strings.TrimSuffix(parts[1], gitSuffix)}, nil
}

const gitSuffix = ".git"

var segmentPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func validSegment(s string) bool {
	return segmentPattern.MatchString(s) && strings.Trim(s, labelSep) != ""
}

// githubMention finds github.com with the label in front of it and the next
// three path segments. The label decides what the segments mean.
var githubMention = regexp.MustCompile(
	`(?i)([a-z0-9.-]*?)github\.com((?:/[A-Za-z0-9._-]+){2,3})`)

const (
	webLabel   = ""
	wwwLabel   = "www."
	apiLabel   = "api."
	apiReposAt = "repos"
)

// notOwners are first path segments on github.com that are site pages, not
// accounts, so github.com/sponsors/x names no repository.
var notOwners = map[string]bool{
	"about": true, "apps": true, "collections": true, "explore": true,
	"features": true, "login": true, "marketplace": true, "notifications": true,
	"orgs": true, "pricing": true, "security": true, "settings": true,
	"site": true, "sponsors": true, "topics": true,
}

// FindRepos lists every repository named in text by a github.com/owner/name
// link or an api.github.com/repos/owner/name address, once each ignoring
// case, in the order first seen (FR-006).
func FindRepos(text string) []Repo {
	var found []Repo
	for _, m := range githubMention.FindAllStringSubmatch(text, -1) {
		repo, ok := mentioned(strings.ToLower(m[1]), strings.Split(strings.TrimPrefix(m[2], pathSep), pathSep))
		if ok && !ContainsRepo(found, repo) {
			found = append(found, repo)
		}
	}
	return found
}

func mentioned(label string, segments []string) (Repo, bool) {
	switch label {
	case webLabel, wwwLabel:
	case apiLabel:
		if len(segments) < 3 || !strings.EqualFold(segments[0], apiReposAt) {
			return Repo{}, false
		}
		segments = segments[1:]
	default:
		return Repo{}, false
	}
	if notOwners[strings.ToLower(segments[0])] {
		return Repo{}, false
	}
	repo, err := ParseRepo(segments[0] + pathSep + segments[1])
	return repo, err == nil
}

// ContainsRepo reports whether repos holds a repo the same as r.
func ContainsRepo(repos []Repo, r Repo) bool {
	for _, have := range repos {
		if have.Same(r) {
			return true
		}
	}
	return false
}

// PreTicked reports which found repos start ticked (FR-007): the only one
// found, else each whose name with punctuation removed equals the site's last
// path segment or the first label of its host.
func PreTicked(site Address, found []Repo) []Repo {
	if len(found) == 1 {
		return found
	}
	wanted := map[string]bool{bare(site.FirstLabel()): true}
	if seg := site.LastSegment(); seg != "" {
		wanted[bare(seg)] = true
	}
	var ticked []Repo
	for _, r := range found {
		if wanted[bare(r.Name)] {
			ticked = append(ticked, r)
		}
	}
	return ticked
}

// bare lower-cases s and drops everything but letters and digits.
func bare(s string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(s) {
		if unicode.IsLetter(c) || unicode.IsDigit(c) {
			b.WriteRune(c)
		}
	}
	return b.String()
}
