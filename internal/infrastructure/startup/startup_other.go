//go:build !windows

package startup

import "errors"

// errWindowsOnly says the entry exists only on Windows, the one platform
// Visitron is built for; the file exists so the package vets everywhere.
var errWindowsOnly = errors.New("start with Windows exists only on Windows")

// Enabled refuses off Windows.
func (Entry) Enabled() (bool, error) { return false, errWindowsOnly }

// SetEnabled refuses off Windows.
func (Entry) SetEnabled(bool) error { return errWindowsOnly }
