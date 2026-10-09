//go:build windows

package startup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows/registry"

	"github.com/oernster/visitron/internal/application"
)

var _ application.Startup = Entry{}

// testEntry writes under a value name of its own (never Visitron's) then
// removes it afterwards.
func testEntry(t *testing.T) Entry {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "Visitron Test.exe")
	if err := os.WriteFile(exe, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	e := Entry{Name: fmt.Sprintf("VisitronTest%d", os.Getpid()), Exe: exe}
	t.Cleanup(func() { _ = e.SetEnabled(false) })
	return e
}

func TestHKCURunEntry(t *testing.T) {
	e := testEntry(t)
	if on, err := e.Enabled(); on || err != nil {
		t.Errorf("before: %v %v", on, err)
	}
	if err := e.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	key, _ := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	value, _, _ := key.GetStringValue(e.Name)
	_ = key.Close()
	if value != `"`+e.Exe+`" -hidden` {
		t.Errorf("stored %q", value)
	}
	if on, err := e.Enabled(); !on || err != nil {
		t.Errorf("after: %v %v", on, err)
	}
	if err := e.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if err := e.SetEnabled(false); err != nil {
		t.Errorf("removing twice: %v", err)
	}
	if on, _ := e.Enabled(); on {
		t.Error("still on after removal")
	}
}

func TestUnhealthyEntriesReadAsOff(t *testing.T) {
	e := testEntry(t)
	for _, value := range []string{`"` + e.Exe + `"`, `"C:\gone\Visitron.exe" -hidden`} {
		key, _, _ := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
		_ = key.SetStringValue(e.Name, value)
		_ = key.Close()
		if on, _ := e.Enabled(); on {
			t.Errorf("%q read as on", value)
		}
	}
}

func TestTargetReading(t *testing.T) {
	cases := map[string]string{
		`"C:\a b\V.exe" -hidden`: `C:\a b\V.exe`,
		`C:\V.exe -hidden`:       `C:\V.exe`,
		`"C:\unclosed -hidden`:   `"C:\unclosed`,
		``:                       ``,
	}
	for value, want := range cases {
		if got := target(value); got != want {
			t.Errorf("target(%q) = %q; want %q", value, got, want)
		}
	}
}
