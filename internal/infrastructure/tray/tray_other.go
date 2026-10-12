//go:build !windows

package tray

import "errors"

// errWindowsOnly says the tray exists only on Windows. The macOS and Linux
// builds have none, so closing the window quits them and they check only
// while open.
var errWindowsOnly = errors.New("the tray exists only on Windows")

// Tray is absent off Windows.
type Tray struct{ commands chan Command }

// New builds an absent tray.
func New() *Tray { return &Tray{commands: make(chan Command)} }

// Commands never delivers.
func (t *Tray) Commands() <-chan Command { return t.commands }

// Start refuses off Windows.
func (t *Tray) Start() error { return errWindowsOnly }

// Stop ends the absent tray.
func (t *Tray) Stop() { close(t.commands) }
