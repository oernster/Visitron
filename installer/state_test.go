package main

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/visitron/internal/infrastructure/setup"
	"github.com/oernster/visitron/internal/infrastructure/setup/setuptest"
	"github.com/oernster/visitron/internal/licence"
)

// thisVersion is the version the setup program under test carries. The machine
// answers what is already installed, so the pair decides the route.
const thisVersion = "1.2.0"

// fresh answers a facade over a machine with nothing installed.
func fresh(t *testing.T) (*App, *setuptest.Machine) {
	t.Helper()
	machine := setuptest.Ready()
	return newApp(machine, []byte("payload"), thisVersion, false), machine
}

func TestSetupOpensOnInstallWhereNothingIsThere(t *testing.T) {
	t.Parallel()
	app, machine := fresh(t)

	state := app.DetectState()

	if state.Mode != "install" {
		t.Errorf("mode = %q, want install", state.Mode)
	}
	if state.Installed {
		t.Error("state says something is installed when nothing is")
	}
	if state.ThisVersion != thisVersion {
		t.Errorf("thisVersion = %q, want %q", state.ThisVersion, thisVersion)
	}
	if state.InstallDir != machine.Dir {
		t.Errorf("installDir = %q, want %q", state.InstallDir, machine.Dir)
	}
	// The page must not write the product's name down, so it is sent one.
	if state.AppName != setup.AppName {
		t.Errorf("appName = %q, want %q", state.AppName, setup.AppName)
	}
}

func TestSetupOpensOnManageWhereItIsAlreadyInstalled(t *testing.T) {
	t.Parallel()
	app, machine := fresh(t)
	machine.Holds = true
	machine.Installed = "1.0.0"
	machine.Shortcuts = setup.Shortcuts{StartMenu: true, Desktop: true}

	state := app.DetectState()

	if state.Mode != "manage" {
		t.Errorf("mode = %q, want manage", state.Mode)
	}
	if state.InstalledVersion != "1.0.0" {
		t.Errorf("installedVersion = %q, want 1.0.0", state.InstalledVersion)
	}
	if !state.StartMenu || !state.Desktop {
		t.Errorf("the boxes opened at %+v, want what the machine already holds", state)
	}
}

func TestSetupStartedForRemovalOpensOnRemoval(t *testing.T) {
	t.Parallel()
	machine := setuptest.Ready()
	machine.Holds = true
	machine.Installed = thisVersion
	app := newApp(machine, nil, thisVersion, true)

	// The Apps list starts setup with the removal flag. That beats the manage
	// screen it would otherwise open on, since removal is what was asked for.
	if mode := app.DetectState().Mode; mode != "uninstall" {
		t.Errorf("mode = %q, want uninstall", mode)
	}
}

func TestTheRouteNamesHowTheVersionsCompare(t *testing.T) {
	t.Parallel()
	// The screen, its heading, its options and its buttons all read this one
	// word, which is what keeps them from drifting apart.
	for _, testCase := range []struct {
		installed string
		want      string
	}{
		{"1.0.0", "newer"},
		{thisVersion, "same"},
		{"2.0.0", "older"},
	} {
		t.Run(testCase.want, func(t *testing.T) {
			t.Parallel()
			app, machine := fresh(t)
			machine.Holds = true
			machine.Installed = testCase.installed

			if relation := app.DetectState().Relation; relation != testCase.want {
				t.Errorf("this %s against installed %s = %q, want %q",
					thisVersion, testCase.installed, relation, testCase.want)
			}
		})
	}
}

func TestNothingInstalledIsNotComparedWithAnything(t *testing.T) {
	t.Parallel()
	app, _ := fresh(t)

	// There is no earlier version to stand in a relation to, so the word has to
	// be the harmless one rather than a comparison against an empty string.
	if relation := app.DetectState().Relation; relation != "same" {
		t.Errorf("relation with nothing installed = %q, want same", relation)
	}
}

func TestInstallHandsTheChoicesOverUnchanged(t *testing.T) {
	t.Parallel()
	app, machine := fresh(t)

	if err := app.Install(OptionsDTO{StartMenu: true, Desktop: false}); err != nil {
		t.Fatalf("Install: %v", err)
	}

	want := setup.Shortcuts{StartMenu: true, Desktop: false}
	if machine.Applied != want {
		t.Errorf("applied = %+v, want %+v", machine.Applied, want)
	}
}

func TestRepairLeavesEveryChoiceAsItStands(t *testing.T) {
	t.Parallel()
	app, machine := fresh(t)
	machine.Shortcuts = setup.Shortcuts{StartMenu: false, Desktop: true}

	if err := app.Repair(); err != nil {
		t.Fatalf("Repair: %v", err)
	}

	// A repair is the quick fix for a damaged install, as distinct from a
	// reinstall, which puts the choices back to those of a new install.
	if machine.Applied != machine.Shortcuts {
		t.Errorf("applied = %+v, want the choices already made %+v",
			machine.Applied, machine.Shortcuts)
	}
}

func TestUninstallGoesThroughToTheMachine(t *testing.T) {
	t.Parallel()
	app, machine := fresh(t)

	if err := app.Uninstall(false); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	if !slices.Contains(machine.Calls, "ScheduleDirDeletion") {
		t.Errorf("calls = %v, want the files scheduled to go", machine.Calls)
	}
}

func TestTheFacadeCarriesProgressOutToWhoeverIsListening(t *testing.T) {
	t.Parallel()
	app, _ := fresh(t)
	var heard []setuptest.Step
	app.report = func(pct int, msg string) {
		heard = append(heard, setuptest.Step{Pct: pct, Msg: msg})
	}

	if err := app.Install(OptionsDTO{}); err != nil {
		t.Fatalf("Install: %v", err)
	}

	// The page draws its bar from these and leaves the progress screen on the
	// last one. A facade that swallowed them would leave it there for good.
	if len(heard) == 0 {
		t.Fatal("the install reported nothing")
	}
	if last := heard[len(heard)-1]; last.Pct != setup.PctDone || last.Msg != setup.MsgDone {
		t.Errorf("the run ended at %+v, want %d%% and %q", last, setup.PctDone, setup.MsgDone)
	}
}

func TestTheFacadeRefusesWhileTheApplicationIsOpen(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name string
		act  func(*App) error
	}{
		{"install", func(a *App) error { return a.Install(OptionsDTO{}) }},
		{"repair", (*App).Repair},
		{"uninstall", func(a *App) error { return a.Uninstall(false) }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			app, machine := fresh(t)
			machine.Running = true

			if err := testCase.act(app); !errors.Is(err, setup.ErrAppRunning) {
				t.Errorf("%s with the application open = %v, want ErrAppRunning",
					testCase.name, err)
			}
		})
	}
}

func TestSetShortcutsAppliesStraightAway(t *testing.T) {
	t.Parallel()
	app, machine := fresh(t)

	// There is nothing to install on the manage screen, so a box that waited for
	// a go-ahead would never take effect at all.
	if err := app.SetShortcuts(true, true); err != nil {
		t.Fatalf("SetShortcuts: %v", err)
	}

	want := setup.Shortcuts{StartMenu: true, Desktop: true}
	if machine.Applied != want {
		t.Errorf("applied = %+v, want %+v", machine.Applied, want)
	}
}

func TestSetShortcutsCarriesAMissingInstallDirectoryOut(t *testing.T) {
	t.Parallel()
	app, machine := fresh(t)
	machine.DirErr = errors.New("no such user profile")

	if err := app.SetShortcuts(true, true); !errors.Is(err, machine.DirErr) {
		t.Errorf("SetShortcuts without an install directory = %v, want the reason", err)
	}
}

func TestAppRunningAsksTheMachine(t *testing.T) {
	t.Parallel()
	app, machine := fresh(t)
	machine.Running = true

	if !app.AppRunning() {
		t.Error("AppRunning = false while the machine says it is open")
	}
}

func TestTheLicenceScreenExplainsBeforeItQuotes(t *testing.T) {
	t.Parallel()
	app, _ := fresh(t)

	held := app.Licence()

	// Naming a licence is not explaining one: somebody about to install a program
	// reads "GNU General Public Licence, version 3" and learns nothing from it.
	if len(held.Plain) == 0 {
		t.Error("the licence is named and not explained")
	}
	if !strings.Contains(held.Lead, setup.AppName) || !strings.Contains(held.Lead, licence.Name) {
		t.Errorf("lead = %q, want it to name both the product and the licence", held.Lead)
	}
	if held.Holder != licence.Holder {
		t.Errorf("holder = %q, want %q", held.Holder, licence.Holder)
	}
	// GPL section 4 asks that a copy travel with the program, so the screen shows
	// the text itself rather than a summary of it or a link to it.
	if held.Text != licence.Text() {
		t.Error("the screen shows something other than the published licence")
	}
	if len(held.Notices) == 0 {
		t.Error("setup names none of the works it is built from")
	}
}
