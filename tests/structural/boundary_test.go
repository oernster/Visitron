// Package structural enforces the architecture with tests rather than
// convention. Ported from SymDiary's tests/structural, which came from ED
// Voyage Companion's.
//
// Every assertion here has been proved to bite by planting a violation and
// reading the exit code. An assertion never seen to fail is not yet a guard.
package structural

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// lineLimit is the module-size cap. dangerBand is five per cent below it: a
// file landing between them is refactored down to safeLanding rather than left
// one edit away from breaching.
const (
	lineLimit   = 400
	dangerBand  = lineLimit - lineLimit/20
	safeLanding = 350
)

// modulePath prefixes every internal import.
const modulePath = "visitron/"

// frontendSource is the front end's own source, held to the same size rule as
// the Go: a component gathers markup, behaviour and the words on screen in one
// place, which is where growth actually happens.
const frontendSource = "frontend/src"

// compositionRoot names the files allowed to import both application and
// infrastructure. main.go builds the adapters; nothing else may.
var compositionRoot = map[string]bool{"main.go": true}

// forbiddenInDomain names the packages that would make the domain impure.
// net/url is not among them: it parses an address and opens nothing.
var forbiddenInDomain = []string{
	"net", "net/http", "os", "path/filepath", "math/rand",
	"database/sql", "io/ioutil", "os/exec", "log",
}

// forbiddenCallsInDomain names calls that read the wall clock or the global
// random source directly, which would make domain behaviour irreproducible.
var forbiddenCallsInDomain = []string{"time.Now", "rand.Intn", "rand.Float64"}

// repoRoot walks up from the test's directory to the module root.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for range 6 {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("could not find go.mod above the test directory")
	return ""
}

// skippedDirs are the trees the walks never enter: other people's code and
// generated output.
var skippedDirs = map[string]bool{
	"node_modules": true, "dist": true, ".git": true, "build": true, "venv": true,
}

// goFiles returns every Go source file in the repository.
func goFiles(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	var found []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if skippedDirs[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			found = append(found, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking repository: %v", err)
	}
	if len(found) == 0 {
		t.Fatal("no Go files found, the walk is wrong")
	}
	return found
}

// frontendSourceExtensions are the front-end files the size rule governs.
var frontendSourceExtensions = map[string]bool{".ts": true, ".tsx": true, ".css": true}

// frontendFiles returns every front-end source file.
func frontendFiles(t *testing.T) []string {
	t.Helper()
	source := filepath.Join(repoRoot(t), frontendSource)
	var found []string
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil //nolint:nilerr // an unreadable subtree is skipped, not fatal
		}
		if frontendSourceExtensions[strings.ToLower(filepath.Ext(path))] {
			found = append(found, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", source, err)
	}
	if len(found) == 0 {
		t.Fatalf("no front-end source found under %s, the walk is wrong", source)
	}
	return found
}

// importsOf parses a file and returns its import paths.
func importsOf(t *testing.T, path string) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	out := make([]string, 0, len(parsed.Imports))
	for _, item := range parsed.Imports {
		out = append(out, strings.Trim(item.Path.Value, `"`))
	}
	return out
}

// relativeTo answers a path relative to the repository, in slashes.
func relativeTo(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(relative)
}

// layerOf returns which architectural layer a file belongs to.
func layerOf(root, path string) string {
	parts := strings.Split(relativeTo(root, path), "/")
	if len(parts) >= 2 && parts[0] == "internal" {
		return parts[1]
	}
	return ""
}

func TestDomainHasNoOutwardImports(t *testing.T) {
	root := repoRoot(t)
	for _, path := range goFiles(t) {
		if layerOf(root, path) != "domain" {
			continue
		}
		for _, imported := range importsOf(t, path) {
			if !strings.HasPrefix(imported, modulePath) {
				continue
			}
			inner := strings.TrimPrefix(imported, modulePath)
			if !strings.HasPrefix(inner, "internal/domain") {
				t.Errorf("%s imports %s: the domain depends on nothing", path, imported)
			}
		}
	}
}

func TestDomainIsPure(t *testing.T) {
	root := repoRoot(t)
	for _, path := range goFiles(t) {
		if layerOf(root, path) != "domain" || strings.HasSuffix(path, "_test.go") {
			continue
		}
		for _, imported := range importsOf(t, path) {
			for _, banned := range forbiddenInDomain {
				if imported == banned {
					t.Errorf("%s imports %q: the domain performs no IO", path, banned)
				}
			}
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		for _, call := range forbiddenCallsInDomain {
			if strings.Contains(string(raw), call+"(") {
				t.Errorf("%s calls %s: the clock is injected", path, call)
			}
		}
	}
}

func TestApplicationDoesNotImportInfrastructure(t *testing.T) {
	root := repoRoot(t)
	for _, path := range goFiles(t) {
		if layerOf(root, path) != "application" {
			continue
		}
		for _, imported := range importsOf(t, path) {
			if strings.Contains(imported, "internal/infrastructure") ||
				strings.Contains(imported, "wails") {
				t.Errorf("%s imports %s: the application depends on ports only", path, imported)
			}
		}
	}
}

// TestCompositionRootIsWhitelisted keeps the wiring in one place. An adapter
// is exempt: it implements an application port and may use another adapter
// (the GitHub and GoatCounter clients both ride the shared web client), which
// is depending on infrastructure rather than wiring it.
func TestCompositionRootIsWhitelisted(t *testing.T) {
	root := repoRoot(t)
	for _, path := range goFiles(t) {
		if strings.HasSuffix(path, "_test.go") || layerOf(root, path) == "infrastructure" {
			continue
		}
		var application, infrastructure bool
		for _, imported := range importsOf(t, path) {
			application = application || strings.Contains(imported, "internal/application")
			infrastructure = infrastructure || strings.Contains(imported, "internal/infrastructure")
		}
		if !application || !infrastructure {
			continue
		}
		if relative := relativeTo(root, path); !compositionRoot[relative] {
			t.Errorf("%s wires application to infrastructure: only the composition root may", relative)
		}
	}
}

// setupPageDir is the setup program's front end. It has no build step, so the
// files there are the source rather than the output of one.
var setupPageDir = filepath.Join("installer", "frontend", "dist")

// setupPageExtensions are the files there the size rule governs. The mark
// beside them is artwork.
var setupPageExtensions = map[string]bool{".html": true, ".css": true, ".js": true}

// setupPageFiles answers the setup program's own source files, held to the
// size rule for the same reason the application's page is.
func setupPageFiles(t *testing.T) []string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), setupPageDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.ToSlash(setupPageDir), err)
	}
	var found []string
	for _, entry := range entries {
		if entry.IsDir() || !setupPageExtensions[strings.ToLower(filepath.Ext(entry.Name()))] {
			continue
		}
		found = append(found, filepath.Join(dir, entry.Name()))
	}
	if len(found) == 0 {
		t.Fatalf("no source found in %s, the walk is wrong", filepath.ToSlash(setupPageDir))
	}
	return found
}

// sourceFiles is every file the size rule governs.
func sourceFiles(t *testing.T) []string {
	t.Helper()
	files := append(goFiles(t), frontendFiles(t)...)
	return append(files, setupPageFiles(t)...)
}

func TestNoFileExceedsLineLimit(t *testing.T) {
	for _, path := range sourceFiles(t) {
		if count := lineCount(t, path); count > lineLimit {
			t.Errorf("%s has %d lines, over the %d limit", path, count, lineLimit)
		}
	}
}

func TestNoFileInDangerBand(t *testing.T) {
	for _, path := range sourceFiles(t) {
		count := lineCount(t, path)
		if count > dangerBand && count <= lineLimit {
			t.Errorf(
				"%s has %d lines, inside the danger band %d to %d: reduce it to %d or fewer",
				path, count, dangerBand+1, lineLimit, safeLanding,
			)
		}
	}
}

// lineCount counts the lines in a file.
func lineCount(t *testing.T, path string) int {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return strings.Count(string(raw), "\n") + 1
}

// TestThePageOpensNoConnection keeps every outbound call in Go, where the host
// list below can see it: the Content-Security-Policy in index.html forbids the
// page any connection of its own (NFR-PRIV-001).
func TestThePageOpensNoConnection(t *testing.T) {
	path := filepath.Join(repoRoot(t), "frontend", "index.html")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	page := string(raw)
	for _, wanted := range []string{"Content-Security-Policy", "connect-src 'none'", "default-src 'self'"} {
		if !strings.Contains(page, wanted) {
			t.Errorf("%s does not carry %q", filepath.Base(path), wanted)
		}
	}
}

// TestEveryExportedTypeIsDocumented keeps the package surface self-describing.
func TestEveryExportedTypeIsDocumented(t *testing.T) {
	for _, path := range goFiles(t) {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		for _, declaration := range parsed.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE || general.Doc != nil {
				continue
			}
			for _, spec := range general.Specs {
				typed, ok := spec.(*ast.TypeSpec)
				if ok && typed.Name.IsExported() && typed.Doc == nil {
					t.Errorf("%s: exported type %s has no doc comment", path, typed.Name.Name)
				}
			}
		}
	}
}
