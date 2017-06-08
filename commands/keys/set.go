package keys

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/lib/auth"
	"github.com/daticahealth/cli/lib/prompts"
	"github.com/daticahealth/cli/models"
	"github.com/mitchellh/go-homedir"
)

func CmdSet(path string, settings *models.Settings) error {
	homePath, err := homedir.Expand(path)
	if err != nil {
		return err
	}
	fullPath, err := filepath.Abs(homePath)
	if err != nil {
		return err
	}

	// make sure both files exist
	_, err = os.Stat(fullPath + ".pub")
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Public key file '%s' does not exist.", fullPath+".pub")
		}
		return err
	}

	_, err = os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Private key file '%s' does not exist.", fullPath)
		}
		return err
	}

	settings.PrivateKeyPath = fullPath
	settings.SessionToken = ""
	a := auth.New(settings, prompts.New())
	user, err := a.Signin()
	if err != nil {
		return err
	}
	logrus.Printf("Successfully added key and signed in as %s.", user.Email)
	return nil
}
