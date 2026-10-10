package setup_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"visitron/internal/infrastructure/setup"
	"visitron/internal/infrastructure/setup/setuptest"
)

// assertOrder states that the named calls happened, in this order, allowing
// other calls between them. The sequence is the thing under test; a call that
// is merely present says nothing about when it ran.
func assertOrder(t *testing.T, calls []string, want ...string) {
	t.Helper()
	at := 0
	for _, name := range want {
		found := slices.Index(calls[at:], name)
		if found < 0 {
			t.Fatalf("calls = %v, want %q after position %d", calls, name, at)
		}
		at += found + 1
	}
}

func TestInstallRefusesWhileTheApplicationIsOpen(t *testing.T) {
	t.Parallel()
	machine := setuptest.Ready()
	machine.Running = true

	err := setup.Install(machine, machine.Report, nil, "1.0.0", setup.Shortcuts{})

	if !errors.Is(err, setup.ErrAppRunning) {
		t.Errorf("Install with the application open = %v, want setup.ErrAppRunning", err)
	}
	// Nothing may be written before that answer: a half-written install over a
	// locked executable is worse than a refusal the reader can act on.
	if slices.Contains(machine.Calls, "ExtractZip") {
		t.Errorf("calls = %v, want nothing written", machine.Calls)
	}
	if len(machine.Steps) != 0 {
		t.Errorf("steps = %v, want no progress reported for work not started", machine.Steps)
	}
}

func TestInstallWritesThenRegistersThenAppliesTheChoices(t *testing.T) {
	t.Parallel()
	machine := setuptest.Ready()
	want := setup.Shortcuts{StartMenu: true, Desktop: false}

	if err := setup.Install(machine, machine.Report, []byte("payload"), "1.0.0", want); err != nil {
		t.Fatalf("Install: %v", err)
	}

	assertOrder(t, machine.Calls,
		"AppRunning", "InstallDir", "ExtractZip",
		"Executable", "CopyFile", "WriteUninstallEntry", "ApplyShortcuts")
	if machine.Applied != want {
		t.Errorf("applied = %+v, want %+v", machine.Applied, want)
	}
}

func TestInstallCarriesTheInstallDirectoryFailureOut(t *testing.T) {
	t.Parallel()
	machine := setuptest.Ready()
	machine.DirErr = errors.New("no such user profile")

	if err := setup.Install(machine, machine.Report, nil, "1.0.0", setup.Shortcuts{}); !errors.Is(err, machine.DirErr) {
		t.Errorf("Install without an install directory = %v, want the reason", err)
	}
	if slices.Contains(machine.Calls, "ExtractZip") {
		t.Errorf("calls = %v, want nothing written", machine.Calls)
	}
}

func TestInstallSaysWhichStepFailed(t *testing.T) {
	t.Parallel()
	// A refusal names what was refused (house robustness rule 10): "setup failed" sends the
	// reader nowhere, while the step that failed sends them somewhere.
	reason := errors.New("the disk is full")
	for _, testCase := range []struct {
		name   string
		break_ func(*setuptest.Machine)
		says   string
	}{
		{"the payload", func(m *setuptest.Machine) { m.Extract = reason }, "write the files"},
		{"setup's own path", func(m *setuptest.Machine) { m.SelfErr = reason }, "locate setup"},
		{"the uninstaller copy", func(m *setuptest.Machine) { m.Copy = reason }, "write the uninstaller"},
		{"the Apps list entry", func(m *setuptest.Machine) { m.Entry = reason }, "register " + setup.AppName},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			machine := setuptest.Ready()
			testCase.break_(machine)

			err := setup.Install(machine, machine.Report, nil, "1.0.0", setup.Shortcuts{})

			if !errors.Is(err, reason) {
				t.Errorf("Install = %v, want the underlying reason wrapped", err)
			}
			if !strings.Contains(err.Error(), testCase.says) {
				t.Errorf("Install = %q, want it to name %q", err, testCase.says)
			}
			if slices.Contains(machine.Calls, "ApplyShortcuts") {
				t.Errorf("calls = %v, want it to stop at the failure", machine.Calls)
			}
		})
	}
}

func TestRemoveRefusesWhileTheApplicationIsOpen(t *testing.T) {
	t.Parallel()
	machine := setuptest.Ready()
	machine.Running = true

	err := setup.Remove(machine, machine.Report, true)

	if !errors.Is(err, setup.ErrAppRunning) {
		t.Errorf("Remove with the application open = %v, want setup.ErrAppRunning", err)
	}
	if slices.Contains(machine.Calls, "RemoveShortcuts") {
		t.Errorf("calls = %v, want nothing removed", machine.Calls)
	}
}

func TestRemoveTakesTheShortcutsBeforeTheRegistryEntry(t *testing.T) {
	t.Parallel()
	machine := setuptest.Ready()

	if err := setup.Remove(machine, machine.Report, false); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// The registry entry is what the Apps list reads. Were it taken first and the
	// run then interrupted, the program would be gone from the list while its
	// shortcuts still sat on the desktop, pointing at files about to vanish.
	assertOrder(t, machine.Calls,
		"AppRunning", "InstallDir", "RemoveShortcuts", "RemoveUninstallEntry",
		"Leftovers", "ScheduleDirDeletion")
}

func TestRemoveKeepsTheRecordUnlessAsked(t *testing.T) {
	t.Parallel()
	machine := setuptest.Ready()

	if err := setup.Remove(machine, machine.Report, false); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if slices.Contains(machine.Calls, "RecordDir") {
		t.Errorf("calls = %v, want the record left alone", machine.Calls)
	}
	if slices.Contains(machine.Removed, machine.Record) {
		t.Errorf("removed = %v, want the record kept", machine.Removed)
	}
}

func TestRemoveTakesTheRecordLastAndOnlyWhenAsked(t *testing.T) {
	t.Parallel()
	machine := setuptest.Ready()

	if err := setup.Remove(machine, machine.Report, true); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Everything before the record can be put back by installing again. The
	// record cannot, so it goes after the steps that are reversible.
	assertOrder(t, machine.Calls,
		"RemoveShortcuts", "RemoveUninstallEntry", "Leftovers", "RecordDir", "ScheduleDirDeletion")
	if !slices.Contains(machine.Removed, machine.Record) {
		t.Errorf("removed = %v, want the record among them", machine.Removed)
	}
}

func TestRemoveLeavesTheRecordWhereItsFolderCannotBeFound(t *testing.T) {
	t.Parallel()
	machine := setuptest.Ready()
	machine.RecordErr = errors.New("no such folder")

	if err := setup.Remove(machine, machine.Report, true); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if slices.Contains(machine.Removed, machine.Record) {
		t.Errorf("removed = %v, want nothing removed for a folder that was not found", machine.Removed)
	}
	if !slices.Contains(machine.Calls, "ScheduleDirDeletion") {
		t.Errorf("calls = %v, want the removal to carry on", machine.Calls)
	}
}

func TestRemoveClearsEveryLeftoverAndCarriesOnThroughAFailure(t *testing.T) {
	t.Parallel()
	machine := setuptest.Ready()
	machine.RemoveErr = errors.New("in use by another process")

	if err := setup.Remove(machine, machine.Report, false); err != nil {
		t.Errorf("Remove = %v, want a leftover that will not go to stop nothing", err)
	}

	// A folder that will not go is not a reason to leave the program installed:
	// the entry in the Apps list is what the reader is trying to be rid of.
	for _, folder := range machine.Leftover {
		if !slices.Contains(machine.Removed, folder) {
			t.Errorf("removed = %v, want %q among them", machine.Removed, folder)
		}
	}
	if !slices.Contains(machine.Calls, "ScheduleDirDeletion") {
		t.Errorf("calls = %v, want the files still scheduled to go", machine.Calls)
	}
}

func TestEveryRunEndsAtOneHundredAndNeverGoesBackwards(t *testing.T) {
	t.Parallel()
	// The page draws the bar straight from these, so a step that went backwards
	// would be seen. The last one is what takes the screen off progress.
	for _, testCase := range []struct {
		name string
		run  func(*setuptest.Machine) error
	}{
		{"install", func(m *setuptest.Machine) error {
			return setup.Install(m, m.Report, nil, "1.0.0", setup.Shortcuts{})
		}},
		{"remove", func(m *setuptest.Machine) error { return setup.Remove(m, m.Report, true) }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			machine := setuptest.Ready()
			if err := testCase.run(machine); err != nil {
				t.Fatalf("%s: %v", testCase.name, err)
			}
			if len(machine.Steps) == 0 {
				t.Fatal("no progress was reported at all")
			}
			for i, reported := range machine.Steps {
				if i > 0 && reported.Pct < machine.Steps[i-1].Pct {
					t.Errorf("step %d went back to %d%% from %d%%",
						i, reported.Pct, machine.Steps[i-1].Pct)
				}
				if reported.Msg == "" {
					t.Errorf("step %d at %d%% says nothing", i, reported.Pct)
				}
			}
			last := machine.Steps[len(machine.Steps)-1]
			if last.Pct != setup.PctDone || last.Msg != setup.MsgDone {
				t.Errorf("the run ended at %+v, want %d%% and %q", last, setup.PctDone, setup.MsgDone)
			}
		})
	}
}

func TestTheRealMachineSatisfiesTheSeam(t *testing.T) {
	t.Parallel()
	// Real is what the setup program runs against; nothing else here exercises
	// it, since every method reaches the computer it is running on.
	var _ setup.Machine = setup.Real{}
}
