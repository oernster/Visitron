package structural

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A requirement's "Verified by" is a promise that a named test proves it. A
// file renamed or a test moved leaves the promise pointing at nothing, which
// reads exactly like coverage while being none: this was measured on
// 2026-10-09, when twelve of the specification's references had drifted.

// specification is the document the references live in.
const specification = "REQUIREMENTS.md"

// leastReferences is a vacuity floor: a change to the document's wording that
// stopped the scan matching would otherwise pass over an empty list.
const leastReferences = 30

var (
	// verifiedBy opens a reference block; it runs on until a blank line or the
	// next list item.
	verifiedBy = regexp.MustCompile(`Verified\s+by`)
	// reference is one backticked name in a block.
	reference = regexp.MustCompile("`([^`]+)`")
	// testSource is a file a reference may name.
	testSource = regexp.MustCompile(`\.(go|tsx?)$`)
	// bareTest is a Go test named without its file.
	bareTest = regexp.MustCompile(`^Test\w+$`)
)

// verificationReferences answers every backticked name in a "Verified by"
// block of the specification.
func verificationReferences(t *testing.T) []string {
	t.Helper()
	var out []string
	inside := false
	for _, line := range strings.Split(readFile(t, filepath.Join(repoRoot(t), specification)), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case verifiedBy.MatchString(line):
			inside = true
		case trimmed == "" || strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "**"):
			inside = false
		}
		if !inside {
			continue
		}
		for _, match := range reference.FindAllStringSubmatch(line, -1) {
			out = append(out, match[1])
		}
	}
	return out
}

// testFiles indexes every test file by its base name and by its path.
func testFiles(t *testing.T) (byName map[string][]string, byPath map[string]string) {
	t.Helper()
	root := repoRoot(t)
	byName, byPath = map[string][]string{}, map[string]string{}
	for _, path := range append(goFiles(t), frontendFiles(t)...) {
		base := filepath.Base(path)
		if !strings.HasSuffix(base, "_test.go") && !strings.Contains(base, ".test.") {
			continue
		}
		byName[base] = append(byName[base], path)
		byPath[relativeTo(root, path)] = path
	}
	return byName, byPath
}

// TestEveryVerificationNamesARealTest fails on a reference to a file that is
// not there; likewise on a test the file does not hold.
func TestEveryVerificationNamesARealTest(t *testing.T) {
	byName, byPath := testFiles(t)
	allGo := ""
	for name, paths := range byName {
		if strings.HasSuffix(name, ".go") {
			for _, path := range paths {
				allGo += readFile(t, path)
			}
		}
	}
	checked := 0
	for _, ref := range verificationReferences(t) {
		file, test, _ := strings.Cut(ref, "::")
		switch {
		case bareTest.MatchString(ref):
			checked++
			if !strings.Contains(allGo, "func "+ref+"(") {
				t.Errorf("%s cites %s, which no Go test declares", specification, ref)
			}
		case testSource.MatchString(file):
			checked++
			paths := byName[filepath.Base(file)]
			if strings.Contains(file, "/") {
				paths = nil
				if path, ok := byPath[file]; ok {
					paths = []string{path}
				}
			}
			if len(paths) == 0 {
				t.Errorf("%s cites %s, which is not in the tree", specification, file)
				continue
			}
			if test != "" && !holds(t, paths, test) {
				t.Errorf("%s cites %s, which %s does not hold", specification, test, file)
			}
		}
	}
	if checked < leastReferences {
		t.Fatalf("only %d references were checked, fewer than the %d that exist: the scan is wrong",
			checked, leastReferences)
	}
}

// holds reports whether one of the files declares the named test: a Go test
// function or a front-end test whose title is the name.
func holds(t *testing.T, paths []string, test string) bool {
	t.Helper()
	for _, path := range paths {
		source := readFile(t, path)
		if strings.Contains(source, "func "+test+"(") ||
			strings.Contains(source, "it('"+test+"'") || strings.Contains(source, "describe('"+test+"'") {
			return true
		}
	}
	return false
}
