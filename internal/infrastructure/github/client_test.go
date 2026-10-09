package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/oernster/visitron/internal/application"
	"github.com/oernster/visitron/internal/domain"
	"github.com/oernster/visitron/internal/infrastructure/web"
)

var _ application.Releases = (*Client)(nil)

var sym = domain.Repo{Owner: "oernster", Name: "SymDiary"}

// secrets answers a fixed token.
type secrets struct {
	token string
	err   error
}

func (s secrets) Get(application.Secret) (string, error) { return s.token, s.err }
func (s secrets) Set(application.Secret, string) error   { return nil }
func (s secrets) Delete(application.Secret) error        { return nil }

func client(t *testing.T, sec secrets, h http.HandlerFunc) *Client {
	t.Helper()
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)
	return NewClient(web.NewClient(), sec, s.URL)
}

func TestFilesFollowsPages(t *testing.T) {
	t.Parallel()
	var auth []string
	var c *Client
	c = client(t, secrets{token: "tok"}, func(w http.ResponseWriter, r *http.Request) {
		auth = append(auth, r.Header.Get("Authorization"))
		if r.URL.Query().Get("page") == "" {
			w.Header().Set("Link", fmt.Sprintf(`<%s/repos/oernster/SymDiary/releases?page=2>; rel="next", <x>; rel="last"`, c.base))
			_, _ = w.Write([]byte(`[{"tag_name":"v2","assets":[{"name":"SymDiary.dmg","download_count":3}]}]`))
			return
		}
		_, _ = w.Write([]byte(`[{"tag_name":"v1","assets":[{"name":"SymDiarySetup.exe","download_count":5},{"name":"symdiary.flatpak","download_count":0}]}]`))
	})
	files, _, err := c.Files(context.Background(), sym)
	if err != nil || len(files) != 3 || files[0].Release != "v2" || files[1].Raw != 5 || files[0].Repo != sym {
		t.Fatalf("files %+v err %v", files, err)
	}
	if len(auth) != 2 || auth[0] != "Bearer tok" {
		t.Errorf("authorisation sent %v", auth)
	}
}

func TestAnonymousWithoutToken(t *testing.T) {
	t.Parallel()
	c := client(t, secrets{}, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("User-Agent") == "" {
			w.WriteHeader(http.StatusBadRequest)
		}
		_, _ = w.Write([]byte(`[]`))
	})
	if files, _, err := c.Files(context.Background(), sym); err != nil || len(files) != 0 {
		t.Errorf("files %v err %v", files, err)
	}
}

func TestRateLimit(t *testing.T) {
	t.Parallel()
	for _, status := range []int{http.StatusForbidden, http.StatusTooManyRequests} {
		c := client(t, secrets{}, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set(remainingHeader, "0")
			w.Header().Set(resetHeader, "1791578438")
			w.WriteHeader(status)
		})
		_, reset, err := c.Files(context.Background(), sym)
		if !errors.Is(err, application.ErrRateLimited) || !reset.Equal(time.Unix(1791578438, 0)) {
			t.Errorf("%d: reset %v err %v", status, reset, err)
		}
	}
}

func TestForbiddenIsNotAlwaysTheLimit(t *testing.T) {
	t.Parallel()
	cases := []map[string]string{
		{remainingHeader: "12"},
		{remainingHeader: "0", resetHeader: "soon"},
	}
	for _, h := range cases {
		c := client(t, secrets{}, func(w http.ResponseWriter, _ *http.Request) {
			for k, v := range h {
				w.Header().Set(k, v)
			}
			w.WriteHeader(http.StatusForbidden)
		})
		if _, _, err := c.Files(context.Background(), sym); err == nil || errors.Is(err, application.ErrRateLimited) {
			t.Errorf("%v: err %v", h, err)
		}
	}
}

func TestFilesFaults(t *testing.T) {
	t.Parallel()
	bad := client(t, secrets{}, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{`)) })
	if _, _, err := bad.Files(context.Background(), sym); err == nil || !strings.Contains(err.Error(), "SymDiary") {
		t.Errorf("bad JSON: %v", err)
	}
	gone := client(t, secrets{}, func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })
	if _, _, err := gone.Files(context.Background(), sym); err == nil {
		t.Error("a 404 was read as no releases")
	}
	locked := client(t, secrets{err: errors.New("locked")}, func(http.ResponseWriter, *http.Request) {})
	if _, _, err := locked.Files(context.Background(), sym); err == nil {
		t.Error("a Credential Manager fault was ignored")
	}
	if _, err := locked.Exists(context.Background(), sym); err == nil {
		t.Error("a Credential Manager fault was ignored by Exists")
	}
	down := NewClient(web.NewClient(), secrets{}, "http://127.0.0.1:1")
	if _, _, err := down.Files(context.Background(), sym); err == nil {
		t.Error("an unreachable GitHub was read")
	}
	if _, err := down.Exists(context.Background(), sym); err == nil {
		t.Error("Exists read an unreachable GitHub")
	}
	if err := down.Verify(context.Background(), "x"); err == nil {
		t.Error("Verify read an unreachable GitHub")
	}
}

func TestExists(t *testing.T) {
	t.Parallel()
	c := client(t, secrets{}, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/oernster/SymDiary":
			_, _ = w.Write([]byte(`{}`))
		case "/repos/oernster/Broken":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	})
	if ok, err := c.Exists(context.Background(), sym); !ok || err != nil {
		t.Errorf("existing: %v %v", ok, err)
	}
	if ok, err := c.Exists(context.Background(), domain.Repo{Owner: "oernster", Name: "Nope"}); ok || err != nil {
		t.Errorf("missing: %v %v", ok, err)
	}
	if _, err := c.Exists(context.Background(), domain.Repo{Owner: "oernster", Name: "Broken"}); err == nil {
		t.Error("a server error read as an answer")
	}
}

func TestVerify(t *testing.T) {
	t.Parallel()
	c := client(t, secrets{}, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good" {
			w.WriteHeader(http.StatusUnauthorized)
		}
	})
	if err := c.Verify(context.Background(), "good"); err != nil {
		t.Errorf("good token: %v", err)
	}
	if err := c.Verify(context.Background(), "bad"); err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("bad token: %v", err)
	}
}
