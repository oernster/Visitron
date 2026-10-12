//go:build !windows

package startup

import "visitron/internal/application"

// Enabled answers that this platform has no sign-in start, so Settings offers
// none; the macOS and Linux builds start when they are opened.
func (Entry) Enabled() (bool, error) { return false, application.ErrNoStartup }

// SetEnabled refuses: there is no sign-in start to turn on.
func (Entry) SetEnabled(bool) error { return application.ErrNoStartup }
