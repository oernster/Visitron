//go:build windows

package startup

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows/registry"
)

const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

// Enabled reports whether a healthy entry is present.
func (e Entry) Enabled() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, fmt.Errorf("opening the Run key: %w", err)
	}
	defer func() { _ = key.Close() }()
	value, _, err := key.GetStringValue(e.Name)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("reading the Run entry: %w", err)
	}
	return healthy(value), nil
}

// SetEnabled writes or removes the entry.
func (e Entry) SetEnabled(on bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("opening the Run key: %w", err)
	}
	defer func() { _ = key.Close() }()
	if !on {
		if err := key.DeleteValue(e.Name); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return fmt.Errorf("removing the Run entry: %w", err)
		}
		return nil
	}
	if err := key.SetStringValue(e.Name, e.value()); err != nil {
		return fmt.Errorf("writing the Run entry: %w", err)
	}
	return nil
}
