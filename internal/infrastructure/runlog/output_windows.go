//go:build windows

package runlog

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// hasErrorOutput reports whether the run was given an error output. A windowed
// program started from a shortcut reads a handle of 0, so everything it writes
// there is lost, the Go runtime's own crash report included.
func hasErrorOutput() bool {
	handle, err := windows.GetStdHandle(windows.STD_ERROR_HANDLE)
	return err == nil && handle != 0 && handle != windows.InvalidHandle
}

// sendAll points the run's error output at log: first the handle the Go
// runtime looks up for each report it writes, then os.Stderr, which was fixed
// from that handle when the program started.
func sendAll(log *os.File) error {
	if err := windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(log.Fd())); err != nil {
		return fmt.Errorf("sending error output to %s: %w", log.Name(), err)
	}
	os.Stderr = log
	return nil
}
