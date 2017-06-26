package services

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/lib/volumes"
	"github.com/daticahealth/cli/models"
	"github.com/olekukonko/tablewriter"
)

// CmdServices lists the names of all services for an environment.
func CmdServices(is IServices, v volumes.IVolumes) error {
	svcs, err := is.List()

	if err != nil {
		return err
	}
	if svcs == nil || len(*svcs) == 0 {
		logrus.Println("No services found")
		return nil
	}
	data := [][]string{{"NAME", "DNS", "RAM (GB)", "CPU", "WORKER LIMIT", "SCALE", "STORAGE (GB)"}}
	for _, s := range *svcs {

		vols, err := v.List(s.ID)
		if err != nil {
			logrus.Errorf("Failed to retrieve volume information for service %s", s.Label)
			logrus.Debugf("Volume information error for %s: %s", s.Label, err)
		}
		if vols == nil || len(*vols) == 0 {
			vols = &[]models.Volume{{ID: 0, Type: "", Size: 0}}
		}

		volume := ""
		for i, v := range *vols {
			if i > 0 {
				volume += ", "
			}
			volume += fmt.Sprintf("%d", v.Size)
		}

		data = append(data, []string{s.Label, s.DNS, fmt.Sprintf("%d", s.Size.RAM), fmt.Sprintf("%d", s.Size.CPU), fmt.Sprintf("%d", s.WorkerScale), fmt.Sprintf("%d", s.Scale), volume})

	}

	table := tablewriter.NewWriter(logrus.StandardLogger().Out)
	table.SetBorder(false)
	table.SetRowLine(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.AppendBulk(data)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	table.Render()
	return nil
}

func (s *SServices) List() (*[]models.Service, error) {
	return s.ListByEnvID(s.Settings.EnvironmentID, s.Settings.Pod)
}

func (s *SServices) ListByEnvID(envID, podID string) (*[]models.Service, error) {
	headers := s.Settings.HTTPManager.GetHeaders(s.Settings.SessionToken, s.Settings.Version, podID, s.Settings.UsersID)
	resp, statusCode, err := s.Settings.HTTPManager.Get(nil, fmt.Sprintf("%s%s/environments/%s/services", s.Settings.PaasHost, s.Settings.PaasHostVersion, envID), headers)
	if err != nil {
		return nil, err
	}
	var services []models.Service
	err = s.Settings.HTTPManager.ConvertResp(resp, statusCode, &services)
	if err != nil {
		return nil, err
	}
	return &services, nil
}

func (s *SServices) RetrieveByLabel(label string) (*models.Service, error) {
	services, err := s.List()
	if err != nil {
		return nil, err
	}
	var service *models.Service
	for _, s := range *services {
		if s.Label == label {
			service = &s
			break
		}
	}
	return service, nil
}
