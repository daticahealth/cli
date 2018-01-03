package sites

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/models"
)

func CmdRm(name string, is ISites, iservices services.IServices, downStream string) error {
	serviceProxy, err := iservices.RetrieveByLabel(downStream)
	if err != nil {
		return err
	}
	sites, err := is.List(serviceProxy.ID)
	if err != nil {
		return err
	}
	var site *models.Site
	for _, s := range *sites {
		if s.Name == name {
			site = &s
			break
		}
	}
	if site == nil {
		return fmt.Errorf("Could not find a site with the label \"%s\". You can list sites with the \"datica sites list\" command.", name)
	}
	err = is.Rm(site.ID, serviceProxy.ID)
	if err != nil {
		return err
	}
	logrus.Println("Site removed")
	logrus.Println("To make your changes go live, you must redeploy your service proxy with the \"datica redeploy service_proxy\" command")
	return nil
}

func (s *SSites) Rm(siteID int, svcID string) error {
	headers := s.Settings.HTTPManager.GetHeaders(s.Settings.SessionToken, s.Settings.Version, s.Settings.Pod, s.Settings.UsersID)
	resp, statusCode, err := s.Settings.HTTPManager.Delete(nil, fmt.Sprintf("%s%s/environments/%s/services/%s/sites/%d", s.Settings.PaasHost, s.Settings.PaasHostVersion, s.Settings.EnvironmentID, svcID, siteID), headers)
	if err != nil {
		return err
	}
	return s.Settings.HTTPManager.ConvertResp(resp, statusCode, nil)
}
