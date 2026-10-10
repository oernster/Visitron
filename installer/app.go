package main

import (
	"context"
	"os"
	"path/filepath"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/oernster/visitron/internal/infrastructure/setup"
	"github.com/oernster/visitron/internal/licence"
)

// App is the Wails facade for the setup program. Everything the page can do
// goes through a method here; the install policy itself lives in
// internal/infrastructure/setup, so this file owns none of it. The machine it
// acts on is held rather than reached for, which is what lets the sequences be
// tested against a recorder instead of a real computer.
type App struct {
	ctx           context.Context
	machine       setup.Machine
	report        setup.Report
	payload       []byte
	version       string
	uninstallMode bool
}

// NewApp builds the facade. Started with -uninstall, as the Apps list starts
// it, setup opens on the removal screen rather than on the manage one.
func NewApp(payload []byte, version string) *App {
	uninstall := len(os.Args) > 1 && os.Args[1] == setup.UninstallFlag
	app := newApp(setup.Real{}, payload, version, uninstall)
	app.report = app.progress
	return app
}

// newApp is the composition point NewApp and the tests share, so neither states
// the shape of an App twice.
//
// It reports to nobody until a caller says otherwise. A facade with no window
// behind it has nowhere to put a progress event; answering with something
// that does nothing beats answering with nothing at all: every step can report
// without first asking whether there is anyone listening.
func newApp(machine setup.Machine, payload []byte, version string, uninstall bool) *App {
	return &App{
		machine:       machine,
		report:        func(int, string) {},
		payload:       payload,
		version:       version,
		uninstallMode: uninstall,
	}
}

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

// StateDTO describes what setup should offer, given what is already installed.
//
// AppName is sent because the page must not write the product's name down. A
// page has no build step, so a name typed into it survives a rename with
// nothing to say so.
type StateDTO struct {
	AppName          string `json:"appName"`
	Mode             string `json:"mode"`
	Relation         string `json:"relation"`
	Installed        bool   `json:"installed"`
	InstalledVersion string `json:"installedVersion"`
	ThisVersion      string `json:"thisVersion"`
	InstallDir       string `json:"installDir"`
	RecordFile       string `json:"recordFile"`
	StartMenu        bool   `json:"startMenu"`
	Desktop          bool   `json:"desktop"`
}

// OptionsDTO carries the choices made on the install or reinstall screen.
type OptionsDTO struct {
	StartMenu bool `json:"startMenu"`
	Desktop   bool `json:"desktop"`
}

// LicenceDTO is the licence screen's whole content, so the page states none of
// it. The name of the licence, who holds the copyright, what it means in
// ordinary words and the published text itself all come from
// internal/licence, which the gate holds to the repository's own LICENSE.
type LicenceDTO struct {
	// Lead names the product and the licence in one sentence. It is built here
	// rather than on the page for the reason StateDTO.AppName exists: a page
	// with no build step that writes the product's name down survives a rename
	// with nothing to say so.
	Lead   string   `json:"lead"`
	Holder string   `json:"holder"`
	Plain  []string `json:"plain"`
	Text   string   `json:"text"`
	// Notices are the works this setup program itself is built from, shown at
	// the foot of the licence text because that is where long legal small
	// print belongs and because the window has no room for a box of its own.
	Notices []string `json:"notices"`
}

// setupCredits names what the setup program ships, which is deliberately not
// the application's list: they are two binaries with two sets of dependencies,
// and setup carries no database. The application's own credits live in
// internal/application/about.go and are shown in its About dialog.
var setupCredits = []string{
	"Go and golang.org/x/sys, BSD 3-Clause, © The Go Authors",
	"Wails, MIT, © Lea Anthony",
}

// Licence answers what Visitron is given under.
//
// The screen it fills exists because naming a licence is not explaining one:
// somebody about to install a program reads "GNU General Public Licence,
// version 3", learns nothing from it and presses the button anyway. The plain
// reading goes first for that reason, with the text beneath it for anyone who
// wants the thing itself rather than a summary of it.
func (a *App) Licence() LicenceDTO {
	return LicenceDTO{
		Lead:    setup.AppName + " is free software under the " + licence.Name + ".",
		Holder:  licence.Holder,
		Plain:   licence.Plainly(),
		Text:    licence.Text(),
		Notices: setupCredits,
	}
}

// Progress is emitted on the "progress" event while a long operation runs.
type Progress struct {
	Pct int    `json:"pct"`
	Msg string `json:"msg"`
}

// relationNames turn the comparison into the word the page routes on.
var relationNames = map[setup.Relation]string{
	setup.Newer: "newer",
	setup.Same:  "same",
	setup.Older: "older",
}

// DetectState reads the machine once and answers the route setup should open
// in. Reading it once is what keeps the screen, its heading, its options and
// its buttons from drifting apart.
func (a *App) DetectState() StateDTO {
	dir, _ := a.machine.InstallDir()
	record, _ := a.machine.RecordFile()
	installedVersion, installed := a.machine.InstalledVersion()
	shortcuts := a.machine.CurrentShortcuts()

	mode := "install"
	switch {
	case a.uninstallMode:
		mode = "uninstall"
	case installed:
		mode = "manage"
	}
	relation := setup.Same
	if installed {
		relation = setup.Compare(a.version, installedVersion)
	}
	return StateDTO{
		AppName:          setup.AppName,
		Mode:             mode,
		Relation:         relationNames[relation],
		Installed:        installed,
		InstalledVersion: installedVersion,
		ThisVersion:      a.version,
		InstallDir:       dir,
		RecordFile:       record,
		StartMenu:        shortcuts.StartMenu,
		Desktop:          shortcuts.Desktop,
	}
}

// AppRunning reports whether Visitron is open, so the page can offer to close it
// rather than failing later on a locked executable.
func (a *App) AppRunning() bool { return a.machine.AppRunning() }

// CloseRunningApp ends the running application so setup can proceed.
func (a *App) CloseRunningApp() error { return setup.CloseRunningApp() }

// Install performs a fresh install, an update, a way back or a reinstall. All
// four are the same act: write the files, then apply the options as given.
func (a *App) Install(choices OptionsDTO) error { return a.write(choices) }

// Repair writes the files again and leaves every option as it stands. It is the
// quick fix for a damaged install, as distinct from a reinstall, which puts the
// choices back to those of a new install.
func (a *App) Repair() error {
	shortcuts := a.machine.CurrentShortcuts()
	return a.write(OptionsDTO{StartMenu: shortcuts.StartMenu, Desktop: shortcuts.Desktop})
}

// write is the single install path behind Install and Repair.
func (a *App) write(choices OptionsDTO) error {
	return setup.Install(a.machine, a.report, a.payload, a.version, setup.Shortcuts{
		StartMenu: choices.StartMenu,
		Desktop:   choices.Desktop,
	})
}

// Uninstall removes the shortcuts, the registry entry, the log and the
// installed files, plus the window's own WebView2 folder, which holds no part
// of the record.
//
// The record itself is kept unless removeRecord says otherwise (FR-080).
// Removing the program is not a decision to throw away years of observations,
// so the screen names the file and asks.
func (a *App) Uninstall(removeRecord bool) error {
	return setup.Remove(a.machine, a.report, removeRecord)
}

// LaunchApp starts the installed application, backing the "start it when this
// finishes" option.
func (a *App) LaunchApp() error { return setup.LaunchApp() }

// SetShortcuts applies the shortcut boxes live from the manage screen, where
// there is nothing to install, so a box that waited for a go-ahead would never
// take effect at all.
func (a *App) SetShortcuts(startMenu, desktop bool) error {
	dir, err := a.machine.InstallDir()
	if err != nil {
		return err
	}
	a.machine.ApplyShortcuts(filepath.Join(dir, setup.ExeName), dir, setup.Shortcuts{
		StartMenu: startMenu,
		Desktop:   desktop,
	})
	return nil
}

// Quit closes the setup program.
func (a *App) Quit() { wailsruntime.Quit(a.ctx) }

// progress reports how far a long operation has got.
func (a *App) progress(pct int, msg string) {
	wailsruntime.EventsEmit(a.ctx, "progress", Progress{Pct: pct, Msg: msg})
}
