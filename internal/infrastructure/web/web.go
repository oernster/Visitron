// Package web is the one place Visitron makes an HTTP request, so every
// caller gets the same timeout, the same cap on what it reads and the same
// treatment of what comes back.
package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RequestTimeout is the longest one request may take (FR-005).
const RequestTimeout = 20 * time.Second

// ErrTooLarge refuses a response bigger than its caller allows. The size is
// foreign input, so it is never trusted to set aside memory (robustness 5).
var ErrTooLarge = errors.New("the response was larger than allowed")

// Response is a read response.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

// Client makes capped GET requests.
type Client struct {
	http *http.Client
}

// NewClient builds a client with RequestTimeout on every request.
func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: RequestTimeout}}
}

// Get fetches url with headers, reading at most limit bytes of the body; a
// longer body is ErrTooLarge.
func (c *Client) Get(ctx context.Context, url string, headers map[string]string, limit int64) (Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Response{}, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return Response{}, fmt.Errorf("reading %s: %w", url, err)
	}
	if int64(len(body)) > limit {
		return Response{}, fmt.Errorf("%w: %s over %d bytes", ErrTooLarge, url, limit)
	}
	return Response{Status: resp.StatusCode, Header: resp.Header, Body: body}, nil
}

// OK reports whether the status is a 2xx success.
func (r Response) OK() bool {
	return r.Status >= http.StatusOK && r.Status < http.StatusMultipleChoices
}

// StatusError names a response that was not a success.
func (r Response) StatusError(what string) error {
	return fmt.Errorf("%s answered %d %s", what, r.Status, http.StatusText(r.Status))
}
