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
	hitsPath  = "/api/v0/stats/hits"
	mePath    = "/api/v0/me"
	pageSize  = 100 // the most paths GoatCounter returns per request
	bodyLimit = 32 << 20
	dateForm  = "2006-01-02"
	idSep     = ","
	pathSep   = "/"
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
	Hits []struct {
		Path   string `json:"path"`
		PathID int64  `json:"path_id"`
		Event  bool   `json:"event"`
		Stats  []struct {
			Day   string `json:"day"`
			Daily int    `json:"daily"`
		} `json:"stats"`
	} `json:"hits"`
	More bool `json:"more"`
}

// Daily lists each path's visitors per day from first to last inclusive,
// page by page. Whether GoatCounter includes its end date is not stated, so
// it is asked for one day more; anything outside the days asked for is
// dropped.
func (c *Client) Daily(ctx context.Context, site domain.GoatCounterSite, key string, first, last domain.Day) ([]application.PathDay, error) {
	end := time.Date(last.Year, time.Month(last.Month), last.Date+1, 0, 0, 0, 0, time.UTC)
	var out []application.PathDay
	var seen []string
	for {
		q := url.Values{}
		q.Set("start", first.String())
		q.Set("end", end.Format(dateForm))
		q.Set("group", "day")
		q.Set("limit", strconv.Itoa(pageSize))
		if len(seen) > 0 {
			q.Set("exclude_paths", strings.Join(seen, idSep))
		}
		var page hitsResponse
		if err := c.get(ctx, site, key, hitsPath+"?"+q.Encode(), &page); err != nil {
			return nil, err
		}
		for _, h := range page.Hits {
			seen = append(seen, strconv.FormatInt(h.PathID, 10))
			if h.Event {
				continue
			}
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
		if !page.More || len(page.Hits) == 0 {
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
