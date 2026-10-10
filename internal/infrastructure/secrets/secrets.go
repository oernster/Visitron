// Package secrets keeps the GoatCounter key and the GitHub token in Windows
// Credential Manager through zalando/go-keyring, as PigeonPost keeps its
// passwords (C-4). Neither ever reaches a file or the log (NFR-SEC-001).
package secrets

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"

	"visitron/internal/application"
	"visitron/internal/product"
)

// Vault reads and writes the two secrets under the product's service name.
type Vault struct{}

// Get answers the secret, "" when it is not set.
func (Vault) Get(name application.Secret) (string, error) {
	value, err := keyring.Get(product.Name, string(name))
	if errors.Is(err, keyring.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("reading %s from Credential Manager: %w", name, err)
	}
	return value, nil
}

// Set stores or replaces the secret.
func (Vault) Set(name application.Secret, value string) error {
	if err := keyring.Set(product.Name, string(name), value); err != nil {
		return fmt.Errorf("storing %s in Credential Manager: %w", name, err)
	}
	return nil
}

// Delete forgets the secret; one that was never set is already forgotten.
func (Vault) Delete(name application.Secret) error {
	err := keyring.Delete(product.Name, string(name))
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("removing %s from Credential Manager: %w", name, err)
	}
	return nil
}
