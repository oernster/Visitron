// Package domain holds Visitron's rules: what a site address is, which
// repositories a site names, which website owns a page load and how downloads
// are counted. It reads no clock, opens no file and touches no network.
package domain

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Refusals for an entry that cannot become a site address (FR-002).
var (
	ErrEmptyAddress = errors.New("enter a website address")
	ErrScheme       = errors.New("a website address starts with http:// or https://")
	ErrNoHost       = errors.New("the address names no website")
	ErrHostNotDNS   = errors.New("the website name needs a dot, as in symdiary.com")
)

const (
	defaultScheme = "https://"
	schemeMark    = "://"
	pathSep       = "/"
	labelSep      = "."
)

// Address is a site address: a lower-case host plus a path that begins and
// ends with "/". The path keeps its case because GitHub Pages paths are case
// sensitive.
type Address struct {
	Host string
	Path string
}

// Normalise reduces an entry to a site address (FR-001), refusing what cannot
// be one (FR-002). A missing scheme means https; a query, a fragment or a
// final file name is dropped.
func Normalise(entry string) (Address, error) {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return Address{}, ErrEmptyAddress
	}
	if !strings.Contains(entry, schemeMark) {
		entry = defaultScheme + entry
	}
	u, err := url.Parse(entry)
	if err != nil {
		return Address{}, fmt.Errorf("%w: %v", ErrNoHost, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return Address{}, ErrScheme
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return Address{}, ErrNoHost
	}
	if !strings.Contains(strings.Trim(host, labelSep), labelSep) {
		return Address{}, ErrHostNotDNS
	}
	return Address{Host: host, Path: directory(u.Path)}, nil
}

// directory turns a URL path into the folder it names: a final segment with a
// dot in it is a file, so it is dropped; the result always ends with "/".
func directory(path string) string {
	segments := strings.Split(strings.Trim(path, pathSep), pathSep)
	if last := segments[len(segments)-1]; strings.Contains(last, labelSep) {
		segments = segments[:len(segments)-1]
	}
	kept := strings.Join(segments, pathSep)
	if kept == "" {
		return pathSep
	}
	return pathSep + kept + pathSep
}

// Key is the address as GoatCounter records a page under it: host plus path,
// with no scheme, such as "ernster.dev/WhatDay/".
func (a Address) Key() string { return a.Host + a.Path }

// URL is the address the crawler fetches and the window shows.
func (a Address) URL() string { return defaultScheme + a.Key() }

// LastSegment is the final folder of the path, empty at the root of a host.
func (a Address) LastSegment() string {
	trimmed := strings.Trim(a.Path, pathSep)
	if trimmed == "" {
		return ""
	}
	return trimmed[strings.LastIndex(trimmed, pathSep)+1:]
}

// FirstLabel is the first label of the host: "symdiary" for symdiary.com.
func (a Address) FirstLabel() string {
	return strings.SplitN(a.Host, labelSep, 2)[0]
}

// Contains reports whether target lies on this site: the same host, under the
// same path.
func (a Address) Contains(target Address) bool {
	return a.Host == target.Host && strings.HasPrefix(target.Path, a.Path)
}
