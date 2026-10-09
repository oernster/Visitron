package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oernster/visitron/internal/application"
)

var _ application.Fetcher = (*Fetcher)(nil)

func server(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)
	return s
}

func TestGetSendsHeadersAndReads(t *testing.T) {
	t.Parallel()
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("auth=" + r.Header.Get("Authorization")))
	})
	resp, err := NewClient().Get(context.Background(), s.URL, map[string]string{"Authorization": "Bearer k"}, 100)
	if err != nil || !resp.OK() || string(resp.Body) != "auth=Bearer k" {
		t.Errorf("resp %+v err %v", resp, err)
	}
}

func TestTooLargeIsRefused(t *testing.T) {
	t.Parallel()
	s := server(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 11)))
	})
	if _, err := NewClient().Get(context.Background(), s.URL, nil, 10); !errors.Is(err, ErrTooLarge) {
		t.Errorf("err = %v", err)
	}
	if resp, err := NewClient().Get(context.Background(), s.URL, nil, 11); err != nil || len(resp.Body) != 11 {
		t.Errorf("exactly at the limit: %v", err)
	}
}

func TestFetcher(t *testing.T) {
	t.Parallel()
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("<a href=x>"))
	})
	f := NewFetcher(NewClient())
	if body, err := f.Fetch(context.Background(), s.URL+"/"); err != nil || body != "<a href=x>" {
		t.Errorf("page %q %v", body, err)
	}
	if _, err := f.Fetch(context.Background(), s.URL+"/missing"); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("missing page: %v", err)
	}
	if _, err := f.Fetch(context.Background(), "http://127.0.0.1:1/"); err == nil {
		t.Error("an unreachable host was read")
	}
	if _, err := f.Fetch(context.Background(), "://bad"); err == nil {
		t.Error("an unreadable URL was fetched")
	}
}

func TestBrokenBody(t *testing.T) {
	t.Parallel()
	s := server(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = w.Write([]byte("short"))
	})
	if _, err := NewClient().Get(context.Background(), s.URL, nil, 1000); err == nil {
		t.Error("a body cut short was accepted")
	}
}
