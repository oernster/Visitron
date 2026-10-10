//go:build windows

package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	uninstallKeyPath = `Software\Microsoft\Windows\CurrentVersion\Uninstall\` + InstallFolder
	shortcutName     = AppName + ".lnk"
)

// UninstallInfo carries the values written to the HKCU uninstall registry
// entry, which is what puts Visitron in Settings and in the Apps list.
type UninstallInfo struct {
	Version      string
	InstallDir   string
	UninstallExe string
	IconPath     string
	EstimatedKB  uint32
}

// hidden keeps a shelled-out child from flashing a console window over the
// progress the user is watching.
func hidden() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
}

// WriteUninstallEntry registers Visitron under the current user's uninstall
// list. NoModify and NoRepair are both zero, so Windows offers Modify and
// Repair alongside Uninstall and each one reopens this setup program.
func WriteUninstallEntry(info UninstallInfo) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, uninstallKeyPath, registry.WRITE)
	if err != nil {
		return fmt.Errorf("create uninstall key: %w", err)
	}
	defer key.Close()

	quoted := quotedPath(info.UninstallExe)
	text := map[string]string{
		"DisplayName":     AppName,
		"DisplayVersion":  info.Version,
		"InstallLocation": info.InstallDir,
		"UninstallString": quoted + " " + UninstallFlag,
		"ModifyPath":      quoted,
		"DisplayIcon":     info.IconPath,
		"Publisher":       Publisher,
	}
	for name, value := range text {
		if err := key.SetStringValue(name, value); err != nil {
			return fmt.Errorf("set %q: %w", name, err)
		}
	}
	numbers := map[string]uint32{"NoModify": 0, "NoRepair": 0, "EstimatedSize": info.EstimatedKB}
	for name, value := range numbers {
		if err := key.SetDWordValue(name, value); err != nil {
			return fmt.Errorf("set %q: %w", name, err)
		}
	}
	return nil
}

// RemoveUninstallEntry deletes the uninstall registry entry.
func RemoveUninstallEntry() error {
	if err := registry.DeleteKey(registry.CURRENT_USER, uninstallKeyPath); err != nil {
		return fmt.Errorf("delete uninstall key: %w", err)
	}
	return nil
}

// InstalledVersion answers the installed version and whether Visitron is
// installed at all, read from the uninstall registry entry.
func InstalledVersion() (string, bool) {
	key, err := registry.OpenKey(registry.CURRENT_USER, uninstallKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return "", false
	}
	defer key.Close()
	value, _, err := key.GetStringValue("DisplayVersion")
	if err != nil {
		return "", false
	}
	return value, true
}

// literal writes a path as a PowerShell single-quoted string, which takes every
// character as it stands.
//
// %q was used here and it is Go's quoting, not PowerShell's: it escapes each
// separator, while a PowerShell double-quoted string keeps a backslash as a
// backslash, so the doubled form reached the shortcut. Measured on the first
// real install: the Start Menu entry's icon read C:\\Users\\Oliver\\...
func literal(path string) string {
	return "'" + strings.ReplaceAll(path, "'", "''") + "'"
}

// createShortcut writes a .lnk through the Windows Script Host, which avoids
// handling COM directly for one call.
func createShortcut(linkPath, target, workDir string) error {
	script := fmt.Sprintf(
		`$s=(New-Object -ComObject WScript.Shell).CreateShortcut(%s);`+
			`$s.TargetPath=%s;$s.IconLocation=%s;$s.WorkingDirectory=%s;$s.Save()`,
		literal(linkPath), literal(target), literal(target), literal(workDir))
	if out, err := runPowerShell(script); err != nil {
		return fmt.Errorf("create shortcut %q: %w: %s", linkPath, err, out)
	}
	return nil
}

// runPowerShell runs one script with no console window of its own. A console
// child of a windowless setup program is given a brand new console by Windows,
// which flashes a black box over the progress the user is watching.
func runPowerShell(script string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = hidden()
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Shortcuts says which shortcuts the user asked for.
type Shortcuts struct {
	StartMenu bool
	Desktop   bool
}

// ApplyShortcuts creates the shortcuts that are wanted and removes the ones
// that are not, so unticking a box on a reinstall takes the shortcut away
// rather than leaving a stale one. Each location is best effort: a missing
// shortcut is not worth failing an install over.
func ApplyShortcuts(exePath, workDir string, want Shortcuts) {
	place := func(dir string, wanted bool) {
		link := filepath.Join(dir, shortcutName)
		if !wanted {
			_ = os.Remove(link)
			return
		}
		// The folder is made first: the Start Menu Programs folder is not
		// guaranteed to exist; the script host writes no shortcut into a
		// folder that is not there. Measured: the first install placed the
		// desktop shortcut and silently placed no Start Menu entry.
		if err := os.MkdirAll(dir, dirPerm); err != nil {
			return
		}
		_ = createShortcut(link, exePath, workDir)
	}
	if dir, err := StartMenuProgramsDir(); err == nil {
		place(dir, want.StartMenu)
	}
	if dir, err := DesktopDir(); err == nil {
		place(dir, want.Desktop)
	}
}

// CurrentShortcuts reports which shortcuts exist, so every screen opens with
// its boxes reflecting the machine rather than all ticked.
func CurrentShortcuts() Shortcuts {
	present := func(dir string, err error) bool {
		if err != nil {
			return false
		}
		_, statErr := os.Stat(filepath.Join(dir, shortcutName))
		return statErr == nil
	}
	start, startErr := StartMenuProgramsDir()
	desktop, desktopErr := DesktopDir()
	return Shortcuts{
		StartMenu: present(start, startErr),
		Desktop:   present(desktop, desktopErr),
	}
}

// RemoveShortcuts deletes both shortcuts.
func RemoveShortcuts() {
	ApplyShortcuts("", "", Shortcuts{})
}

// DesktopDir answers the current user's Desktop directory.
func DesktopDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, "Desktop"), nil
}

// StartMenuProgramsDir answers the current user's Start Menu Programs
// directory.
func StartMenuProgramsDir() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", fmt.Errorf("APPDATA is not set")
	}
	return filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs"), nil
}
