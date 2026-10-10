// Package goatcounter reads page loads from the owner's GoatCounter site with
// their API key. GoatCounter reports each page's visitors per day, which is
// the figure Visitron shows as page loads (Amendment 2). The site is the one
// named in Settings (Amendment 11).
package goatcounter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"visitron/internal/application"
	"visitron/internal/domain"
	"visitron/internal/infrastructure/web"
)

const (
	hitsPath      = "/api/v0/stats/hits"
	locationsPath = "/api/v0/stats/locations"
	mePath        = "/api/v0/me"
	pageSize      = 100 // the most rows GoatCounter returns per request
	bodyLimit     = 32 << 20
	dateForm      = "2006-01-02"
	idSep         = ","
	pathSep       = "/"
)

// Client reads GoatCounter.
type Client struct {
	web     *web.Client
	address func(domain.GoatCounterSite) string
}

// NewClient builds a GoatCounter reader. address answers where a site's API
// sits: domain.GoatCounterSite.URL in the application, a local server in a
// test.
func NewClient(w *web.Client, address func(domain.GoatCounterSite) string) *Client {
	return &Client{web: w, address: address}
}

type hitsResponse struct {
	Hits []hit `json:"hits"`
	More bool  `json:"more"`
}

// hit is one path with its visitors per day.
type hit struct {
	Path   string `json:"path"`
	PathID int64  `json:"path_id"`
	Event  bool   `json:"event"`
	Stats  []struct {
		Day   string `json:"day"`
		Daily int    `json:"daily"`
	} `json:"stats"`
}

type locationsResponse struct {
	Stats []struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Count int    `json:"count"`
	} `json:"stats"`
	More bool `json:"more"`
}

// window is the start and end GoatCounter is asked for. Whether it includes
// its end date is not stated, so it is asked for one day more; whatever falls
// outside the days asked for is dropped by the reader.
func window(first, last domain.Day) url.Values {
	end := time.Date(last.Year, time.Month(last.Month), last.Date+1, 0, 0, 0, 0, time.UTC)
	q := url.Values{}
	q.Set("start", first.String())
	q.Set("end", end.Format(dateForm))
	q.Set("limit", strconv.Itoa(pageSize))
	return q
}

// hits reads every page path site recorded from first to last, page by page,
// leaving out events: GoatCounter pages paths by excluding those already seen.
func (c *Client) hits(ctx context.Context, site domain.GoatCounterSite, key string, first, last domain.Day) ([]hit, error) {
	var out []hit
	var seen []string
	for {
		q := window(first, last)
		q.Set("group", "day")
		if len(seen) > 0 {
			q.Set("exclude_paths", strings.Join(seen, idSep))
		}
		var page hitsResponse
		if err := c.get(ctx, site, key, hitsPath+"?"+q.Encode(), &page); err != nil {
			return nil, err
		}
		for _, h := range page.Hits {
			seen = append(seen, strconv.FormatInt(h.PathID, 10))
			if !h.Event {
				out = append(out, h)
			}
		}
		if !page.More || len(page.Hits) == 0 {
			return out, nil
		}
	}
}

// Daily lists each path's visitors per day from first to last inclusive.
func (c *Client) Daily(ctx context.Context, site domain.GoatCounterSite, key string, first, last domain.Day) ([]application.PathDay, error) {
	hits, err := c.hits(ctx, site, key, first, last)
	if err != nil {
		return nil, err
	}
	var out []application.PathDay
	for _, h := range hits {
		for _, s := range h.Stats {
			day, err := time.Parse(dateForm, s.Day)
			if err != nil {
				return nil, fmt.Errorf("GoatCounter gave the day %q: %w", s.Day, err)
			}
			d := application.DayOf(day)
			if s.Daily > 0 && !d.Before(first) && !last.Before(d) {
				out = append(out, application.PathDay{Path: strings.TrimPrefix(h.Path, pathSep), Day: d, Count: s.Daily})
			}
		}
	}
	return out, nil
}

// Paths lists every page path site recorded from first to last inclusive,
// each with the id the other reports filter by.
func (c *Client) Paths(ctx context.Context, site domain.GoatCounterSite, key string, first, last domain.Day) ([]application.SitePath, error) {
	hits, err := c.hits(ctx, site, key, first, last)
	if err != nil {
		return nil, err
	}
	out := make([]application.SitePath, len(hits))
	for i, h := range hits {
		out[i] = application.SitePath{Path: strings.TrimPrefix(h.Path, pathSep), ID: h.PathID}
	}
	return out, nil
}

// Countries lists the visitors by country to the paths with ids from first to
// last inclusive, page by page by offset until GoatCounter has no more.
func (c *Client) Countries(ctx context.Context, site domain.GoatCounterSite, key string, first, last domain.Day,
	ids []int64) ([]domain.CountryCount, error) {
	filter := make([]string, len(ids))
	for i, id := range ids {
		filter[i] = strconv.FormatInt(id, 10)
	}
	out := []domain.CountryCount{}
	for {
		q := window(first, last)
		q.Set("include_paths", strings.Join(filter, idSep))
		q.Set("offset", strconv.Itoa(len(out)))
		var page locationsResponse
		if err := c.get(ctx, site, key, locationsPath+"?"+q.Encode(), &page); err != nil {
			return nil, err
		}
		for _, s := range page.Stats {
			out = append(out, domain.CountryCount{Code: s.ID, Name: s.Name, Count: s.Count})
		}
		if !page.More || len(page.Stats) == 0 {
			return out, nil
		}
	}
}

// Verify tries key once against the site's own details (FR-061).
func (c *Client) Verify(ctx context.Context, site domain.GoatCounterSite, key string) error {
	var ignored json.RawMessage
	return c.get(ctx, site, key, mePath, &ignored)
}

func (c *Client) get(ctx context.Context, site domain.GoatCounterSite, key, path string, into any) error {
	resp, err := c.web.Get(ctx, c.address(site)+path, map[string]string{
		"Authorization": "Bearer " + key,
		"Content-Type":  "application/json",
	}, bodyLimit)
	if err != nil {
		return err
	}
	if !resp.OK() {
		return resp.StatusError("GoatCounter")
	}
	if err := json.Unmarshal(resp.Body, into); err != nil {
		return fmt.Errorf("reading GoatCounter's answer: %w", err)
	}
	return nil
}
