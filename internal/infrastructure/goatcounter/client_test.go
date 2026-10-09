package goatcounter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/oernster/visitron/internal/application"
	"github.com/oernster/visitron/internal/domain"
	"github.com/oernster/visitron/internal/infrastructure/web"
)

var _ application.PageLoads = (*Client)(nil)

var (
	oct8 = domain.Day{Year: 2026, Month: 10, Date: 8}
	oct9 = domain.Day{Year: 2026, Month: 10, Date: 9}
)

func client(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)
	return NewClient(web.NewClient(), s.URL)
}

func TestDailyPagesThroughPaths(t *testing.T) {
	t.Parallel()
	var queries []string
	c := client(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		queries = append(queries, r.URL.RawQuery)
		if r.URL.Query().Get("exclude_paths") == "" {
			_, _ = w.Write([]byte(`{"more":true,"hits":[
				{"path":"symdiary.com/","path_id":1,"stats":[{"day":"2026-10-07","daily":9},{"day":"2026-10-08","daily":2},{"day":"2026-10-10","daily":5}]},
				{"path":"click","path_id":2,"event":true,"stats":[{"day":"2026-10-08","daily":4}]}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"more":false,"hits":[
			{"path":"/ernster.dev/WhatDay/","path_id":3,"stats":[{"day":"2026-10-09","daily":3},{"day":"2026-10-09","daily":0}]}]}`))
	})
	got, err := c.Daily(context.Background(), "key", oct8, oct9)
	if err != nil {
		t.Fatal(err)
	}
	want := []application.PathDay{
		{Path: "symdiary.com/", Day: oct8, Count: 2},
		{Path: "ernster.dev/WhatDay/", Day: oct9, Count: 3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Daily = %+v; want %+v", got, want)
	}
	if len(queries) != 2 || !strings.Contains(queries[0], "start=2026-10-08") ||
		!strings.Contains(queries[0], "end=2026-10-10") || !strings.Contains(queries[1], "exclude_paths=1%2C2") {
		t.Errorf("queries %v", queries)
	}
}

func TestDailyStopsOnAnEmptyPage(t *testing.T) {
	t.Parallel()
	calls := 0
	c := client(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"more":true,"hits":[]}`))
	})
	if _, err := c.Daily(context.Background(), "k", oct8, oct9); err != nil || calls != 1 {
		t.Errorf("calls %d err %v", calls, err)
	}
}

func TestDailyFaults(t *testing.T) {
	t.Parallel()
	cases := map[string]http.HandlerFunc{
		"refused": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) },
		"garbage": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{`)) },
		"bad day": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"hits":[{"path":"a","path_id":1,"stats":[{"day":"9th","daily":1}]}]}`))
		},
	}
	for name, h := range cases {
		if _, err := client(t, h).Daily(context.Background(), "k", oct8, oct9); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := NewClient(web.NewClient(), "http://127.0.0.1:1").Daily(context.Background(), "k", oct8, oct9); err == nil {
		t.Error("an unreachable GoatCounter was read")
	}
}

func TestVerify(t *testing.T) {
	t.Parallel()
	c := client(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != mePath || r.Header.Get("Authorization") != "Bearer good" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"user":{}}`))
	})
	if err := c.Verify(context.Background(), "good"); err != nil {
		t.Errorf("good key: %v", err)
	}
	if err := c.Verify(context.Background(), "bad"); err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("bad key: %v", err)
	}
}
