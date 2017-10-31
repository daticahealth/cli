package jobs

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/services"
)

func CmdStart(jobID string, svcName string, ij IJobs, is services.IServices) error {
	service, err := is.RetrieveByLabel(svcName)
	if err != nil {
		return err
	}
	if service == nil {
		return fmt.Errorf("Could not find a service with the label \"%s\". You can list services with the \"datica services list\" command.", svcName)
	}

	err = ij.Start(jobID, service.ID)

	if err != nil {
		return err
	}
	logrus.Printf("Job '%s' has been successfully started.", jobID)
	return nil
}

func (j *SJobs) Start(jobID string, svcID string) error {

	headers := j.Settings.HTTPManager.GetHeaders(j.Settings.SessionToken, j.Settings.Version, j.Settings.Pod, j.Settings.UsersID)
	resp, statusCode, err := j.Settings.HTTPManager.Post(nil, 
		fmt.Sprintf("%s%s/environments/%s/services/%s/jobs/%s/start", 
			j.Settings.PaasHost, j.Settings.PaasHostVersion, j.Settings.EnvironmentID, svcID, jobID), headers)
	if err != nil {
		return err
	}
	
	return j.Settings.HTTPManager.ConvertResp(resp, statusCode, nil)
	
}