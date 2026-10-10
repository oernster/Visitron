//go:build windows

package setup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// removalDeadline bounds the wait for the scheduled removal. The command it
// spawns waits about two seconds before it starts, so this is generous enough
// that a slow machine does not fail and short enough that a removal which
// never happens is reported rather than waited on.
const removalDeadline = 20 * time.Second

// removalPoll is how often the folder is looked for while waiting.
const removalPoll = 200 * time.Millisecond

// TestSchedulingARemovalActuallyRemovesTheFolder keeps the probe that found
// the defect instead of throwing it away.
//
// The removal is a detached shell command, so nothing about it is checkable by
// reading the code: the first version was passed as an argument, Go escaped it
// the way a C program reads it, cmd.exe read the whole line as one broken
// command and eight seconds later the folder was still there. It now goes in
// verbatim through SysProcAttr.CmdLine. This test is the only thing standing
// between that and a silent return to leaving the install folder behind.
//
// What it cannot cover is the reason the removal is scheduled at all: on a
// real uninstall setup is running from inside the folder and holds its own
// executable open, which no test can reproduce without being that executable.
func TestSchedulingARemovalActuallyRemovesTheFolder(t *testing.T) {
	t.Parallel()
	folder := filepath.Join(t.TempDir(), "Programs", "Visitron")
	if err := os.MkdirAll(filepath.Join(folder, "docs"), dirPerm); err != nil {
		t.Fatalf("making the install folder: %v", err)
	}
	for _, name := range []string{ExeName, filepath.Join("docs", "README.txt")} {
		if err := os.WriteFile(filepath.Join(folder, name), []byte("installed"), dirPerm); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	ScheduleDirDeletion(folder)

	deadline := time.Now().Add(removalDeadline)
	for {
		if _, err := os.Stat(folder); os.IsNotExist(err) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%q was still there %v after the removal was scheduled: "+
				"the command reached cmd.exe in a form it could not read", folder, removalDeadline)
		}
		time.Sleep(removalPoll)
	}
}
