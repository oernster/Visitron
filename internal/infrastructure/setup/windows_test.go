//go:build windows

package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// shortcutHome points the Desktop and Start Menu folders at temporary
// directories, so a test places real shortcuts without touching the machine's
// own.
func shortcutHome(t *testing.T) (desktop, startMenu string) {
	t.Helper()
	home := t.TempDir()
	roaming := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", roaming)
	desktop = filepath.Join(home, "Desktop")
	if err := os.MkdirAll(desktop, 0o755); err != nil {
		t.Fatal(err)
	}
	return desktop, filepath.Join(roaming, "Microsoft", "Windows", "Start Menu", "Programs")
}

// readShortcut answers a .lnk's target and icon through the same script host
// that wrote it.
func readShortcut(t *testing.T, path string) (target, icon string) {
	t.Helper()
	script := `$s=(New-Object -ComObject WScript.Shell).CreateShortcut(` + literal(path) +
		`); Write-Output $s.TargetPath; Write-Output $s.IconLocation`
	out, err := runPowerShell(script)
	if err != nil {
		t.Fatalf("reading %s: %v: %s", path, err, out)
	}
	lines := strings.Split(strings.ReplaceAll(strings.TrimSpace(out), "\r\n", "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("reading %s answered %q", path, out)
	}
	return strings.TrimSpace(lines[0]), strings.TrimSpace(lines[1])
}

// TestShortcutsCarryPlainPaths proves the shortcut is written with the path as
// it stands.
//
// The script was built with %q, which is Go's quoting: PowerShell keeps a
// backslash as a backslash, so every separator reached the shortcut doubled.
// Measured on a real install before the fix: the icon read
// C:\\Users\\Oliver\\...
func TestShortcutsCarryPlainPaths(t *testing.T) {
	desktop, startMenu := shortcutHome(t)
	install := t.TempDir()
	exePath := filepath.Join(install, ExeName)
	if err := os.WriteFile(exePath, []byte("the application"), 0o600); err != nil {
		t.Fatal(err)
	}

	// The Start Menu folder does not exist yet, which is the case that placed
	// no entry at all before the folder was created first.
	ApplyShortcuts(exePath, install, Shortcuts{StartMenu: true, Desktop: true})

	for _, dir := range []string{desktop, startMenu} {
		link := filepath.Join(dir, shortcutName)
		if _, err := os.Stat(link); err != nil {
			t.Fatalf("no shortcut at %s: %v", link, err)
		}
		target, icon := readShortcut(t, link)
		if target != exePath {
			t.Errorf("%s points at %q, want %q", link, target, exePath)
		}
		if strings.Contains(icon, `\\`) {
			t.Errorf("%s has doubled separators in its icon: %q", link, icon)
		}
	}

	if held := CurrentShortcuts(); !held.StartMenu || !held.Desktop {
		t.Errorf("CurrentShortcuts = %+v, want both", held)
	}

	// Unticking a box takes the shortcut away rather than leaving a stale one.
	ApplyShortcuts(exePath, install, Shortcuts{StartMenu: true})
	if held := CurrentShortcuts(); !held.StartMenu || held.Desktop {
		t.Errorf("after unticking the desktop box: %+v", held)
	}

	RemoveShortcuts()
	if held := CurrentShortcuts(); held.StartMenu || held.Desktop {
		t.Errorf("after removing both: %+v", held)
	}
}

func TestLiteralQuotesForPowerShell(t *testing.T) {
	t.Parallel()
	if got := literal(`C:\Users\Oliver\Visitron.exe`); got != `'C:\Users\Oliver\Visitron.exe'` {
		t.Errorf("literal = %s", got)
	}
	// A quote inside a path is doubled, which is how PowerShell reads one
	// literally, so a folder named O'Brien cannot end the string early.
	if got := literal(`C:\Users\O'Brien\a.exe`); got != `'C:\Users\O''Brien\a.exe'` {
		t.Errorf("literal = %s", got)
	}
}
