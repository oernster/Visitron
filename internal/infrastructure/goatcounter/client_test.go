package goatcounter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"visitron/internal/application"
	"visitron/internal/domain"
	"visitron/internal/infrastructure/web"
)

var _ application.PageLoads = (*Client)(nil)

var (
	oct8    = domain.Day{Year: 2026, Month: 10, Date: 8}
	oct9    = domain.Day{Year: 2026, Month: 10, Date: 9}
	someone = mustSite("someone")
)

func mustSite(code string) domain.GoatCounterSite {
	site, err := domain.ParseGoatCounterSite(code)
	if err != nil {
		panic(err)
	}
	return site
}

// at answers every site's address as base, a local test server.
func at(base string) func(domain.GoatCounterSite) string {
	return func(domain.GoatCounterSite) string { return base }
}

func client(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)
	return NewClient(web.NewClient(), at(s.URL))
}

func TestTheSiteNamesTheAddress(t *testing.T) {
	t.Parallel()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(s.Close)
	var asked []string
	c := NewClient(web.NewClient(), func(site domain.GoatCounterSite) string {
		asked = append(asked, site.URL())
		return s.URL
	})
	_ = c.Verify(context.Background(), mustSite("other"), "k")
	if len(asked) != 1 || asked[0] != "https://other.goatcounter.com" {
		t.Errorf("addresses asked for %v; want the site handed in", asked)
	}
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
				{"path":"example.org/","path_id":1,"stats":[{"day":"2026-10-07","daily":9},{"day":"2026-10-08","daily":2},{"day":"2026-10-10","daily":5}]},
				{"path":"click","path_id":2,"event":true,"stats":[{"day":"2026-10-08","daily":4}]}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"more":false,"hits":[
			{"path":"/example.com/App/","path_id":3,"stats":[{"day":"2026-10-09","daily":3},{"day":"2026-10-09","daily":0}]}]}`))
	})
	got, err := c.Daily(context.Background(), someone, "key", oct8, oct9)
	if err != nil {
		t.Fatal(err)
	}
	want := []application.PathDay{
		{Path: "example.org/", Day: oct8, Count: 2},
		{Path: "example.com/App/", Day: oct9, Count: 3},
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
	if _, err := c.Daily(context.Background(), someone, "k", oct8, oct9); err != nil || calls != 1 {
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
		if _, err := client(t, h).Daily(context.Background(), someone, "k", oct8, oct9); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := NewClient(web.NewClient(), at("http://127.0.0.1:1")).Daily(context.Background(), someone, "k", oct8, oct9); err == nil {
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
	if err := c.Verify(context.Background(), someone, "good"); err != nil {
		t.Errorf("good key: %v", err)
	}
	if err := c.Verify(context.Background(), someone, "bad"); err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("bad key: %v", err)
	}
}
