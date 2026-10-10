//go:build windows

package setup

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"visitron/internal/product"
)

// ErrAppRunning says Visitron is open, so an install or an uninstall that would
// overwrite or delete the running executable must not start. Asking before any
// file is touched is the point: extracting over a locked executable fails part
// way and leaves a half-written install.
//
//lint:ignore ST1005 setup dialog text shown to the user verbatim
var ErrAppRunning = errors.New(product.Name + " is open. Please close it and try again.")

// ErrAppStillRunning says Visitron was asked to close but was still there after
// the wait, so setup stops rather than writing over a locked file.
//
//lint:ignore ST1005 setup dialog text shown to the user verbatim
var ErrAppStillRunning = errors.New(product.Name + " could not be closed. Please close it yourself and try again.")

const (
	// closeTimeout bounds the wait for the executable lock to release, so a
	// stuck process cannot hang setup for ever.
	closeTimeout = 5 * time.Second
	// closePollStep is how often the wait rechecks.
	closePollStep = 100 * time.Millisecond
	// forcedExitCode is reported for a process ended by setup.
	forcedExitCode = 1
)

// IsAppRunning reports whether Visitron is currently running.
func IsAppRunning() bool { return len(processIDs(ExeName)) > 0 }

// processIDs answers the ids of every running process with the given
// executable name, compared without regard to case.
//
// The match is by image name and never by the process tree: a tree kill decides
// descent from recorded parent process ids, which churn on a machine where the
// application has been started and killed repeatedly, so setup can end up
// recorded as a descendant and terminate itself. The window then vanishes with
// no traceback, because a terminate is not a crash.
func processIDs(exeName string) []uint32 {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return nil
	}
	target := strings.ToLower(exeName)
	var found []uint32
	for {
		if strings.ToLower(windows.UTF16ToString(entry.ExeFile[:])) == target {
			found = append(found, entry.ProcessID)
		}
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			return found
		}
	}
}

// CloseRunningApp ends every running instance and waits for the executable lock
// to release.
func CloseRunningApp() error {
	for _, pid := range processIDs(ExeName) {
		terminate(pid)
	}
	deadline := time.Now().Add(closeTimeout)
	for IsAppRunning() {
		if time.Now().After(deadline) {
			return ErrAppStillRunning
		}
		time.Sleep(closePollStep)
	}
	return nil
}

// terminate forcibly ends one process. Every failure is ignored: a process that
// has already exited needs no further action; one this user cannot open is not
// this user's to end.
func terminate(pid uint32) {
	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, pid)
	if err != nil {
		return
	}
	defer windows.CloseHandle(handle)
	_ = windows.TerminateProcess(handle, forcedExitCode)
}

// LaunchApp starts the installed application detached, with the install
// directory as its working directory, so it outlives the setup program.
func LaunchApp() error {
	dir, err := InstallDir()
	if err != nil {
		return err
	}
	cmd := exec.Command(filepath.Join(dir, ExeName))
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch %s: %w", AppName, err)
	}
	return cmd.Process.Release()
}

// ScheduleDirDeletion spawns a detached shell that waits briefly, so this
// process can exit and release its own copy of the executable, then removes the
// install directory. Setup lives inside the directory it is deleting, which is
// why the removal has to outlive it.
//
// The command line is written verbatim through SysProcAttr.CmdLine rather than
// passed as an argument. Go escapes an argument the way a C program reads it,
// with a backslash before each inner quote; cmd.exe does not read that escaping
// and takes the whole line as one broken command. Measured with a probe: the
// argument form removed nothing at all in eight seconds, while the verbatim
// form removed the directory in three.
func ScheduleDirDeletion(dir string) {
	cmd := exec.Command("cmd")
	attributes := hidden()
	attributes.CmdLine = fmt.Sprintf(
		`cmd.exe /C ping 127.0.0.1 -n 3 >nul & rmdir /s /q "%s"`, dir)
	cmd.SysProcAttr = attributes
	_ = cmd.Start()
}
