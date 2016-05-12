package redeploy

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/catalyzeio/cli/commands/services"
	"github.com/catalyzeio/cli/lib/httpclient"
	"github.com/catalyzeio/cli/models"
)

func CmdRedeploy(svcName string, ir IRedeploy, is services.IServices) error {
	service, err := is.RetrieveByLabel(svcName)
	if err != nil {
		return err
	}
	if service == nil {
		return fmt.Errorf("Could not find a service with the label \"%s\". You can list services with the \"catalyze services\" command.", svcName)
	}
	logrus.Printf("Redeploying %s (ID = %s)", svcName, service.ID)
	err = ir.Redeploy(service)
	if err != nil {
		return err
	}
	logrus.Println("Redeploy successful! Check the status with \"catalyze status\" and your logging dashboard for updates")
	return nil
}

// Redeploy offers a way of deploying a service without having to do a git push
// first. The same version of the currently running service is deployed with
// no changes.
func (r *SRedeploy) Redeploy(service *models.Service) error {
	headers := httpclient.GetHeaders(r.Settings.SessionToken, r.Settings.Version, r.Settings.Pod, r.Settings.UsersID)
	resp, statusCode, err := httpclient.Post(nil, fmt.Sprintf("%s%s/environments/%s/services/%s/deploy?redeploy=true", r.Settings.PaasHost, r.Settings.PaasHostVersion, r.Settings.EnvironmentID, service.ID), headers)
	if err != nil {
		return err
	}
	return httpclient.ConvertResp(resp, statusCode, nil)
}