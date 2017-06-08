package redeploy

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/environments"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/lib/jobs"
)

func CmdRedeploy(envID, svcName string, ij jobs.IJobs, is services.IServices, ie environments.IEnvironments) error {
	env, err := ie.Retrieve(envID)
	if err != nil {
		return err
	}
	service, err := is.RetrieveByLabel(svcName)
	if err != nil {
		return err
	}
	if service == nil {
		return fmt.Errorf("Could not find a service with the label \"%s\". You can list services with the \"datica services list\" command.", svcName)
	}
	logrus.Printf("Redeploying service %s (ID = %s) in environment %s (ID = %s)", svcName, service.ID, env.Name, env.ID)
	err = ij.Redeploy(service.ID)
	if err != nil {
		return err
	}
	logrus.Println("Redeploy successful! Check the status with \"datica status\" and your logging dashboard for updates")
	return nil
}
