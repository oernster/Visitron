package structural

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"

	"visitron/internal/application"
)

// FR-072: About credits every open source work Visitron ships. The credits are
// a list a person wrote, so a dependency added to go.mod or package.json
// without one would ship uncredited with nothing to say so.

// creditFor names the credit that covers each runtime dependency. Adding a
// dependency means adding it here, which is the moment to write its credit.
var creditFor = map[string]string{
	"github.com/wailsapp/wails/v2":  "Wails",
	"github.com/zalando/go-keyring": "go-keyring",
	"golang.org/x/sys":              "Go and golang.org/x/sys",
	"modernc.org/sqlite":            "modernc.org/sqlite",
	"react":                         "React",
	"react-dom":                     "React",
}

// creditedWithoutADependency are the credits no dependency file names, each
// with why it is there.
var creditedWithoutADependency = map[string]string{
	"SQLite":                         "the database modernc.org/sqlite is a translation of",
	"GoatCounter (the service read)": "the service the page loads come from",
}

// directRequire matches a direct requirement in go.mod; an indirect one is
// shipped because a direct one needs it and is credited through that one.
var directRequire = regexp.MustCompile(`(?m)^\s+(\S+)\s+v\S+\s*$`)

// runtimeDependencies answers every direct Go module plus every package the
// page ships. Development packages build or test the page; none ships.
func runtimeDependencies(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	var out []string
	for _, match := range directRequire.FindAllStringSubmatch(readFile(t, filepath.Join(root, "go.mod")), -1) {
		out = append(out, match[1])
	}
	raw, err := os.ReadFile(filepath.Join(root, "frontend", "package.json"))
	if err != nil {
		t.Fatalf("reading package.json: %v", err)
	}
	var manifest struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("reading package.json: %v", err)
	}
	for name := range manifest.Dependencies {
		out = append(out, name)
	}
	sort.Strings(out)
	if len(out) == 0 {
		t.Fatal("no dependencies found, the reading is wrong")
	}
	return out
}

// TestEveryDependencyIsCredited fails on a runtime dependency without a credit
// and on a credit nothing ships.
func TestEveryDependencyIsCredited(t *testing.T) {
	credits := map[string]bool{}
	for _, credit := range application.Credits {
		credits[credit.Work] = true
	}
	claimed := map[string]bool{}
	for _, dependency := range runtimeDependencies(t) {
		work, ok := creditFor[dependency]
		if !ok {
			t.Errorf("%s ships with Visitron and names no credit in creditFor: About must "+
				"credit it (FR-072)", dependency)
			continue
		}
		if !credits[work] {
			t.Errorf("%s is credited as %q, which application.Credits does not hold", dependency, work)
		}
		claimed[work] = true
	}
	for _, credit := range application.Credits {
		if !claimed[credit.Work] && creditedWithoutADependency[credit.Work] == "" {
			t.Errorf("%q is credited but nothing shipped names it: declare why in "+
				"creditedWithoutADependency or remove the credit", credit.Work)
		}
	}
}
