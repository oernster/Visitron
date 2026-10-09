package structural

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// wireContract is the front end's hand-written statement of the shapes that
// cross the window boundary.
//
// Wails generates the same shapes into frontend/wailsjs at build time. That
// output is gitignored and imported by nothing, so it is not the contract. This
// one is, typed out by a person. Nothing in either build compares the two
// halves: the type checker sees the interface, the marshaller sees the struct
// and neither can see the other.
var wireContract = filepath.Join("frontend", "src", "api.ts")

// dtoSuffix marks a struct as part of the wire.
const dtoSuffix = "DTO"

// wireShapes pairs each DTO with the interface that restates it. The list is
// here so that adding a shape is a decision taken twice, once in the code and
// once against this.
var wireShapes = map[string]string{
	"StateDTO":      "State",
	"WebsiteRowDTO": "WebsiteRow",
	"OverviewDTO":   "Overview",
	"NamedCountDTO": "NamedCount",
	"DayCountDTO":   "DayCount",
	"DetailDTO":     "Detail",
	"ProposalDTO":   "Proposal",
	"ProgressDTO":   "Progress",
	"SettingsDTO":   "Settings",
	"CreditDTO":     "Credit",
	"AboutDTO":      "About",
}

var (
	tsInterface = regexp.MustCompile(`(?s)export interface (\w+) \{(.*?)\n\}`)
	tsField     = regexp.MustCompile(`(?m)^\s+([A-Za-z_]\w*)\??:`)
	tsComment   = regexp.MustCompile(`(?s)/\*.*?\*/`)
)

// facadeFieldNames reads the wire field names off one struct, taking the json
// tag rather than the Go name because the tag is what reaches the page.
func facadeFieldNames(t *testing.T, path, name string, structure *ast.StructType) []string {
	t.Helper()
	var fields []string
	for _, field := range structure.Fields.List {
		if len(field.Names) == 0 || !field.Names[0].IsExported() {
			continue
		}
		if field.Tag == nil {
			t.Errorf("%s: %s.%s has no json tag, so it ships under its Go name",
				filepath.ToSlash(path), name, field.Names[0].Name)
			continue
		}
		raw, err := strconv.Unquote(field.Tag.Value)
		if err != nil {
			t.Fatalf("%s: unreadable tag on %s.%s", path, name, field.Names[0].Name)
		}
		tag := strings.Split(reflect.StructTag(raw).Get("json"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		fields = append(fields, tag)
	}
	sort.Strings(fields)
	return fields
}

// facadeDTOs returns every wire struct the application's facade declares, by
// json field name. Only the files at the repository root are read: that is
// where the facade is.
func facadeDTOs(t *testing.T, root string) map[string][]string {
	t.Helper()
	return dtosIn(t, root, root)
}

// dtosIn reads the wire structs one directory declares. It takes the directory
// rather than assuming the root because there are two facades: the
// application's, at the root, plus the setup program's, under installer.
func dtosIn(t *testing.T, root, dir string) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	for _, path := range goFiles(t) {
		if filepath.Dir(path) != dir || strings.HasSuffix(path, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		relative, _ := filepath.Rel(root, path)
		for _, declaration := range parsed.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}
			for _, spec := range general.Specs {
				typed, ok := spec.(*ast.TypeSpec)
				if !ok || !strings.HasSuffix(typed.Name.Name, dtoSuffix) {
					continue
				}
				if structure, ok := typed.Type.(*ast.StructType); ok {
					out[typed.Name.Name] = facadeFieldNames(t, relative, typed.Name.Name, structure)
				}
			}
		}
	}
	if len(out) == 0 {
		t.Fatalf("no DTOs found in %s, the walk is wrong", filepath.ToSlash(dir))
	}
	return out
}

// wireInterfaces returns every interface the front end declares, by field name.
func wireInterfaces(t *testing.T, root string) map[string][]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, wireContract))
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.ToSlash(wireContract), err)
	}
	source := tsComment.ReplaceAll(raw, nil)
	out := map[string][]string{}
	for _, match := range tsInterface.FindAllSubmatch(source, -1) {
		var fields []string
		for _, field := range tsField.FindAllSubmatch(match[2], -1) {
			fields = append(fields, string(field[1]))
		}
		sort.Strings(fields)
		out[string(match[1])] = fields
	}
	if len(out) == 0 {
		t.Fatalf("no interfaces found in %s, the pattern is wrong", filepath.ToSlash(wireContract))
	}
	return out
}

// absent returns the entries of want that got does not hold.
func absent(want, got []string) []string {
	present := map[string]bool{}
	for _, item := range got {
		present[item] = true
	}
	var out []string
	for _, item := range want {
		if !present[item] {
			out = append(out, item)
		}
	}
	return out
}

// TestTheWireContractMatchesOnBothSides keeps the two statements in step.
//
// A field renamed or removed on the Go side alone leaves the interface
// declaring something that never arrives, which the type checker cannot see and
// the page reads as undefined; the reverse leaves a value crossing on every
// call that nothing collects.
func TestTheWireContractMatchesOnBothSides(t *testing.T) {
	root := repoRoot(t)
	structs := facadeDTOs(t, root)
	interfaces := wireInterfaces(t, root)

	var undeclared []string
	for name := range structs {
		if _, ok := wireShapes[name]; !ok {
			undeclared = append(undeclared, name)
		}
	}
	sort.Strings(undeclared)
	for _, name := range undeclared {
		t.Errorf("%s crosses the wire but is not paired in wireShapes: a shape the page "+
			"receives is a decision, so it is declared there as well", name)
	}

	names := make([]string, 0, len(wireShapes))
	for name := range wireShapes {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		checkShape(t, name, wireShapes[name], structs, interfaces)
	}
}

// checkShape compares one pair, kept apart from the loop so each failure reads
// as a statement about that shape.
func checkShape(t *testing.T, name, shape string, structs, interfaces map[string][]string) {
	t.Helper()
	fields, ok := structs[name]
	if !ok {
		t.Errorf("wireShapes pairs %s, which no longer exists in the facade", name)
		return
	}
	declared, ok := interfaces[shape]
	if !ok {
		t.Errorf("%s has no interface %s in %s", name, shape, filepath.ToSlash(wireContract))
		return
	}
	for _, field := range absent(fields, declared) {
		t.Errorf("%s.%s is sent but interface %s does not declare it: the page cannot read "+
			"a field it has not been told about", name, field, shape)
	}
	for _, field := range absent(declared, fields) {
		t.Errorf("interface %s declares %s but %s does not send it: it would read as "+
			"undefined at runtime with nothing to say so", shape, field, name)
	}
}
