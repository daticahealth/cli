package jobs

import (
	"fmt"
	"sort"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/models"
	"github.com/olekukonko/tablewriter"
)

type SortedJobs []models.Job

func (jobs SortedJobs) Len() int {
	return len(jobs)
}

func (jobs SortedJobs) Swap(i, j int) {
	jobs[i], jobs[j] = jobs[j], jobs[i]
}

func (jobs SortedJobs) Less(i, j int) bool {
	return jobs[i].Type < jobs[j].Type
}

func CmdList(svcName string, ij IJobs, is services.IServices) error {
	service, err := is.RetrieveByLabel(svcName)
	if err != nil {
		return err
	}
	if service == nil {
		return fmt.Errorf("Could not find a service with the label \"%s\". You can list services with the \"datica services list\" command.", svcName)
	}

	jbs, err := ij.List(service.ID)
	if err != nil {
		return err
	}

	if jbs == nil || len(*jbs) == 0 {
		logrus.Println("No releases found")
		return nil
	}

	sort.Sort(SortedJobs(*jbs))
	const dateForm = "2006-01-02T15:04:05"
	data := [][]string{{"Job Id", "Status", "Created At", "Type", "Target"}}
	for _, j := range *jbs {
		id := j.ID

		t, _ := time.Parse(dateForm, j.CreatedAt)
		data = append(data, []string{id, j.Status, t.Local().Format(time.Stamp), j.Type, j.Target})
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

func (j *SJobs) List(svcID string) (*[]models.Job, error) {
	headers := j.Settings.HTTPManager.GetHeaders(j.Settings.SessionToken, j.Settings.Version, j.Settings.Pod, j.Settings.UsersID)
	resp, statusCode, err := j.Settings.HTTPManager.Get(nil,
		fmt.Sprintf("%s%s/environments/%s/services/%s/jobs",
			j.Settings.PaasHost, j.Settings.PaasHostVersion, j.Settings.EnvironmentID, svcID), headers)
	if err != nil {
		return nil, err
	}
	var jbs []models.Job
	err = j.Settings.HTTPManager.ConvertResp(resp, statusCode, &jbs)
	if err != nil {
		return nil, err
	}
	return &jbs, nil
}
