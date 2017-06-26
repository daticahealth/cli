package git

import (
	"fmt"
	"os/exec"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/services"
)

func CmdAdd(svcName, remote string, force bool, ig IGit, is services.IServices) error {
	service, err := is.RetrieveByLabel(svcName)
	if err != nil {
		return err
	}
	if service == nil {
		return fmt.Errorf("Could not find a service with the label \"%s\". You can list services with the \"datica services list\" command.", svcName)
	}
	if service.Source == "" {
		return fmt.Errorf("No git remote found for the \"%s\" service.", svcName)
	}
	remotes, err := ig.List()
	if err != nil {
		return err
	}
	exists := false
	for _, r := range remotes {
		if r == remote {
			exists = true
			break
		}
	}
	if exists && !force {
		return fmt.Errorf("A git remote named \"%s\" already exists, please specify --force to overwrite it", remote)
	} else if exists {
		err = ig.SetURL(remote, service.Source)
		if err != nil {
			return fmt.Errorf("Failed to update existing git remote: %s", err)
		}
	} else {
		err = ig.Add(remote, service.Source)
		if err != nil {
			return fmt.Errorf("Failed to add a git remote: %s", err)
		}
	}
	logrus.Printf("\"%s\" remote added.", remote)
	return nil
}

// Add a git remote to a git repo in the current working directory. If the
// current working directory is not yet a git repo, this command will fail.
func (g *SGit) Add(remote, gitURL string) error {
	_, err := exec.Command("git", "remote", "add", remote, gitURL).Output()
	return err
}

// SetURL updates the remove URL for a given git repo.
func (g *SGit) SetURL(remote, gitURL string) error {
	_, err := exec.Command("git", "remote", "set-url", remote, gitURL).Output()
	return err
}
