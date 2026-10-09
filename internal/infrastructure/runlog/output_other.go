//go:build !windows

package runlog

import (
	"errors"
	"os"
)

// hasErrorOutput answers yes off Windows, so Keep copies the crash report to
// the log and leaves the error output where it is. The failure this guards
// against, a windowed run handed a handle of 0, was measured on Windows and has
// no counterpart here: a Linux or macOS build started from a launcher keeps a
// working standard error.
func hasErrorOutput() bool { return true }

// sendAll is not built for this platform, so it says so rather than doing
// nothing quietly. Nothing calls it: hasErrorOutput answers yes above; Keep
// asks for it only when that is false.
func sendAll(*os.File) error {
	return errors.New("sending error output to a file is not built for this platform")
}
