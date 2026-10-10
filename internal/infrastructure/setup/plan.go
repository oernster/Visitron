package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"visitron/internal/infrastructure/startup"
)

// UninstallExeName is the copy of setup left inside the install directory, so
// the Apps list has something to call after the downloaded setup file is gone.
const UninstallExeName = "uninstall.exe"

// The weighting is measured rather than counted: extracting the payload is most
// of the work, so the bar sits in it rather than reaching the end in a twentieth
// of a second and waiting there. Removal is the other way round, since no step
// of it moves much data.
const (
	pctWriting   = 10
	pctRegister  = 70
	pctChoices   = 90
	pctShortcuts = 20
	pctRegistry  = 45
	pctLeftovers = 60
	pctRecord    = 75
	pctFiles     = 90
	msgWriting   = "Writing the files..."
	msgChoices   = "Applying your choices..."
	msgShortcuts = "Removing shortcuts..."
	msgRegistry  = "Removing the registry entries..."
	msgLeftovers = "Clearing what the window kept..."
	msgRecord    = "Deleting your websites and their history..."
	msgFiles     = "Removing the files..."
)

// PctDone and MsgDone are the last thing every run reports. The page leaves the
// progress screen on reading them, so they are part of what crosses rather than
// a detail of the bar.
const (
	PctDone = 100
	MsgDone = "Done."
)

// Machine is everything an install or a removal does to the computer it runs on.
//
// It exists so the ORDER of those acts can be stated and tested. Each act on its
// own is already held by this package's own tests: the payload fence, the paths,
// the version comparison, the shortcut writing, the folders a removal clears.
// What needed a seam is the sequence, which is where the decisions live: that a
// removal takes the shortcuts before the registry entry, that it refuses outright
// while the application is open, that the record goes last and only when asked.
type Machine interface {
	AppRunning() bool
	InstallDir() (string, error)
	RecordDir() (string, error)
	RecordFile() (string, error)
	Executable() (string, error)
	InstalledVersion() (string, bool)
	CurrentShortcuts() Shortcuts
	ExtractZip(payload []byte, dest string) error
	CopyFile(src, dst string) error
	DirSizeKB(dir string) (uint32, error)
	RemoveTree(dir string) error
	Leftovers() []string
	WriteUninstallEntry(info UninstallInfo) error
	RemoveUninstallEntry() error
	ApplyShortcuts(exePath, workDir string, want Shortcuts)
	RemoveShortcuts()
	// StartsWithWindows reports whether the sign-in entry is present and
	// healthy; SetStartWithWindows writes or removes it (FR-052).
	StartsWithWindows() bool
	SetStartWithWindows(exePath string, on bool) error
	ScheduleDirDeletion(dir string)
}

// Report says how far a long operation has got. The setup program passes the
// page's progress event; a test passes a recorder and reads the sequence back.
type Report func(pct int, msg string)

// Install writes the files, registers the program with Windows and applies the
// shortcut and sign-in choices as given.
//
// A fresh install, an update, a way back and a reinstall are all this one act.
// What tells them apart is the screen the user came from and the choices they
// arrived with, never a different sequence here.
func Install(machine Machine, report Report, payload []byte, version string, want Shortcuts, startWithWindows bool) error {
	if machine.AppRunning() {
		return ErrAppRunning
	}
	dir, err := machine.InstallDir()
	if err != nil {
		return err
	}

	report(pctWriting, msgWriting)
	if err := machine.ExtractZip(payload, dir); err != nil {
		return fmt.Errorf("write the files: %w", err)
	}
	exePath := filepath.Join(dir, ExeName)

	report(pctRegister, "Registering "+AppName+" with Windows...")
	if err := register(machine, version, dir, exePath); err != nil {
		return err
	}

	report(pctChoices, msgChoices)
	machine.ApplyShortcuts(exePath, dir, want)
	// Unlike a shortcut, a sign-in entry that silently failed would leave the
	// reader believing Visitron watches from sign-in when it does not, so its
	// failure is said rather than swallowed.
	if err := machine.SetStartWithWindows(exePath, startWithWindows); err != nil {
		return fmt.Errorf("%s is installed; starting it with Windows could not be set: %w", AppName, err)
	}

	report(PctDone, MsgDone)
	return nil
}

// register leaves a copy of setup beside the application and writes the Apps
// list entry that points at it, so Windows reopens setup for a Modify or a
// Repair rather than sending the reader back to the download.
func register(machine Machine, version, dir, exePath string) error {
	self, err := machine.Executable()
	if err != nil {
		return fmt.Errorf("locate setup: %w", err)
	}
	uninstallExe := filepath.Join(dir, UninstallExeName)
	if err := machine.CopyFile(self, uninstallExe); err != nil {
		return fmt.Errorf("write the uninstaller: %w", err)
	}
	sizeKB, _ := machine.DirSizeKB(dir)
	if err := machine.WriteUninstallEntry(UninstallInfo{
		Version:      version,
		InstallDir:   dir,
		UninstallExe: uninstallExe,
		IconPath:     exePath,
		EstimatedKB:  sizeKB,
	}); err != nil {
		return fmt.Errorf("register %s: %w", AppName, err)
	}
	return nil
}

// Remove takes the program off the machine: the shortcuts, the registry entry,
// the log and the installed files, plus the window's own WebView2 folder, which
// holds no part of the record.
//
// The record itself is kept unless removeRecord says otherwise (FR-080).
// Removing the program is not a decision to throw away years of observations, so
// the screen names the file and asks. That is also why the record goes last:
// every step before it is reversible by installing again.
func Remove(machine Machine, report Report, removeRecord bool) error {
	if machine.AppRunning() {
		return ErrAppRunning
	}
	dir, err := machine.InstallDir()
	if err != nil {
		return err
	}

	report(pctShortcuts, msgShortcuts)
	machine.RemoveShortcuts()

	report(pctRegistry, msgRegistry)
	_ = machine.SetStartWithWindows("", false)
	_ = machine.RemoveUninstallEntry()

	report(pctLeftovers, msgLeftovers)
	for _, folder := range machine.Leftovers() {
		_ = machine.RemoveTree(folder)
	}

	if removeRecord {
		report(pctRecord, msgRecord)
		if record, recordErr := machine.RecordDir(); recordErr == nil {
			_ = machine.RemoveTree(record)
		}
	}

	// The install directory holds the running setup program, which cannot delete
	// itself, so it is handed to Windows to take once this process is gone.
	report(pctFiles, msgFiles)
	machine.ScheduleDirDeletion(dir)

	report(PctDone, MsgDone)
	return nil
}

// Real is the machine the setup program actually runs against. Every method is
// one call, so there is nothing here for a test to hold and nothing a fake has
// to reproduce beyond the signature.
type Real struct{}

func (Real) AppRunning() bool                     { return IsAppRunning() }
func (Real) InstallDir() (string, error)          { return InstallDir() }
func (Real) RecordDir() (string, error)           { return RecordDir() }
func (Real) RecordFile() (string, error)          { return RecordFile() }
func (Real) Executable() (string, error)          { return os.Executable() }
func (Real) InstalledVersion() (string, bool)     { return InstalledVersion() }
func (Real) CurrentShortcuts() Shortcuts          { return CurrentShortcuts() }
func (Real) CopyFile(src, dst string) error       { return CopyFile(src, dst) }
func (Real) DirSizeKB(dir string) (uint32, error) { return DirSizeKB(dir) }
func (Real) RemoveTree(dir string) error          { return RemoveTree(dir) }
func (Real) Leftovers() []string                  { return Leftovers() }
func (Real) RemoveUninstallEntry() error          { return RemoveUninstallEntry() }
func (Real) RemoveShortcuts()                     { RemoveShortcuts() }
func (Real) ScheduleDirDeletion(dir string)       { ScheduleDirDeletion(dir) }

func (Real) ExtractZip(payload []byte, dest string) error { return ExtractZip(payload, dest) }

func (Real) WriteUninstallEntry(info UninstallInfo) error { return WriteUninstallEntry(info) }

func (Real) ApplyShortcuts(exePath, workDir string, want Shortcuts) {
	ApplyShortcuts(exePath, workDir, want)
}

// StartsWithWindows and SetStartWithWindows go through the one entry the
// application's own Settings switch uses, under the same name, so setup and
// Settings can never disagree about it (FR-052).
func (Real) StartsWithWindows() bool {
	on, _ := startup.Entry{Name: AppName}.Enabled()
	return on
}

func (Real) SetStartWithWindows(exePath string, on bool) error {
	return startup.Entry{Name: AppName, Exe: exePath}.SetEnabled(on)
}
