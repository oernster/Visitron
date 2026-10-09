// Package startup is the start-with-Windows entry (FR-052), ported from
// ScreenState: one value under the current user's Run key holding the quoted
// program path plus HiddenFlag, so a sign-in start waits in the tray. The
// setup program writes the same single entry, so the two cannot disagree.
package startup

import (
	"os"
	"strings"
)

// HiddenFlag starts Visitron with its window closed.
const HiddenFlag = "-hidden"

const quote = `"`

// Entry is the Run value for one program.
type Entry struct {
	// Name is the Run value's name.
	Name string
	// Exe is the program the entry starts.
	Exe string
}

// value is what the entry holds: the quoted path plus the flag.
func (e Entry) value() string { return quote + e.Exe + quote + " " + HiddenFlag }

// target reads the program path back out of a Run value.
func target(value string) string {
	if strings.HasPrefix(value, quote) {
		if end := strings.Index(value[1:], quote); end >= 0 {
			return value[1 : end+1]
		}
	}
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// healthy reports whether a stored value would start a real program hidden.
// An entry left by a moved install reads as off, as does one without the flag;
// turning the setting on then rewrites it, which is how a wrong entry heals.
func healthy(value string) bool {
	if !strings.Contains(value, HiddenFlag) {
		return false
	}
	_, err := os.Stat(target(value))
	return err == nil
}
