package db

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/lib/jobs"
	"github.com/daticahealth/cli/models"
)

func CmdLogs(databaseName, backupID string, id IDb, is services.IServices, ij jobs.IJobs) error {
	service, err := is.RetrieveByLabel(databaseName)
	if err != nil {
		return err
	}
	if service == nil {
		return fmt.Errorf("Could not find a service with the label \"%s\". You can list services with the \"datica services list\" command.", databaseName)
	}
	job, err := ij.Retrieve(backupID, service.ID, false)
	if err != nil {
		return err
	}
	return id.DumpLogs(job.Type, job, service)
}

// DumpLogs dumps logs from a Backup/Restore/Import/Export job to the console
func (d *SDb) DumpLogs(taskType string, job *models.Job, service *models.Service) error {
	logrus.Printf("Retrieving %s logs for job %s...", service.Label, job.ID)
	tempURL, err := d.TempLogsURL(job.ID, service.ID)
	if err != nil {
		return err
	}
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}

	encrFile, err := ioutil.TempFile(dir, "")
	if err != nil {
		return err
	}
	resp, err := http.Get(tempURL.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(encrFile, resp.Body)
	encrFile.Close()

	plainFile, err := ioutil.TempFile(dir, "")
	if err != nil {
		return err
	}
	// do we have to close this before calling DecryptFile?
	// or can two processes have a file open simultaneously?
	plainFile.Close()

	if taskType == "backup" {
		logsKey := job.Backup.KeyLogs
		if logsKey == "" {
			logsKey = job.Backup.Key
		}
		err := d.Crypto.DecryptFile(encrFile.Name(), logsKey, job.Backup.IV, plainFile.Name())
		if err != nil {
			return err
		}
	} else if taskType == "restore" {
		logsKey := job.Restore.KeyLogs
		if logsKey == "" {
			logsKey = job.Restore.Key
		}
		err := d.Crypto.DecryptFile(encrFile.Name(), logsKey, job.Restore.IV, plainFile.Name())
		if err != nil {
			return err
		}
	}
	logrus.Printf("-------------------------- Begin %s logs --------------------------", service.Label)
	plainFile, _ = os.Open(plainFile.Name())
	io.Copy(os.Stdout, plainFile)
	plainFile.Close()
	logrus.Printf("--------------------------- End %s logs ---------------------------", service.Label)
	os.Remove(encrFile.Name())
	os.Remove(plainFile.Name())
	os.Remove(dir)
	return nil
}

func (d *SDb) TempLogsURL(jobID string, serviceID string) (*models.TempURL, error) {
	headers := d.Settings.HTTPManager.GetHeaders(d.Settings.SessionToken, d.Settings.Version, d.Settings.Pod, d.Settings.UsersID)
	resp, statusCode, err := d.Settings.HTTPManager.Get(nil, fmt.Sprintf("%s%s/environments/%s/services/%s/backup-restore-logs-url/%s", d.Settings.PaasHost, d.Settings.PaasHostVersion, d.Settings.EnvironmentID, serviceID, jobID), headers)
	if err != nil {
		return nil, err
	}
	var tempURL models.TempURL
	err = d.Settings.HTTPManager.ConvertResp(resp, statusCode, &tempURL)
	if err != nil {
		return nil, err
	}
	return &tempURL, nil
}
