// Package setuptest holds the double the install and removal sequences are
// tested against.
//
// It is a package of its own rather than a type inside either test file because
// two suites need the same double: this package's own tests, which state what
// the sequences do, plus the setup program's, which state that its facade hands
// the work over unchanged. One double means one statement of what a machine
// answers, so the two suites cannot drift apart in what they believe.
package setuptest

import (
	"path/filepath"

	"github.com/oernster/visitron/internal/infrastructure/setup"
)

// recordName is the file the application keeps its observations in. The double
// answers a path under its own pretend folders, so nothing here touches a real
// one.
const recordName = "visitron.db"

// Machine records what was done to it, in the order it was done, so a test can
// state the sequence rather than the individual acts. Every field an act reads
// is settable, so a test says what the computer answers and nothing reaches the
// computer the test is running on.
type Machine struct {
	Calls     []string
	Steps     []Step
	Removed   []string
	Applied   setup.Shortcuts
	Running   bool
	Dir       string
	DirErr    error
	Record    string
	RecordErr error
	Self      string
	SelfErr   error
	Extract   error
	Copy      error
	Entry     error
	RemoveErr error
	Leftover  []string
	Installed string
	Holds     bool
	Shortcuts setup.Shortcuts
	Sized     uint32
}

// Step is one progress report, kept so the bar's own sequence is testable.
type Step struct {
	Pct int
	Msg string
}

// Ready answers a machine on which everything succeeds.
func Ready() *Machine {
	return &Machine{
		Dir:      filepath.Join("C:\\", "Programs", setup.AppName),
		Record:   filepath.Join("C:\\", "Record", setup.AppName),
		Self:     filepath.Join("C:\\", "Downloads", "VisitronSetup.exe"),
		Leftover: []string{"webview", "logs"},
	}
}

// Report is the progress reporter to hand the sequence under test.
func (m *Machine) Report(pct int, msg string) {
	m.Steps = append(m.Steps, Step{Pct: pct, Msg: msg})
}

func (m *Machine) did(name string) { m.Calls = append(m.Calls, name) }

func (m *Machine) AppRunning() bool { m.did("AppRunning"); return m.Running }

func (m *Machine) InstallDir() (string, error) {
	m.did("InstallDir")
	return m.Dir, m.DirErr
}

func (m *Machine) RecordDir() (string, error) {
	m.did("RecordDir")
	return m.Record, m.RecordErr
}

func (m *Machine) RecordFile() (string, error) {
	m.did("RecordFile")
	return filepath.Join(m.Record, recordName), m.RecordErr
}

func (m *Machine) Executable() (string, error) {
	m.did("Executable")
	return m.Self, m.SelfErr
}

func (m *Machine) InstalledVersion() (string, bool) {
	m.did("InstalledVersion")
	return m.Installed, m.Holds
}

func (m *Machine) CurrentShortcuts() setup.Shortcuts {
	m.did("CurrentShortcuts")
	return m.Shortcuts
}

func (m *Machine) ExtractZip([]byte, string) error { m.did("ExtractZip"); return m.Extract }

func (m *Machine) CopyFile(string, string) error { m.did("CopyFile"); return m.Copy }

func (m *Machine) DirSizeKB(string) (uint32, error) { m.did("DirSizeKB"); return m.Sized, nil }

func (m *Machine) RemoveTree(dir string) error {
	m.did("RemoveTree")
	m.Removed = append(m.Removed, dir)
	return m.RemoveErr
}

func (m *Machine) Leftovers() []string { m.did("Leftovers"); return m.Leftover }

func (m *Machine) WriteUninstallEntry(setup.UninstallInfo) error {
	m.did("WriteUninstallEntry")
	return m.Entry
}

func (m *Machine) RemoveUninstallEntry() error {
	m.did("RemoveUninstallEntry")
	return m.RemoveErr
}

func (m *Machine) ApplyShortcuts(_, _ string, want setup.Shortcuts) {
	m.did("ApplyShortcuts")
	m.Applied = want
}

func (m *Machine) RemoveShortcuts() { m.did("RemoveShortcuts") }

func (m *Machine) ScheduleDirDeletion(string) { m.did("ScheduleDirDeletion") }
