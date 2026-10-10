package structural

// What checks the setup program's page.
//
// The application's page has a build behind it: eslint, then tsc, then Vite, plus
// 100 tests over the components. The setup program's page deliberately has none,
// because that is what makes the setup program a single file with no build step
// of its own. So nothing resolves a name in it. It reaches into the document by
// string, `$('licence-open')`, then reads fields off a state object Go marshals,
// `state.recordFile`. Rename the id in the HTML or the field in the Go struct
// and the page still parses, still passes every other check in this suite and
// gives the reader a screen that does nothing.
//
// The house has been bitten by exactly this shape before: a product name typed
// into a setup page in sixteen places survived a rename that reached every other
// surface; setup went on announcing a product that no longer existed.

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// setupPageDir, in boundary_test.go, is the directory these read.
//
// setupMarkup is the document every script reaches into.
const setupMarkup = "index.html"

// setupStateReaders are the scripts handed the Go state. setup-scroll.js is
// deliberately not one of them: its own `state` is the auto-scroll machine's,
// whose fields are the page's own and answer to nothing across the wire.
var setupStateReaders = []string{"setup-routes.js", "setup-shell.js"}

// setupWire pairs each name the page reads a field off with the struct that
// states what those fields are. The variable name is the page's only clue to
// which shape it is holding, so the pairing is written down rather than guessed.
var setupWire = map[string]string{
	"state": "StateDTO",
	"held":  "LicenceDTO",
}

var (
	elementLookup = regexp.MustCompile(`\$\('([A-Za-z0-9_-]+)'\)`)
	markupID      = regexp.MustCompile(`id="([A-Za-z0-9_-]+)"`)
)

// setupScripts answers every script the page loads, newest reading off disk so
// a script added without a thought is still checked.
func setupScripts(t *testing.T, root string) map[string]string {
	t.Helper()
	dir := filepath.Join(root, setupPageDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.ToSlash(setupPageDir), err)
	}
	out := map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".js") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("reading %s: %v", entry.Name(), err)
		}
		out[entry.Name()] = string(raw)
	}
	if len(out) == 0 {
		t.Fatalf("no scripts found in %s, the walk is wrong", filepath.ToSlash(setupPageDir))
	}
	return out
}

func TestTheSetupPageLooksUpElementsThatAreThere(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, setupPageDir, setupMarkup))
	if err != nil {
		t.Fatalf("reading %s: %v", setupMarkup, err)
	}
	present := map[string]bool{}
	for _, match := range markupID.FindAllStringSubmatch(string(raw), -1) {
		present[match[1]] = true
	}
	if len(present) == 0 {
		t.Fatalf("%s declares no ids, the pattern is wrong", setupMarkup)
	}

	for name, source := range setupScripts(t, root) {
		var missing []string
		for _, match := range elementLookup.FindAllStringSubmatch(source, -1) {
			if !present[match[1]] {
				missing = append(missing, match[1])
			}
		}
		sort.Strings(missing)
		for _, id := range missing {
			t.Errorf("%s asks the document for %q, which %s does not declare: the "+
				"lookup answers nothing and the screen it fills does nothing, with "+
				"no error anywhere a reader can see it", name, id, setupMarkup)
		}
	}
}

func TestTheSetupPageReadsFieldsThatActuallyCross(t *testing.T) {
	root := repoRoot(t)
	shapes := dtosIn(t, root, filepath.Join(root, "installer"))
	scripts := setupScripts(t, root)

	holders := make([]string, 0, len(setupWire))
	for holder := range setupWire {
		holders = append(holders, holder)
	}
	sort.Strings(holders)

	for _, holder := range holders {
		shape := setupWire[holder]
		fields, ok := shapes[shape]
		if !ok {
			t.Errorf("setupWire pairs %q with %s, which the setup facade no longer "+
				"declares", holder, shape)
			continue
		}
		known := map[string]bool{}
		for _, field := range fields {
			known[field] = true
		}
		reader := regexp.MustCompile(`\b` + holder + `\.([A-Za-z_][A-Za-z0-9_]*)`)
		for _, name := range setupStateReaders {
			source, ok := scripts[name]
			if !ok {
				t.Errorf("setupStateReaders names %s, which is not in %s",
					name, filepath.ToSlash(setupPageDir))
				continue
			}
			checkReads(t, name, holder, shape, source, reader, known)
		}
	}
}

// checkReads reports every field one script reads off one shape that the shape
// does not send. Kept apart from the loops so each failure reads as a statement
// about that field rather than about the walk that found it.
func checkReads(t *testing.T, name, holder, shape, source string,
	reader *regexp.Regexp, known map[string]bool) {
	t.Helper()
	var missing []string
	for _, match := range reader.FindAllStringSubmatch(source, -1) {
		if !known[match[1]] {
			missing = append(missing, match[1])
		}
	}
	sort.Strings(missing)
	for _, field := range missing {
		t.Errorf("%s reads %s.%s, which %s does not send: the page reads undefined "+
			"and draws a screen that says nothing, while Go and TypeScript both "+
			"stay silent because neither can see the other half", name, holder, field, shape)
	}
}
