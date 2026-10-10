package structural

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// logWriters names every file allowed to write to the run log, with the number
// of places in it that do, then says what each one writes.
//
// NFR-SEC-001 promises that the GoatCounter key and the GitHub token never
// reach the log. Nothing about a write to standard error announces whether its
// arguments carry a secret, so the promise cannot be kept by inspecting the
// arguments; it is kept by there being few enough writers to read, all of
// them known. A new one fails here and has to be declared, which is the moment
// to ask what it puts in the file. Every write below is a fixed sentence plus
// an error, a recovered panic value or a stack; no secret is handed to any.
//
// The runlog package is not listed: it owns the log and writes the run's own
// lines.
var logWriters = map[string]int{
	// A scheduled check failed; a check panicked (value and
	// stack) and recording that failed; no focuser (a fixed sentence); the
	// data could not be closed; a bound method panicked (value and stack).
	"app.go": 6,
	// A refresh's check failed.
	"actions.go": 1,
	// The data could not be opened; no tray
	// icon (a fixed sentence); the window could not be run; the check's own
	// log, which writes its start, each website's outcome and its end
	// (NFR-OBS-001, Amendment 21) and never a key or a token.
	"main.go": 4,
	// Handing over the keyboard panicked; following the tray panicked (value
	// and stack each); a refresh asked for from the tray was refused.
	"window.go": 3,
	// The tray's message loop panicked: the recovered value and the stack.
	"internal/infrastructure/tray/tray_windows.go": 1,
}

// TestOnlyTheKnownPlacesWriteToTheLog keeps NFR-SEC-001's promise checkable.
func TestOnlyTheKnownPlacesWriteToTheLog(t *testing.T) {
	root := repoRoot(t)
	found := map[string]int{}
	for _, path := range goFiles(t) {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatalf("relative path of %s: %v", path, err)
		}
		relative = filepath.ToSlash(relative)
		if strings.HasPrefix(relative, "internal/infrastructure/runlog/") {
			continue
		}
		found[relative] += logWritesIn(t, path)
	}

	for name, count := range found {
		if count == 0 {
			continue
		}
		allowed, ok := logWriters[name]
		if !ok {
			t.Errorf("%s writes to the log in %d place(s) and is not declared in "+
				"logWriters: NFR-SEC-001 promises the log holds no secret, "+
				"which is only checkable while every writer is known", name, count)
			continue
		}
		if count != allowed {
			t.Errorf("%s writes to the log in %d place(s), declared %d: a new write "+
				"is a decision about what reaches the file, so it is taken twice",
				name, count, allowed)
		}
	}

	var absent []string
	for name := range logWriters {
		if found[name] == 0 {
			absent = append(absent, name)
		}
	}
	sort.Strings(absent)
	for _, name := range absent {
		t.Errorf("logWriters declares %s, which no longer writes to the log: a "+
			"declaration nothing matches is a rule a reader will look for and not find", name)
	}
}

// logWritesIn counts the calls in one file that write to standard error, which
// is where the run log is pointed before anything else can fail.
func logWritesIn(t *testing.T, path string) int {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	count := 0
	ast.Inspect(parsed, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		for _, argument := range call.Args {
			if isStandardError(argument) {
				count++
				return true
			}
		}
		return true
	})
	return count
}

// isStandardError answers whether an expression is os.Stderr.
func isStandardError(node ast.Expr) bool {
	selector, ok := node.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Stderr" {
		return false
	}
	packageName, ok := selector.X.(*ast.Ident)
	return ok && packageName.Name == "os"
}
