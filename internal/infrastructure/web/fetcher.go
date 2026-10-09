package web

import "context"

// PageLimit is the most of one page the crawl reads: 2 MiB (FR-005).
const PageLimit = 2 << 20

// Fetcher reads website pages for the crawl.
type Fetcher struct {
	client *Client
}

// NewFetcher builds the crawl's page reader.
func NewFetcher(client *Client) *Fetcher { return &Fetcher{client: client} }

// Fetch answers the text of the page at url; a page that is not a success is
// an error, so the crawl skips it.
func (f *Fetcher) Fetch(ctx context.Context, url string) (string, error) {
	resp, err := f.client.Get(ctx, url, nil, PageLimit)
	if err != nil {
		return "", err
	}
	if !resp.OK() {
		return "", resp.StatusError(url)
	}
	return string(resp.Body), nil
}
