package db

import (
	"fmt"
	"sort"

	"github.com/catalyzeio/cli/httpclient"
	"github.com/catalyzeio/cli/models"
	"github.com/catalyzeio/cli/services"
)

func CmdList(databaseName string, page, pageSize int, id IDb, is services.IServices) error {
	service, err := is.RetrieveByLabel(databaseName)
	if err != nil {
		return err
	}
	if service == nil {
		return fmt.Errorf("Could not find a service with the label \"%s\"\n", databaseName)
	}
	jobs, err := id.List(page, pageSize, service)
	if err != nil {
		return err
	}
	for _, job := range *jobs {
		fmt.Printf("%s %s (status = %s)\n", job.ID, job.CreatedAt, job.Status)
	}
	if len(*jobs) == pageSize && page == 1 {
		fmt.Println("(for older backups, try with --page 2 or adjust --page-size)")
	}
	if len(*jobs) == 0 && page == 1 {
		fmt.Println("No backups created yet for this service.")
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
	headers := httpclient.GetHeaders(d.Settings.SessionToken, d.Settings.Version, d.Settings.Pod)
	resp, statusCode, err := httpclient.Get(nil, fmt.Sprintf("%s%s/services/%s/brrgc/backup?pageNumber=%d&pageSize=%d", d.Settings.PaasHost, d.Settings.PaasHostVersion, service.ID, page, pageSize), headers)
	if err != nil {
		return nil, err
	}
	var jobsMap map[string]models.Job
	err = httpclient.ConvertResp(resp, statusCode, &jobsMap)
	if err != nil {
		return nil, err
	}
	var jobs []models.Job
	for jobID, job := range jobsMap {
		job.ID = jobID
		jobs = append(jobs, job)
	}
	sort.Sort(SortedJobs(jobs))
	return &jobs, nil
}
