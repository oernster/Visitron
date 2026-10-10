package structural

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Nothing Visitron ships may carry the identity of the person building it
// (Amendment 15): not their account, not their domains, not as a default, an
// example or a word of help. The test names no one either. It learns whose
// identity to look for from the checkout itself: the owner of the origin
// remote.

// originOwner reads the owner off a GitHub origin remote, https or ssh.
var originOwner = regexp.MustCompile(`^(?:https://github\.com/|git@github\.com:)([A-Za-z0-9-]+)/`)

// frontendFixtures are page files that only tests load.
var frontendFixtures = map[string]bool{"bridge-fake.ts": true, "test-setup.ts": true}

// buildersIdentity answers the identifiers to refuse, in lower case.
func buildersIdentity(t *testing.T, root string) []string {
	t.Helper()
	var found []string
	out, err := exec.Command("git", "-C", root, "remote", "get-url", "origin").Output()
	if err == nil {
		if m := originOwner.FindStringSubmatch(strings.TrimSpace(string(out))); m != nil {
			found = append(found, strings.ToLower(m[1]))
		}
	}
	return found
}

// TestNothingShippedNamesTheBuilder holds every string literal in the Go
// source (imports aside) plus every page file to it.
//
// Proved by planting: putting the origin's owner back into the repository
// error in internal/domain/repo.go fails it by name.
func TestNothingShippedNamesTheBuilder(t *testing.T) {
	root := repoRoot(t)
	identity := buildersIdentity(t, root)
	if len(identity) == 0 {
		t.Skip("no GitHub origin: there is no identity to look for")
	}
	named := func(text string) string {
		lower := strings.ToLower(text)
		for _, id := range identity {
			if strings.Contains(lower, id) {
				return id
			}
		}
		return ""
	}
	for _, path := range goFiles(t) {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		rel := relativeTo(root, path)
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", rel, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			if _, ok := node.(*ast.ImportSpec); ok {
				return false
			}
			if literal, ok := node.(*ast.BasicLit); ok && literal.Kind == token.STRING {
				if id := named(literal.Value); id != "" {
					t.Errorf("%s names %q in %s: Visitron serves any owner", rel, id, literal.Value)
				}
			}
			return true
		})
	}
	for _, path := range frontendFiles(t) {
		name := filepath.Base(path)
		if strings.Contains(name, ".test.") || frontendFixtures[name] {
			continue
		}
		if id := named(readFile(t, path)); id != "" {
			t.Errorf("%s names %q: Visitron serves any owner", relativeTo(root, path), id)
		}
	}
}
