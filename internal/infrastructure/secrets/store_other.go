//go:build !windows && !darwin

package secrets

// storeName is where go-keyring keeps the secrets on this platform: the
// Secret Service, which the desktop shows as its keyring.
const storeName = "the system keyring"
