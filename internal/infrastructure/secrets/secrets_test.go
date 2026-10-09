package secrets

import (
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/oernster/visitron/internal/application"
)

var _ application.Secrets = Vault{}

// The library's in-memory provider stands in for Credential Manager, so the
// suite never writes the owner's real key.
func TestRoundTrip(t *testing.T) {
	keyring.MockInit()
	v := Vault{}
	if got, err := v.Get(application.GoatCounterKey); got != "" || err != nil {
		t.Errorf("unset: %q %v", got, err)
	}
	if err := v.Delete(application.GoatCounterKey); err != nil {
		t.Errorf("deleting an unset key: %v", err)
	}
	if err := v.Set(application.GoatCounterKey, "k"); err != nil {
		t.Fatal(err)
	}
	if got, _ := v.Get(application.GoatCounterKey); got != "k" {
		t.Errorf("got %q", got)
	}
	if got, _ := v.Get(application.GitHubToken); got != "" {
		t.Errorf("the token read the key: %q", got)
	}
	_ = v.Delete(application.GoatCounterKey)
	if got, _ := v.Get(application.GoatCounterKey); got != "" {
		t.Errorf("after delete: %q", got)
	}
}

func TestFaultsAreReported(t *testing.T) {
	why := errors.New("locked")
	keyring.MockInitWithError(why)
	v := Vault{}
	if _, err := v.Get(application.GitHubToken); !errors.Is(err, why) {
		t.Errorf("get: %v", err)
	}
	if err := v.Set(application.GitHubToken, "t"); !errors.Is(err, why) {
		t.Errorf("set: %v", err)
	}
	if err := v.Delete(application.GitHubToken); !errors.Is(err, why) {
		t.Errorf("delete: %v", err)
	}
}
