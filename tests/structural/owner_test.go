package structural

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// authorAccount is the author's own account name. Visitron serves any owner,
// so nothing it ships may name the author's account (Amendment 11): not as a
// GoatCounter site, not as an example, not in a word of help.
const authorAccount = "oernster"

// authorAccountAllowed is where the name may stand, each with why. The module
// path in an import is not text the program uses, so imports are passed over.
var authorAccountAllowed = map[string]string{
	// Where Visitron's own releases are published, which the update check
	// asks (FR-075): the publisher, not the owner.
	"internal/product/product.go": "Visitron's own repository",
}

// frontendFixtures are page files that only tests load.
var frontendFixtures = map[string]bool{"bridge-fake.ts": true, "test-setup.ts": true}

// TestNothingShippedNamesTheAuthorsAccount holds every string literal in the
// Go source and every page file to it.
//
// Proved by planting: putting the account back into the repository error in
// internal/domain/repo.go fails it by name.
func TestNothingShippedNamesTheAuthorsAccount(t *testing.T) {
	root := repoRoot(t)
	for _, path := range goFiles(t) {
		rel := relativeTo(root, path)
		if strings.HasSuffix(path, "_test.go") || authorAccountAllowed[rel] != "" {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", rel, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			if _, ok := node.(*ast.ImportSpec); ok {
				return false
			}
			literal, ok := node.(*ast.BasicLit)
			if ok && literal.Kind == token.STRING && strings.Contains(strings.ToLower(literal.Value), authorAccount) {
				t.Errorf("%s names the author's account in %s: Visitron serves any owner", rel, literal.Value)
			}
			return true
		})
	}
	for _, path := range frontendFiles(t) {
		name := filepath.Base(path)
		if strings.Contains(name, ".test.") || frontendFixtures[name] {
			continue
		}
		if strings.Contains(strings.ToLower(readFile(t, path)), authorAccount) {
			t.Errorf("%s names the author's account: Visitron serves any owner", relativeTo(root, path))
		}
	}
}
