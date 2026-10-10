package update

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/oernster/visitron/internal/application"
	"github.com/oernster/visitron/internal/infrastructure/web"
)

var _ application.ReleaseSource = (*Source)(nil)

func source(t *testing.T, h http.HandlerFunc) *Source {
	t.Helper()
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)
	return NewAt(web.NewClient(), s.URL)
}

const release = `{"tag_name":"v1.2.0","html_url":"https://github.com/oernster/Visitron/releases/tag/v1.2.0",
"assets":[{"name":"VisitronSetup.exe","browser_download_url":"https://github.com/oernster/Visitron/releases/download/v1.2.0/VisitronSetup.exe"},
{"name":"","browser_download_url":"https://github.com/x"},
{"name":"elsewhere.exe","browser_download_url":"https://elsewhere.example/VisitronSetup.exe"}]}`

func TestLatestAsksTheRightPlaceUnauthenticated(t *testing.T) {
	t.Parallel()
	s := source(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != LatestPath || r.Header.Get("Accept") != acceptHeader || r.Header.Get("Authorization") != "" {
			t.Errorf("asked %s with Accept %q, Authorization %q", r.URL.Path, r.Header.Get("Accept"), r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(release))
	})
	got, err := s.Latest(context.Background())
	if err != nil || got.Tag != "v1.2.0" || !strings.HasPrefix(got.Page, releaseHost) {
		t.Fatalf("release %+v, %v", got, err)
	}
	if len(got.Assets) != 1 || got.Assets[0].Name != "VisitronSetup.exe" {
		t.Errorf("assets %+v: a nameless file or one off GitHub was kept", got.Assets)
	}
}

func TestTheEndpointIsVisitronsOwn(t *testing.T) {
	t.Parallel()
	if got := New(web.NewClient()).address; got != "https://api.github.com/repos/oernster/Visitron/releases/latest" {
		t.Errorf("address = %s", got)
	}
}

func TestEveryFailureIsAnError(t *testing.T) {
	t.Parallel()
	answers := map[string]http.HandlerFunc{
		"refused":  func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) },
		"not JSON": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html>")) },
		"no tag": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"html_url":"https://github.com/x"}`))
		},
		"off GitHub": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"tag_name":"v1","html_url":"https://elsewhere.example/x"}`))
		},
		"too large": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(make([]byte, answerCap+1)) },
	}
	for name, h := range answers {
		if _, err := source(t, h).Latest(context.Background()); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	unreachable := NewAt(web.NewClient(), "http://127.0.0.1:0")
	if _, err := unreachable.Latest(context.Background()); err == nil {
		t.Error("an unreachable host: no error")
	}
}

func TestNoPublishedReleaseIsNamed(t *testing.T) {
	t.Parallel()
	s := source(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })
	if _, err := s.Latest(context.Background()); !errors.Is(err, application.ErrNoRelease) {
		t.Errorf("err = %v", err)
	}
}

func TestTheCheckGivesUpAfterItsLimit(t *testing.T) {
	t.Parallel()
	release := make(chan struct{})
	s := source(t, func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	})
	defer close(release)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := s.Latest(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v", err)
	}
}
