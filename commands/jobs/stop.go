package jobs

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/lib/prompts"
)

func CmdStop(jobID string, svcName string, ij IJobs, is services.IServices, force bool, ip prompts.IPrompts) error {
	if !force {
		err := ip.YesNo(fmt.Sprintf("Stopping %s %s will immediately stop this job.", svcName, jobID), fmt.Sprintf("Are you sure you want to stop %s? (y/n) ", svcName))
		if err != nil {
			return err
		}
	}

	service, err := is.RetrieveByLabel(svcName)
	if err != nil {
		return err
	}
	if service == nil {
		return fmt.Errorf("Could not find a service with the label \"%s\". You can list services with the \"datica services list\" command.", svcName)
	}
	err = ij.Stop(jobID, service.ID)
	if err != nil {
		return err
	}
	logrus.Printf("Job '%s' will be stopped in 15 seconds.", jobID)
	return nil
}

func (j *SJobs) Stop(jobID string, svcID string) error {

	headers := j.Settings.HTTPManager.GetHeaders(j.Settings.SessionToken, j.Settings.Version, j.Settings.Pod, j.Settings.UsersID)
	resp, statusCode, err := j.Settings.HTTPManager.Post(nil,
		fmt.Sprintf("%s%s/environments/%s/services/%s/jobs/%s/stop",
			j.Settings.PaasHost, j.Settings.PaasHostVersion, j.Settings.EnvironmentID, svcID, jobID), headers)
	if err != nil {
		return err
	}

	return j.Settings.HTTPManager.ConvertResp(resp, statusCode, nil)

}
