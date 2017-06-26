package environments

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/config"
	"github.com/daticahealth/cli/models"
)

// CmdList lists all environments which the user has access to
func CmdList(settings *models.Settings, environments IEnvironments) error {
	envs, errs := environments.List()
	if envs == nil || len(*envs) == 0 {
		logrus.Println("no environments found")
	} else {
		for _, env := range *envs {
			logrus.Printf("%s: %s", env.Name, env.ID)
		}
		config.StoreEnvironments(envs, settings)
	}
	if errs != nil && len(errs) > 0 {
		for pod, err := range errs {
			logrus.Debugf("Failed to list environments for pod \"%s\": %s", pod, err)
		}
		logrus.Println("If the environment you're looking for is not listed, ensure you have the correct permissions from your organization owner. If the environment is still not listed, please contact Datica Support at https://datica.com/support.")
	}
	return nil
}

func (e *SEnvironments) List() (*[]models.Environment, map[string]error) {
	allEnvs := []models.Environment{}
	errs := map[string]error{}
	for _, pod := range *e.Settings.Pods {
		headers := e.Settings.HTTPManager.GetHeaders(e.Settings.SessionToken, e.Settings.Version, pod.Name, e.Settings.UsersID)
		resp, statusCode, err := e.Settings.HTTPManager.Get(nil, fmt.Sprintf("%s%s/environments", e.Settings.PaasHost, e.Settings.PaasHostVersion), headers)
		if err != nil {
			errs[pod.Name] = err
			continue
		}
		var envs []models.Environment
		err = e.Settings.HTTPManager.ConvertResp(resp, statusCode, &envs)
		if err != nil {
			errs[pod.Name] = err
			continue
		}
		for i := 0; i < len(envs); i++ {
			envs[i].Pod = pod.Name
		}
		allEnvs = append(allEnvs, envs...)
	}
	return &allEnvs, errs
}

func (e *SEnvironments) Retrieve(envID string) (*models.Environment, error) {
	headers := e.Settings.HTTPManager.GetHeaders(e.Settings.SessionToken, e.Settings.Version, e.Settings.Pod, e.Settings.UsersID)
	resp, statusCode, err := e.Settings.HTTPManager.Get(nil, fmt.Sprintf("%s%s/environments/%s", e.Settings.PaasHost, e.Settings.PaasHostVersion, envID), headers)
	if err != nil {
		return nil, err
	}
	var env models.Environment
	err = e.Settings.HTTPManager.ConvertResp(resp, statusCode, &env)
	if err != nil {
		return nil, err
	}
	env.Pod = e.Settings.Pod
	return &env, nil
}
