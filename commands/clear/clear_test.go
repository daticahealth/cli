package clear

import (
	"testing"

	"github.com/daticahealth/cli/test"
)

var clearTests = []struct {
	privKey bool
	session bool
	envs    bool
	pods    bool
}{
	{false, false, false, false},
	{false, false, false, true},
	{false, false, true, false},
	{false, true, false, false},
	{true, false, false, false},
	{true, true, true, true},
}

func TestClear(t *testing.T) {
	for _, data := range clearTests {
		t.Logf("Data: %+v", data)
		settings := test.GetSettings("")
		err := CmdClear(data.privKey, data.session, data.envs, data.pods, settings)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
			continue
		}
		if data.privKey && settings.PrivateKeyPath != "" {
			t.Errorf("Private key should have been cleared")
			continue
		}
		if data.session && settings.SessionToken != "" {
			t.Errorf("Session token should have been cleared")
		}
		if data.envs && settings.Environments != nil && len(settings.Environments) != 0 {
			t.Errorf("Environments should have been cleared")
		}
		if data.pods && settings.Pods != nil && len(*settings.Pods) != 0 {
			t.Errorf("Pods should have been cleared")
		}
	}
}
