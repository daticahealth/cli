package db

import (
	"fmt"
	"sort"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/models"
)

func CmdList(databaseName string, page, pageSize int, id IDb, is services.IServices) error {
	service, err := is.RetrieveByLabel(databaseName)
	if err != nil {
		return err
	}
	if service == nil {
		return fmt.Errorf("Could not find a service with the label \"%s\". You can list services with the \"datica services list\" command.", databaseName)
	}
	jobs, err := id.List(page, pageSize, service)
	if err != nil {
		return err
	}
	sort.Sort(SortedJobs(*jobs))
	for _, job := range *jobs {
		logrus.Printf("%s %s (status = %s)", job.ID, job.CreatedAt, job.Status)
	}
	if len(*jobs) == pageSize && page == 1 {
		logrus.Println("(for older backups, try with --page 2 or adjust --page-size)")
	}
	if len(*jobs) == 0 && page == 1 {
		logrus.Println("No backups created yet for this service.")
	} else if len(*jobs) == 0 {
		logrus.Println("No backups found with the given parameters.")
	}
	return nil
}

// SortedJobs is a wrapper for Jobs array in order to sort them by CreatedAt
// for the ListBackups command
type SortedJobs []models.Job

func (jobs SortedJobs) Len() int {
	return len(jobs)
}

func (jobs SortedJobs) Swap(i, j int) {
	jobs[i], jobs[j] = jobs[j], jobs[i]
}

func (jobs SortedJobs) Less(i, j int) bool {
	return jobs[i].CreatedAt < jobs[j].CreatedAt
}

// List lists the created backups for the service sorted from oldest to newest
func (d *SDb) List(page, pageSize int, service *models.Service) (*[]models.Job, error) {
	headers := d.Settings.HTTPManager.GetHeaders(d.Settings.SessionToken, d.Settings.Version, d.Settings.Pod, d.Settings.UsersID)
	resp, statusCode, err := d.Settings.HTTPManager.Get(nil, fmt.Sprintf("%s%s/environments/%s/services/%s/jobs?type=backup&pageNumber=%d&pageSize=%d", d.Settings.PaasHost, d.Settings.PaasHostVersion, d.Settings.EnvironmentID, service.ID, page, pageSize), headers)
	if err != nil {
		return nil, err
	}
	var jobs []models.Job
	err = d.Settings.HTTPManager.ConvertResp(resp, statusCode, &jobs)
	if err != nil {
		return nil, err
	}
	return &jobs, nil
}
