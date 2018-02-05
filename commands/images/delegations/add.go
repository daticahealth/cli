package delegations

import (
	"fmt"

	"github.com/daticahealth/cli/commands/environments"
	"github.com/daticahealth/cli/lib/images"
	"github.com/daticahealth/cli/lib/prompts"
	"github.com/daticahealth/cli/models"
)

func cmdDelegationsAdd(envID, image, certPath string, user *models.User, ie environments.IEnvironments, ii images.IImages, ip prompts.IPrompts) error {
	env, err := ie.Retrieve(envID)
	if err != nil {
		return err
	}

	_, tag, err := ii.GetGloballyUniqueNamespace(image, env, true)
	if err != nil {
		return err
	}
	if tag != "" {
		return fmt.Errorf("Cannot add signing priveledges for just one tag on an image")
		//TODO: JK, you totally can. --all-paths in notary means you can sign any tag. Should determine how we want to do this
	}

	return nil
}
