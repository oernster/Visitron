package setup

import (
	"fmt"
	"os"
	"testing"
)

// TestMain points every folder this package resolves at a temporary one
// before any test runs, so no test can reach the profile of the person
// running the suite whatever a later change to a path rule does.
//
// Measured on 2026-10-10: when the data folder moved from APPDATA to the
// store's LOCALAPPDATA rule, a test that redirected APPDATA alone wrote over
// the real %LOCALAPPDATA%\Visitron\visitron.db, then tried to delete its
// folder. A test that redirects one variable is safe only while the code
// reads that one; redirecting both here makes it safe whatever the code reads.
func TestMain(m *testing.M) {
	scratch, err := os.MkdirTemp("", "visitron-setup-tests-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "making the scratch profile:", err)
		os.Exit(1)
	}
	for _, name := range []string{"APPDATA", "LOCALAPPDATA"} {
		if err := os.Setenv(name, scratch); err != nil {
			fmt.Fprintln(os.Stderr, "redirecting", name+":", err)
			os.Exit(1)
		}
	}
	code := m.Run()
	_ = os.RemoveAll(scratch)
	os.Exit(code)
}
