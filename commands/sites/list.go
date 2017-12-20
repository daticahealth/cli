package sites

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/models"
	"github.com/olekukonko/tablewriter"
)

func CmdList(is ISites, iservices services.IServices, downStream string) error {
	serviceProxy, err := iservices.RetrieveByLabel(downStream)
	if err != nil {
		return err
	}
	sites, err := is.List(serviceProxy.ID)
	if err != nil {
		return err
	}
	if sites == nil || len(*sites) == 0 {
		logrus.Println("No sites found")
		return nil
	}
	svcs, err := iservices.List()
	svcMap := map[string]string{}
	for _, s := range *svcs {
		svcMap[s.ID] = s.Label
	}

	data := [][]string{{"NAME", "CERT", "UPSTREAM SERVICE"}}
	for _, s := range *sites {
		data = append(data, []string{s.Name, s.Cert, svcMap[s.UpstreamService]})
	}

	table := tablewriter.NewWriter(logrus.StandardLogger().Out)
	table.SetBorder(false)
	table.SetRowLine(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.AppendBulk(data)
	table.Render()
	return nil
}

func (s *SSites) List(svcID string) (*[]models.Site, error) {
	headers := s.Settings.HTTPManager.GetHeaders(s.Settings.SessionToken, s.Settings.Version, s.Settings.Pod, s.Settings.UsersID)
	resp, statusCode, err := s.Settings.HTTPManager.Get(nil, fmt.Sprintf("%s%s/environments/%s/services/%s/sites", s.Settings.PaasHost, s.Settings.PaasHostVersion, s.Settings.EnvironmentID, svcID), headers)
	if err != nil {
		return nil, err
	}
	var sites []models.Site
	err = s.Settings.HTTPManager.ConvertResp(resp, statusCode, &sites)
	if err != nil {
		return nil, err
	}
	return &sites, nil
}
